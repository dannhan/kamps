package core

import (
	"strings"
	"time"

	"github.com/ruyam/kamps/internal/format"
	"github.com/ruyam/kamps/internal/model"
)

// Query narrows a task or material list. CourseID 0 means all courses.
type Query struct {
	Filter   string
	Search   string
	CourseID int
}

// Tasks returns the visible assignments in display order.
func (s *Service) Tasks(query Query) []model.Assignment {
	search := strings.ToLower(strings.TrimSpace(query.Search))
	out := make([]model.Assignment, 0, len(s.snapshot.Assignments))
	for _, assignment := range s.snapshot.Assignments {
		if query.CourseID != 0 && assignment.CourseID != query.CourseID {
			continue
		}
		if !matchTaskFilter(query.Filter, assignment) {
			continue
		}
		if search != "" && !taskMatches(assignment, search, s.snapshot.Courses) {
			continue
		}
		out = append(out, assignment)
	}
	format.SortAssignments(out)
	return out
}

// Materials returns the visible materials in display order.
func (s *Service) Materials(query Query) []model.Material {
	search := strings.ToLower(strings.TrimSpace(query.Search))
	out := make([]model.Material, 0, len(s.snapshot.Materials))
	for _, material := range s.snapshot.Materials {
		if query.CourseID != 0 && material.CourseID != query.CourseID {
			continue
		}
		if !matchMaterialFilter(query.Filter, material) {
			continue
		}
		if search != "" && !materialMatches(material, search, s.snapshot.Courses) {
			continue
		}
		out = append(out, material)
	}
	return out
}

// Course resolves a numeric ID or a code/short/name fragment.
func (s *Service) Course(raw string) (model.Course, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return model.Course{}, false
	}
	for _, course := range s.snapshot.Courses {
		if raw == course.Short || raw == course.Code || strings.EqualFold(raw, course.Short) ||
			strings.EqualFold(raw, course.Code) || strings.EqualFold(raw, course.Name) {
			return course, true
		}
	}
	return model.Course{}, false
}

func matchTaskFilter(filter string, a model.Assignment) bool {
	switch filter {
	case "", "all":
		return true
	case "undecided", "new":
		return a.Triage == model.Undecided
	case "tracked":
		return a.Triage == model.Tracked
	case "missed":
		return a.Triage == model.Missed
	case "ignored":
		return a.Triage == model.Ignored
	case "overdue":
		return a.Overdue(time.Now())
	}
	return true
}

func matchMaterialFilter(filter string, m model.Material) bool {
	switch filter {
	case "", "all":
		return true
	case "new":
		return !m.Downloaded
	case "downloaded", "local":
		return m.Downloaded
	case "link":
		return m.Kind == "url"
	}
	return true
}

func taskMatches(a model.Assignment, search string, courses []model.Course) bool {
	fields := []string{a.Name, a.ID, a.Goal, a.Section, a.SectionName}
	if course, ok := findCourse(courses, a.CourseID); ok {
		fields = append(fields, course.Name, course.Code, course.Short)
	}
	return containsAny(search, fields)
}

func materialMatches(m model.Material, search string, courses []model.Course) bool {
	fields := []string{m.Name, m.ID, m.Section, m.SectionName, m.Kind, m.CourseCode}
	if course, ok := findCourse(courses, m.CourseID); ok {
		fields = append(fields, course.Name, course.Code, course.Short)
	}
	return containsAny(search, fields)
}

func containsAny(search string, fields []string) bool {
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), search) {
			return true
		}
	}
	return false
}

func findCourse(courses []model.Course, id int) (model.Course, bool) {
	for _, course := range courses {
		if course.ID == id {
			return course, true
		}
	}
	return model.Course{}, false
}
