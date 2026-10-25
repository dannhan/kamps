package kulon2

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/session"
	"github.com/ruyam/kamps/internal/source"
)

const defaultMaxConcurrent = 4

// LiveSource fetches enrolled courses and their supported coursework from Kulon2.
type LiveSource struct {
	siteURL       string
	session       session.Session
	minRequestGap time.Duration
	maxConcurrent int
	client        *http.Client
	downloadSlots chan struct{}
	loadMu        sync.Mutex
	requestMu     sync.Mutex
	lastRequest   time.Time
}

// NewSource loads a saved session and binds it to the currently configured site.
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
		siteURL: siteURL, session: saved, minRequestGap: 100 * time.Millisecond, maxConcurrent: defaultMaxConcurrent,
		downloadSlots: make(chan struct{}, defaultMaxConcurrent),
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

func (s *LiveSource) Label() string { return "Kulon2" }

// Capabilities reports that live Kulon2 mode supports the full surface.
func (s *LiveSource) Capabilities() source.Capabilities {
	return source.Capabilities{Assignments: true, Materials: true, Downloads: true, Session: true}
}

// SessionStatus reports the best-known expiry of the saved session.
func (s *LiveSource) SessionStatus() source.SessionStatus {
	expiry := s.session.Expiry()
	return source.SessionStatus{ExpiresAt: expiry, OK: !expiry.IsZero()}
}

