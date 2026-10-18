package tui

import (
	"context"
	"fmt"
	"io"
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
	sortSlice(assignments)

	fmt.Fprintf(w, "kamps (plain mode) — source: %s\n", src.Label())
	fmt.Fprintf(w, "%d courses · %d assignments · %d materials\n", len(courses), len(assignments), len(materials))

	fmt.Fprintf(w, "\nASSIGNMENTS\n")
	now := time.Now()
	for _, a := range assignments {
		due := "no due"
		if a.HasDue {
			due = a.Due.Format("2006-01-02")
			if a.Overdue(now) {
				due += " (overdue)"
			}
		}
		flag := a.Triage.Letter()
		if flag == "" {
			flag = "-"
		}
		fmt.Fprintf(w, "  [%s] %-16s %-10s %-9s %s\n", flag, due, shortOf(courses, a.CourseID), a.Triage.String(), a.Name)
	}

	fmt.Fprintf(w, "\nMATERIALS\n")
	for _, mt := range materials {
		state := "new"
		if mt.Downloaded {
			state = "local"
		}
		fmt.Fprintf(w, "  %-8s %-6s %-40s %s\n", sectionOr(mt.Section), state, mt.Name, humanSize(mt.Size))
	}
}

func shortOf(courses []model.Course, id int) string {
	for _, c := range courses {
		if c.ID == id {
			if c.Short != "" {
				return c.Short
			}
			return c.Code
		}
	}
	return "—"
}

func sectionOr(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
