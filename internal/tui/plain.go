package tui

import (
	"context"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/source"
)

// PrintPlain writes a plain-text snapshot of the source. It is used when
// stdout is not a terminal (pipes, files, CI) so `kamps` is still useful in
// scripts. No ANSI, no animation.
func PrintPlain(w io.Writer, src source.Source) {
	ctx := context.Background()
	courses, _ := src.Courses(ctx)
	assignments, _ := src.Assignments(ctx)
	materials, _ := src.Materials(ctx)

	sort.Slice(courses, func(i, j int) bool { return courses[i].Name < courses[j].Name })
	sort.Slice(assignments, func(i, j int) bool { return lessAssignment(assignments[i], assignments[j]) })

	fmt.Fprintf(w, "kamps (plain mode) — source: %s\n", src.Label())
	fmt.Fprintf(w, "%d courses · %d assignments · %d materials\n", len(courses), len(assignments), len(materials))

	fmt.Fprintf(w, "\nCOURSES\n")
	for _, c := range courses {
		tag := ""
		if c.HasPraktikum {
			tag = "  [praktikum]"
		}
		if c.Source != "lms" {
			tag = "  [" + c.Source + "]"
		}
		fmt.Fprintf(w, "  %-28s %-11s%s\n", c.Name, c.Code, tag)
	}

	now := time.Now()
	fmt.Fprintf(w, "\nASSIGNMENTS\n")
	for _, a := range assignments {
		due := "no due"
		if a.HasDue {
			due = a.Due.Format("2006-01-02")
			if a.Status != model.Done && a.Due.Before(now) {
				due += " (overdue)"
			}
		}
		fmt.Fprintf(w, "  %-16s %-10s %-8s %s\n", due, a.CourseCode, a.Status.String(), a.Name)
	}

	fmt.Fprintf(w, "\nMATERIALS\n")
	for _, mt := range materials {
		state := "new"
		if mt.Downloaded {
			state = "local"
		}
		week := mt.Section
		if week == "" {
			week = "—"
		}
		fmt.Fprintf(w, "  %-8s %-6s %-40s %s\n", week, state, mt.Name, humanSize(mt.Size))
	}
}
