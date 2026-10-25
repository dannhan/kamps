package kulon2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/ruyam/kamps/internal/model"
)

const courseListMethod = "core_course_get_enrolled_courses_by_timeline_classification"

// ListCourses calls Moodle's enrolled-courses AJAX service. It refuses to
// interpret an API error, malformed response, or unexpected empty result as a
// successful empty course list.
func ListCourses(ctx context.Context, siteURL, cookie, sesskey string) ([]model.Course, error) {
	return listCoursesWithClient(ctx, siteURL, cookie, sesskey, nil)
}

func listCoursesWithClient(ctx context.Context, siteURL, cookie, sesskey string, client *http.Client) ([]model.Course, error) {
	if strings.TrimSpace(cookie) == "" {
		return nil, errors.New("no MoodleSession cookie; run `kamps login` first")
	}
	if strings.TrimSpace(sesskey) == "" {
		return nil, errors.New("saved session has no sesskey; run `kamps login` again")
	}
	base, err := parseSiteURL(siteURL)
	if err != nil {
		return nil, err
	}

	client = cloneClientWithCookie(client, base, cookie, 30*time.Second)

	endpoint := *base
	endpoint.Path = strings.TrimRight(base.Path, "/") + "/lib/ajax/service.php"
	query := url.Values{}
	query.Set("sesskey", sesskey)
	query.Set("info", courseListMethod)
	endpoint.RawQuery = query.Encode()

	payload := []struct {
		Index      int            `json:"index"`
		MethodName string         `json:"methodname"`
		Args       map[string]any `json:"args"`
	}{{
		Index:      0,
		MethodName: courseListMethod,
		Args: map[string]any{
			"offset":         0,
			"limit":          0,
			"classification": "inprogress",
			"sort":           "fullname",
		},
	}}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "kamps/0.1 (Kulon2 client; polite requests)")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch enrolled courses: %w", err)
	}
	defer resp.Body.Close()
	if resp.Request.URL.Host != base.Host || (resp.StatusCode >= 300 && resp.StatusCode < 400) {
		return nil, ErrSessionExpired
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("course-list request: %s", resp.Status)
	}
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("read course list: %w", err)
	}
	courses, err := decodeCourses(responseBody)
	if err != nil {
		return nil, err
	}
	if len(courses) == 0 {
		return nil, errors.New("Kulon2 returned no courses; refusing to treat this as a valid empty list")
	}
	return courses, nil
}

type serviceReply struct {
	Error     bool            `json:"error"`
	Exception string          `json:"exception"`
	Message   string          `json:"message"`
	Data      json.RawMessage `json:"data"`
}

type rawCourse struct {
	ID        int    `json:"id"`
	ShortName string `json:"shortname"`
	FullName  string `json:"fullname"`
}

type courseListData struct {
	Courses []rawCourse `json:"courses"`
}

func decodeCourses(body []byte) ([]model.Course, error) {
	var replies []serviceReply
	if err := json.Unmarshal(body, &replies); err != nil {
		if bytes.Contains(bytes.ToLower(body), []byte("login")) {
			return nil, ErrSessionExpired
		}
		return nil, fmt.Errorf("parse Kulon2 course-list response: %w", err)
	}
	if len(replies) != 1 {
		return nil, fmt.Errorf("unexpected course-list response: got %d service replies", len(replies))
	}
	reply := replies[0]
	if reply.Error {
		detail := strings.TrimSpace(reply.Exception + ": " + reply.Message)
		lowerError := strings.ToLower(detail)
		if strings.Contains(lowerError, "sesskey") || strings.Contains(strings.ToLower(reply.Exception), "invalidsesskey") {
			return nil, ErrInvalidSesskey
		}
		if strings.Contains(lowerError, "session") {
			return nil, ErrSessionExpired
		}
		return nil, fmt.Errorf("Kulon2 course-list service error: %s", strings.Trim(detail, ": "))
	}
	if len(reply.Data) == 0 || string(reply.Data) == "null" {
		return nil, errors.New("Kulon2 course-list response has no data")
	}
	var rawCourses []rawCourse
	if err := json.Unmarshal(reply.Data, &rawCourses); err != nil {
		var data courseListData
		if objectErr := json.Unmarshal(reply.Data, &data); objectErr != nil || data.Courses == nil {
			return nil, fmt.Errorf("parse Kulon2 course-list data: %w", err)
		}
		rawCourses = data.Courses
	}
	courses := make([]model.Course, 0, len(rawCourses))
	for _, item := range rawCourses {
		if item.ID <= 0 || strings.TrimSpace(item.FullName) == "" {
			return nil, errors.New("Kulon2 course-list response contains a course without a valid id or name")
		}
		short := extractCourseCode(item.ShortName, item.FullName)
		if short == "" {
			short = strings.TrimSpace(item.ShortName)
		}
		courses = append(courses, model.Course{
			ID:     item.ID,
			Code:   short,
			Short:  short,
			Name:   cleanCourseName(item.FullName),
			Source: "lms",
		})
	}
	return courses, nil
}

var (
	leadingCourseTags = regexp.MustCompile(`^\s*(?:\[[^\]]*\]\s*)+`)
	semesterPrefix    = regexp.MustCompile(`(?i)^S[0-9]+\s+[0-9]{4}/[0-9]{4}\s+(Ganjil|Genap)\s+`)
	bracketedCode     = regexp.MustCompile(`\[([A-Z]{2,}[0-9][A-Z0-9]*)\]`)
)

// cleanCourseName strips Moodle's category chain and semester prefix.
func cleanCourseName(full string) string {
	name := strings.TrimSpace(leadingCourseTags.ReplaceAllString(full, ""))
	name = strings.TrimSpace(semesterPrefix.ReplaceAllString(name, ""))
	if name == "" {
		return strings.TrimSpace(full)
	}
	return name
}

// extractCourseCode picks the longest code-like bracket, e.g. MIK1624302.
func extractCourseCode(short, full string) string {
	best := ""
	for _, source := range []string{short, full} {
		for _, match := range bracketedCode.FindAllStringSubmatch(source, -1) {
			if len(match[1]) > len(best) {
				best = match[1]
			}
		}
	}
	return best
}