func (s *LiveSource) Load(ctx context.Context) (source.Snapshot, error) {
	s.loadMu.Lock()
	defer s.loadMu.Unlock()

	sesskey := s.session.SessKey
	var courses []model.Course
	var err error
	if sesskey != "" {
		courses, err = s.listCourses(ctx, sesskey)
		if errors.Is(err, ErrInvalidSesskey) {
			sesskey = ""
		} else if err != nil {
			return source.Snapshot{}, err
		}
	}
	if sesskey == "" {
		identity, verifyErr := s.verifySession(ctx)
		if verifyErr != nil {
			if errors.Is(verifyErr, ErrSessionExpired) {
				return source.Snapshot{}, errors.New("Kulon2 session expired; run `kamps login` again")
			}
			return source.Snapshot{}, fmt.Errorf("verify Kulon2 session: %w", verifyErr)
		}
		sesskey = identity.SessKey
		courses, err = s.listCourses(ctx, sesskey)
	}
	if errors.Is(err, ErrSessionExpired) {
		return source.Snapshot{}, errors.New("Kulon2 session expired; run `kamps login` again")
	}
	if err != nil {
		return source.Snapshot{}, err
	}

	snapshot := source.Snapshot{Courses: courses}
	coursePages := make([][]byte, len(courses))
	courseErrors := make([]string, len(courses))
	s.fetchConcurrent(ctx, len(courses), func(index int) {
		course := courses[index]
		pageURL := strings.TrimRight(s.siteURL, "/") + "/course/view.php?id=" + strconv.Itoa(course.ID)
		page, fetchErr := s.getPage(ctx, pageURL)
		if fetchErr != nil {
			courseErrors[index] = fmt.Sprintf("%s: %v", course.Name, fetchErr)
			return
		}
		coursePages[index] = page
	})

	modulesByCourse := make([][]Module, len(courses))
	for i, page := range coursePages {
		if courseErrors[i] != "" || page == nil {
			continue
		}
		_, modules, parseErr := ParseCoursePage(page)
		if parseErr != nil {
			courseErrors[i] = fmt.Sprintf("%s (parse): %v", courses[i].Name, parseErr)
			continue
		}
		modulesByCourse[i] = modules
	}
	for _, failure := range courseErrors {
		if strings.Contains(failure, ErrSessionExpired.Error()) {
			return source.Snapshot{}, errors.New("Kulon2 session expired; run `kamps login` again")
		}
		if failure != "" {
			snapshot.Notice = appendNotice(snapshot.Notice, "some course pages failed: "+failure)
		}
	}

	type assignmentTask struct {
		course model.Course
		module Module
	}
	tasks := make([]assignmentTask, 0)
	folderTasks := make([]assignmentTask, 0)
	for courseIndex, modules := range modulesByCourse {
		course := courses[courseIndex]
		for _, module := range modules {
			assignment, material := MapModule(course, module, s.siteURL)
			if assignment != nil {
				snapshot.Assignments = append(snapshot.Assignments, *assignment)
			}
			if material != nil {
				snapshot.Materials = append(snapshot.Materials, *material)
			}
			if assignment != nil && (module.Type == "assign" || module.Type == "quiz") {
				tasks = append(tasks, assignmentTask{course: course, module: module})
			}
			if module.Type == "folder" {
				folderTasks = append(folderTasks, assignmentTask{course: course, module: module})
			}
		}
	}
	assignmentPages := make([][]byte, len(tasks))
	assignmentErrors := make([]string, len(tasks))
	s.fetchConcurrent(ctx, len(tasks), func(index int) {
		task := tasks[index]
		pageURL := strings.TrimRight(s.siteURL, "/") + "/mod/" + task.module.Type + "/view.php?id=" + task.module.CMID
		page, fetchErr := s.getPage(ctx, pageURL)
		if fetchErr != nil {
			assignmentErrors[index] = fmt.Sprintf("%s: %v", task.module.Name, fetchErr)
			return
		}
		assignmentPages[index] = page
	})
	for _, failure := range assignmentErrors {
		if strings.Contains(failure, ErrSessionExpired.Error()) {
			return source.Snapshot{}, errors.New("Kulon2 session expired; run `kamps login` again")
		}
	}
	assignmentIndex := 0
	for courseIndex, modules := range modulesByCourse {
		for _, module := range modules {
			assignment, _ := MapModule(courses[courseIndex], module, s.siteURL)
			if assignment == nil || (module.Type != "assign" && module.Type != "quiz") {
				continue
			}
			page := assignmentPages[assignmentIndex]
			failure := assignmentErrors[assignmentIndex]
			if failure != "" {
				snapshot.Notice = appendNotice(snapshot.Notice, "some assignment details failed: "+failure)
			}
			if page != nil {
				parsedPage, parseErr := parseActivityPageForModule(module.Type, page, module.CMID)
				if parseErr != nil {
					snapshot.Notice = appendNotice(snapshot.Notice, "assignment detail parse failed: "+module.Name)
				} else {
					for i := range snapshot.Assignments {
						if snapshot.Assignments[i].ID != assignment.ID {
							continue
						}
						snapshot.Assignments[i].Attachments = len(parsedPage.Attachments)
						switch parsedPage.LMSStatus {
						case "submitted":
							snapshot.Assignments[i].Submission = model.SubmissionSubmitted
						case "not_submitted":
							snapshot.Assignments[i].Submission = model.SubmissionNotSubmitted
						}
						if parsedPage.DueISO != "" {
							due, dateErr := time.ParseInLocation("2006-01-02T15:04:05", parsedPage.DueISO, time.Local)
							if dateErr != nil {
								snapshot.Notice = appendNotice(snapshot.Notice, "assignment date could not be parsed: "+module.Name)
							} else {
								snapshot.Assignments[i].Due, snapshot.Assignments[i].HasDue = due, true
							}
						}
						for _, attachment := range parsedPage.Attachments {
							snapshot.Materials = append(snapshot.Materials, model.Material{
								ID:           attachment.ID,
								CourseID:     assignment.CourseID,
								CourseCode:   assignment.CourseCode,
								Name:         attachment.Filename,
								Section:      assignment.Section,
								Kind:         "attachment",
								URL:          ResolveURL(s.siteURL, attachment.URL),
								AssignmentID: assignment.ID,
							})
						}
						break
					}
				}
			}
			assignmentIndex++
		}
	}

	folderPages := make([][]byte, len(folderTasks))
	folderErrors := make([]string, len(folderTasks))
	s.fetchConcurrent(ctx, len(folderTasks), func(index int) {
		task := folderTasks[index]
		pageURL := strings.TrimRight(s.siteURL, "/") + "/mod/folder/view.php?id=" + task.module.CMID
		page, fetchErr := s.getPage(ctx, pageURL)
		if fetchErr != nil {
			folderErrors[index] = fmt.Sprintf("%s: %v", task.module.Name, fetchErr)
			return
		}
		folderPages[index] = page
	})
	for index, task := range folderTasks {
		if failure := folderErrors[index]; failure != "" {
			if strings.Contains(failure, ErrSessionExpired.Error()) {
				return source.Snapshot{}, errors.New("Kulon2 session expired; run `kamps login` again")
			}
			snapshot.Notice = appendNotice(snapshot.Notice, "some folders failed: "+failure)
			continue
		}
		files := ParseFolderPage(folderPages[index], task.module.CMID)
		if len(files) == 0 {
			snapshot.Notice = appendNotice(snapshot.Notice, "folder has no downloadable files: "+task.module.Name)
			continue
		}
		for _, file := range files {
			snapshot.Materials = append(snapshot.Materials, model.Material{
				ID:         file.ID,
				CourseID:   task.course.ID,
				CourseCode: task.course.Short,
				Name:       file.Filename,
				Section:    task.module.Section,
				Folder:     task.module.Name,
				Kind:       "folder-file",
				URL:        ResolveURL(s.siteURL, file.URL),
				ParentID:   MaterialID(task.module.CMID),
			})
		}
	}
	return snapshot, nil
}

