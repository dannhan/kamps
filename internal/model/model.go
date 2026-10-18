// Package model holds the core data types kamps works with.
//
// These are deliberately dumb structs: they describe courses, assignments and
// materials as they exist on an LMS. Persistence and fetching live elsewhere
// (see internal/source and, later, internal/store).
package model

import "time"

// Status is the local, user-owned state of an assignment.
//
// It is intentionally NOT read from the LMS: the LMS knows whether you
// submitted, but only you know whether you still intend to do something.
type Status int

const (
	Todo    Status = iota // not started
	Doing                 // in progress
	Done                  // finished
	Ignored               // not mine / not relevant
)

func (s Status) String() string {
	switch s {
	case Todo:
		return "todo"
	case Doing:
		return "doing"
	case Done:
		return "done"
	case Ignored:
		return "ignored"
	}
	return "?"
}

// Glyph is the compact marker shown in lists.
func (s Status) Glyph() string {
	switch s {
	case Todo:
		return "○"
	case Doing:
		return "◐"
	case Done:
		return "●"
	case Ignored:
		return "–"
	}
	return "?"
}

// Course is one enrolled course.
type Course struct {
	ID           int    // LMS course id (0 for manual/whatsapp courses)
	Code         string // e.g. MIK1624303
	Name         string // e.g. Basis Data C
	Term         string // e.g. smt11
	Source       string // "lms" | "whatsapp" | "manual"
	HasPraktikum bool   // shows the Praktikum tab
	PraktikumURL string // Google Drive folder the lab feeds (optional)
}

// Assignment is a graded item with (usually) a due date.
type Assignment struct {
	ID          string // stable id, e.g. K165402
	CourseID    int
	CourseCode  string // filled in by the source for convenience
	Name        string
	Due         time.Time // zero value + HasDue=false means "no due date"
	HasDue      bool
	Status      Status
	Praktikum   bool // belongs to a practicum stream
	Attachments int  // number of files attached to the assignment
}

// Material is a downloadable/readable course artifact.
type Material struct {
	ID         string // stable id, e.g. M165401
	CourseID   int
	CourseCode string
	Name       string
	Section    string // "week-05", or "" when unknown
	Kind       string // "file" | "attachment" | "url" | "praktikum"
	Size       int64  // bytes; 0 = unknown
	Downloaded bool
	URL        string
}
