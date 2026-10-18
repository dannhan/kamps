// Package model holds the core data types kamps works with.
//
// These are deliberately dumb structs: they describe courses, assignments and
// materials as they exist on an LMS. Persistence and fetching live elsewhere
// (see internal/source and, later, internal/store).
package model

import "time"

// Triage is the local, user-owned decision about an assignment.
//
// It is NOT read from the LMS. This tool is not a todo list: it is where you
// record what you decided to do about an assignment that arrived from the LMS,
// from your own tracking, or from a WhatsApp hand-out.
//
//	Undecided  you have not looked at it yet (the default)
//	Tracked    it is handled: submitted, a teammate submitted it for the group,
//	           it is already tracked elsewhere, or it is simply fine
//	Missed     the deadline passed and it is gone
//	Ignored    not real work: a template, last year's file, an optional extra
type Triage int

const (
	Undecided Triage = iota
	Tracked
	Missed
	Ignored
)

func (t Triage) String() string {
	switch t {
	case Tracked:
		return "tracked"
	case Missed:
		return "missed"
	case Ignored:
		return "ignored"
	default:
		return "undecided"
	}
}

// Letter is the compact one-character flag shown in lists ("" when undecided).
func (t Triage) Letter() string {
	switch t {
	case Tracked:
		return "t"
	case Missed:
		return "m"
	case Ignored:
		return "i"
	default:
		return ""
	}
}

// Course is one enrolled course.
type Course struct {
	ID           int    // LMS course id (0 for manual/whatsapp courses)
	Code         string // e.g. MIK1624303
	Short        string // short label used in lists, e.g. "ppl"
	Name         string // e.g. Basis Data C
	Term         string // e.g. smt11
	Source       string // "lms" | "whatsapp" | "manual"
	HasPraktikum bool   // shows the Praktikum column
	PraktikumURL string // Google Drive folder the lab feeds (optional)
}

// Assignment is a graded item, plus the local decision about it.
type Assignment struct {
	ID          string // stable id, e.g. K163827
	CourseID    int
	CourseCode  string // short course code, filled in by the source
	Name        string
	Due         time.Time // zero value + HasDue=false means "no due date"
	HasDue      bool
	Triage      Triage
	Praktikum   bool // belongs to a practicum stream
	Attachments int
	EstMinutes  int    // rough effort estimate, 0 = unknown
	Goal        string // working/ goal this rolls up to, e.g. G-college
}

// Overdue reports whether the item is past due and not yet dealt with.
func (a Assignment) Overdue(now time.Time) bool {
	return a.HasDue && a.Due.Before(now) && a.Triage == Undecided
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