func parseActivityPageForModule(moduleType string, page []byte, cmid string) (AssignmentPage, error) {
	if moduleType == "quiz" {
		return ParseQuizPage(page, cmid)
	}
	return ParseAssignmentPage(page, cmid)
}

func appendNotice(notice, addition string) string {
	if notice == "" {
		return addition
	}
	return notice + "; " + addition
}

func (s *LiveSource) listCourses(ctx context.Context, sesskey string) ([]model.Course, error) {
	if err := s.waitForRequestSlot(ctx); err != nil {
		return nil, err
	}
	courses, err := listCoursesWithClient(ctx, s.siteURL, s.session.Cookie, sesskey, s.client)
	s.lastRequest = time.Now()
	if err != nil {
		return nil, err
	}
	return courses, nil
}

func (s *LiveSource) verifySession(ctx context.Context) (Identity, error) {
	if err := s.waitForRequestSlot(ctx); err != nil {
		return Identity{}, err
	}
	identity, err := verifySessionWithClient(ctx, s.siteURL, s.session.Cookie, s.client)
	s.lastRequest = time.Now()
	return identity, err
}

func (s *LiveSource) getPage(ctx context.Context, pageURL string) ([]byte, error) {
	return getPageWithRetry(ctx, pageURL, s.session.Cookie, s.client, s.waitForRequestSlot, s.sleepContext)
}

func (s *LiveSource) sleepContext(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// waitForRequestSlot serializes request starts, keeping the configured interval
// even when page fetching uses several workers.
func (s *LiveSource) waitForRequestSlot(ctx context.Context) error {
	s.requestMu.Lock()
	defer s.requestMu.Unlock()
	if s.minRequestGap > 0 && !s.lastRequest.IsZero() {
		wait := time.Until(s.lastRequest.Add(s.minRequestGap))
		if wait > 0 {
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
	s.lastRequest = time.Now()
	return nil
}

func (s *LiveSource) fetchConcurrent(ctx context.Context, count int, fetch func(index int)) {
	workers := s.maxConcurrent
	if workers < 1 {
		workers = 1
	}
	if workers > count {
		workers = count
	}
	if workers == 0 {
		return
	}
	indices := make(chan int, count)
	for index := 0; index < count; index++ {
		indices <- index
	}
	close(indices)
	var group sync.WaitGroup
	for i := 0; i < workers; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for index := range indices {
				if ctx.Err() != nil {
					continue
				}
				fetch(index)
			}
		}()
	}
	group.Wait()
}

func sameSite(a, b string) bool {
	normalize := func(value string) string { return strings.TrimRight(strings.TrimSpace(value), "/") }
	return normalize(a) != "" && normalize(a) == normalize(b)
}

var _ source.Source = (*LiveSource)(nil)
var _ source.Downloader = (*LiveSource)(nil)
