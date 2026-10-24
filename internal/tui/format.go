package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/ruyam/kamps/internal/model"
)

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
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
