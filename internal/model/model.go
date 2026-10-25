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

// SubmissionState is what the LMS reports about a submission. It is source
// data, not a local decision; the TUI uses it to derive automatic triage.
type SubmissionState int

const (
	SubmissionUnknown SubmissionState = iota
	SubmissionNotSubmitted
	SubmissionSubmitted
)

func (s SubmissionState) String() string {
	switch s {
	case SubmissionSubmitted:
		return "submitted"
	case SubmissionNotSubmitted:
		return "not submitted"
	default:
		return "unknown"
	}
}

// Course is one enrolled course.
type Course struct {
	ID            int    // LMS course id (0 for manual/whatsapp courses)
	Code          string // e.g. MIK1624303
	Short         string // short label used in lists, e.g. "ppl"
	Name          string // e.g. Basis Data C
	Term          string // e.g. smt11
	Source        string // "lms" | "whatsapp" | "manual"
	HasPraktikum  bool   // shows the Praktikum column
	PraktikumURL  string // Google Drive folder the lab feeds (optional)
	WorkspacePath string // resolved local workspace folder (runtime only)
}

// Assignment is a graded item, plus the local decision about it.
type Assignment struct {
	ID          string // stable id, e.g. K163827
	CourseID    int
	CourseCode  string // short course code, filled in by the source
	Name        string
	Section     string    // course-page section label, when known
	Due         time.Time // zero value + HasDue=false means "no due date"
	HasDue      bool
	Triage      Triage
	Submission  SubmissionState
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
//
// Kind is one of:
//
//	file        a resource file downloaded from pluginfile.php
//	attachment  an assignment attachment (pluginfile.php)
//	folder-file a child of a folder module
//	url         an external link (open with `o`)
//	book        an HTML book activity (open with `o`, not downloadable)
//	praktikum   a lab-drive link
type Material struct {
	ID           string // stable id, e.g. M165401 or M165401:file.pdf
	CourseID     int
	CourseCode   string
	Name         string
	Section      string // section/week label, or "" when unknown
	Folder       string // folder-module title for folder children
	Kind         string
	Size         int64 // bytes; 0 = unknown
	Downloaded   bool
	URL          string
	AssignmentID string // for attachments: the owning K<cmid>
	ParentID     string // for folder children: the M<cmid> folder module
}
