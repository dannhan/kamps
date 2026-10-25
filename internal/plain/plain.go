// Package plain renders a non-interactive snapshot for pipes, files, and CI.
// It consumes the shared core service and has no Bubble Tea dependency.
package plain

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/core"
	"github.com/ruyam/kamps/internal/format"
	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/source"
)

// PrintPlain writes a plain-text snapshot of the source. It is used when
// stdout is not a terminal (pipes, files, CI) so `kamps` is still useful in
// scripts. No ANSI, no animation. A source-load error is returned before any
// output is written unless a usable cache exists.
func PrintPlain(w io.Writer, src source.Source, cfg config.Config) error {
	service := core.New(src, cfg)
	snapshot, err := service.Load(context.Background())
	if err != nil {
		cached, ok := service.LoadCached()
		if !ok {
			return fmt.Errorf("load %s: %w", src.Label(), err)
		}
		snapshot = cached
	}
	courses, assignments, materials := snapshot.Courses, snapshot.Assignments, snapshot.Materials
	format.SortAssignments(assignments)

	fmt.Fprintf(w, "kamps (plain mode) — source: %s\n", src.Label())
	if snapshot.FromCache {
		fmt.Fprintf(w, "offline: cached from %s\n", snapshot.CachedAt.Format("2006-01-02 15:04"))
	}
	if snapshot.Notice != "" {
		fmt.Fprintf(w, "notice: %s\n", snapshot.Notice)
	}
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
		fmt.Fprintf(w, "  %-8s %-6s %-40s %s\n", core.SectionLabel(mt.Section), state, mt.Name, format.HumanSize(mt.Size))
	}
	return nil
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
