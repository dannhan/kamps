package kulon2

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ruyam/kamps/internal/session"
	"github.com/ruyam/kamps/internal/source"
)

// LiveSource fetches the enrolled course list from Kulon2. The first live
// milestone intentionally does not claim assignments or materials are loaded.
type LiveSource struct {
	siteURL       string
	session       session.Session
	minRequestGap time.Duration
	client        *http.Client
	loadMu        sync.Mutex
	mu            sync.Mutex
	lastRequest   time.Time
}

// NewSource loads a saved session and binds it to the currently configured
// site. A session captured for a different site is refused.
func NewSource(siteURL, sessionPath string) (*LiveSource, error) {
	saved, err := session.Load(sessionPath)
	if err != nil {
		if errors.Is(err, session.ErrNotFound) {
			return nil, errors.New("no saved Kulon2 session; run `kamps login` first")
		}
		return nil, fmt.Errorf("load Kulon2 session: %w", err)
	}
	if !sameSite(siteURL, saved.SiteURL) {
		return nil, errors.New("saved session belongs to a different site; run `kamps login` again")
	}
	return &LiveSource{
		siteURL:       siteURL,
		session:       saved,
		minRequestGap: time.Second,
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) > 0 && req.URL.Host != via[0].URL.Host {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
	}, nil
}

func (s *LiveSource) Label() string { return "Kulon2 (course list only)" }

func (s *LiveSource) Load(ctx context.Context) (source.Snapshot, error) {
	s.loadMu.Lock()
	defer s.loadMu.Unlock()

	identity, err := verifySessionWithClient(ctx, s.siteURL, s.session.Cookie, s.client)
	if err == nil || requestMayHaveStarted(err) {
		s.noteRequest()
	}
	if errors.Is(err, ErrSessionExpired) {
		return source.Snapshot{}, errors.New("Kulon2 session expired; run `kamps login` again")
	}
	if err != nil {
		return source.Snapshot{}, fmt.Errorf("verify Kulon2 session: %w", err)
	}
	if err := s.waitForRequestGap(ctx); err != nil {
		return source.Snapshot{}, err
	}
	courses, err := listCoursesWithClient(ctx, s.siteURL, s.session.Cookie, identity.SessKey, s.client)
	if err == nil || requestMayHaveStarted(err) {
		s.noteRequest()
	}
	if errors.Is(err, ErrSessionExpired) {
		return source.Snapshot{}, errors.New("Kulon2 session expired; run `kamps login` again")
	}
	if err != nil {
		return source.Snapshot{}, err
	}

	snapshot := source.Snapshot{Courses: courses}
	for _, course := range courses {
		if err := s.waitForRequestGap(ctx); err != nil {
			return source.Snapshot{}, err
		}
		pageURL := strings.TrimRight(s.siteURL, "/") + "/course/view.php?id=" + strconv.Itoa(course.ID)
		page, fetchErr := getPage(ctx, pageURL, s.session.Cookie, s.client)
		if fetchErr == nil || requestMayHaveStarted(fetchErr) {
			s.noteRequest()
		}
		if errors.Is(fetchErr, ErrSessionExpired) {
			return source.Snapshot{}, errors.New("Kulon2 session expired; run `kamps login` again")
		}
		if fetchErr != nil {
			return source.Snapshot{}, fmt.Errorf("fetch course %d: %w", course.ID, fetchErr)
		}
		_, modules, parseErr := ParseCoursePage(page)
		if parseErr != nil {
			return source.Snapshot{}, fmt.Errorf("parse course %d (%s): %w", course.ID, course.Name, parseErr)
		}
		for _, module := range modules {
			assignment, material := MapModule(course, module)
			if assignment != nil {
				snapshot.Assignments = append(snapshot.Assignments, *assignment)
			}
			if material != nil {
				snapshot.Materials = append(snapshot.Materials, *material)
			}
		}
	}
	snapshot.Notice = "course pages loaded; assignment due dates, assignment attachments, and downloads are not loaded yet"
	return snapshot, nil
}

func (s *LiveSource) waitForRequestGap(ctx context.Context) error {
	s.mu.Lock()
	lastRequest := s.lastRequest
	s.mu.Unlock()
	if s.minRequestGap <= 0 || lastRequest.IsZero() {
		return nil
	}
	wait := time.Until(lastRequest.Add(s.minRequestGap))
	if wait <= 0 {
		return nil
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func requestMayHaveStarted(err error) bool {
	var networkError *url.Error
	if !errors.As(err, &networkError) {
		return true
	}
	return !errors.Is(networkError.Err, context.Canceled) && !errors.Is(networkError.Err, context.DeadlineExceeded)
}

func (s *LiveSource) noteRequest() {
	s.mu.Lock()
	s.lastRequest = time.Now()
	s.mu.Unlock()
}

func sameSite(a, b string) bool {
	normalize := func(value string) string {
		return strings.TrimRight(strings.TrimSpace(value), "/")
	}
	return normalize(a) != "" && normalize(a) == normalize(b)
}

var _ source.Source = (*LiveSource)(nil)
