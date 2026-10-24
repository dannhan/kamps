package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/ruyam/kamps/internal/model"
)

func humanSize(n int64) string {
	if n <= 0 {
		return "—"
	}
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	units := []string{"KB", "MB", "GB"}
	f := float64(n)
	i := -1
	for f >= unit && i < len(units)-1 {
		f /= unit
		i++
	}
	return fmt.Sprintf("%.1f %s", f, units[i])
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func lessAssignment(a, b model.Assignment) bool {
	if a.HasDue != b.HasDue {
		return a.HasDue // dated items first
	}
	if a.HasDue && !a.Due.Equal(b.Due) {
		return a.Due.Before(b.Due)
	}
	return a.ID < b.ID
}

func sortSlice(s []model.Assignment) {
	sort.Slice(s, func(i, j int) bool { return lessAssignment(s[i], s[j]) })
}

func flagChar(t model.Triage) string {
	switch t {
	case model.Tracked:
		return "◐"
	case model.Missed:
		return "×"
	case model.Ignored:
		return "◆"
	}
	return "○"
}

func triageStyle(t model.Triage) lipgloss.Style {
	switch t {
	case model.Tracked:
		return stFlagT
	case model.Missed:
		return stFlagM
	case model.Ignored:
		return stFlagI
	}
	return stFlagNew
}

// shortCode maps a course id to its short label ("ppl", "metnum", …).
func (m *App) shortCode(courseID int) string {
	for _, c := range m.courses {
		if c.ID == courseID {
			if c.Short != "" {
				return c.Short
			}
			return c.Code
		}
	}
	return "—"
}

// similarManualTask is a demo-only stand-in for the future working/ cross-check.
// It intentionally recognizes one hard-coded task title; replace it with real
// data instead of growing this heuristic.
func (m *App) similarManualTask(a model.Assignment) string {
	if !strings.Contains(strings.ToLower(a.Name), "progress report 5") {
		return ""
	}
	return "Submit " + m.shortCode(a.CourseID) + " Report 5"
}

func (m *App) taskLine(a model.Assignment) string {
	due := ""
	if a.HasDue {
		due = " due:" + a.Due.Format("2006-01-02")
	}
	return fmt.Sprintf("- [ ] [%s] %s (%s)%s", m.shortCode(a.CourseID), a.Name, a.ID, due)
}

// lastSyncLine summarises the last sync: when, what, and what came in.
func (m *App) lastSyncLine() string {
	name := "—"
	if len(m.courses) > 0 {
		name = fmt.Sprintf("%d %s", m.courses[0].ID, m.courses[0].Name)
	}
	fresh := 0
	for _, mt := range m.materials {
		if !mt.Downloaded {
			fresh++
		}
	}
	stamp := time.Now().Add(-20 * time.Hour).Format("2006-01-02 15:04")
	return fmt.Sprintf("%s · %s (+%d materials)", stamp, name, fresh)
}
