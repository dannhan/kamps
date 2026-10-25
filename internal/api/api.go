// Package api defines the stable JSON shapes for headless kamps commands.
package api

import (
	"encoding/json"
	"io"
	"time"

	"github.com/ruyam/kamps/internal/model"
)

// Version is the wire-contract version.
const Version = 1

// Envelope wraps every successful response.
type Envelope struct {
	Version     int       `json:"version"`
	Command     string    `json:"command"`
	GeneratedAt time.Time `json:"generated_at"`
	Items       any       `json:"items,omitempty"`
	Result      any       `json:"result,omitempty"`
}

// Error is the structured error written to stderr.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorEnvelope wraps an error response.
type ErrorEnvelope struct {
	Version int   `json:"version"`
	Error   Error `json:"error"`
}

// Course is the headless representation of a course.
type Course struct {
	ID            int    `json:"id"`
	Code          string `json:"code"`
	Short         string `json:"short"`
	Name          string `json:"name"`
	Term          string `json:"term"`
	Source        string `json:"source"`
	WorkspacePath string `json:"workspace_path,omitempty"`
}

// Task is the headless representation of an assignment.
type Task struct {
	ID          string     `json:"id"`
	CourseID    int        `json:"course_id"`
	CourseCode  string     `json:"course_code"`
	Name        string     `json:"name"`
	Due         *time.Time `json:"due"`
	HasDue      bool       `json:"has_due"`
	Triage      string     `json:"triage"`
	Submission  string     `json:"submission"`
	Section     string     `json:"section,omitempty"`
	SectionName string     `json:"section_name,omitempty"`
	Overdue     bool       `json:"overdue"`
	Attachments int        `json:"attachments"`
}

// Material is the headless representation of a material.
type Material struct {
	ID           string `json:"id"`
	CourseID     int    `json:"course_id"`
	CourseCode   string `json:"course_code"`
	Name         string `json:"name"`
	Section      string `json:"section,omitempty"`
	SectionName  string `json:"section_name,omitempty"`
	Folder       string `json:"folder,omitempty"`
	Kind         string `json:"kind"`
	Size         int64  `json:"size"`
	Downloaded   bool   `json:"downloaded"`
	URL          string `json:"url,omitempty"`
	AssignmentID string `json:"assignment_id,omitempty"`
}

// Decision is one triage result.
type Decision struct {
	ID     string `json:"id"`
	Triage string `json:"triage"`
}

// ExportResult reports one export write.
type ExportResult struct {
	Added   int    `json:"added"`
	Skipped int    `json:"skipped"`
	Path    string `json:"path"`
}

// Download is one completed transfer.
type Download struct {
	ID    string `json:"id"`
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}

// Courses converts model courses for JSON output.
func Courses(courses []model.Course) []Course {
	out := make([]Course, 0, len(courses))
	for _, course := range courses {
		out = append(out, Course{
			ID:            course.ID,
			Code:          course.Code,
			Short:         course.Short,
			Name:          course.Name,
			Term:          course.Term,
			Source:        course.Source,
			WorkspacePath: course.WorkspacePath,
		})
	}
	return out
}

// Tasks converts model assignments for JSON output.
func Tasks(assignments []model.Assignment, now time.Time) []Task {
	out := make([]Task, 0, len(assignments))
	for _, assignment := range assignments {
		var due *time.Time
		if assignment.HasDue {
			value := assignment.Due
			due = &value
		}
		out = append(out, Task{
			ID:          assignment.ID,
			CourseID:    assignment.CourseID,
			CourseCode:  assignment.CourseCode,
			Name:        assignment.Name,
			Due:         due,
			HasDue:      assignment.HasDue,
			Triage:      assignment.Triage.String(),
			Submission:  assignment.Submission.String(),
			Section:     assignment.Section,
			SectionName: assignment.SectionName,
			Overdue:     assignment.Overdue(now),
			Attachments: assignment.Attachments,
		})
	}
	return out
}

// Materials converts model materials for JSON output.
func Materials(materials []model.Material) []Material {
	out := make([]Material, 0, len(materials))
	for _, material := range materials {
		out = append(out, Material{
			ID:           material.ID,
			CourseID:     material.CourseID,
			CourseCode:   material.CourseCode,
			Name:         material.Name,
			Section:      material.Section,
			SectionName:  material.SectionName,
			Folder:       material.Folder,
			Kind:         material.Kind,
			Size:         material.Size,
			Downloaded:   material.Downloaded,
			URL:          material.URL,
			AssignmentID: material.AssignmentID,
		})
	}
	return out
}

// Write encodes one success envelope.
func Write(w io.Writer, command string, items, result any) error {
	return json.NewEncoder(w).Encode(Envelope{
		Version:     Version,
		Command:     command,
		GeneratedAt: time.Now(),
		Items:       items,
		Result:      result,
	})
}

// WriteError encodes one structured error.
func WriteError(w io.Writer, code, message string) error {
	return json.NewEncoder(w).Encode(ErrorEnvelope{
		Version: Version,
		Error:   Error{Code: code, Message: message},
	})
}
