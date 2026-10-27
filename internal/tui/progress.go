package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/ruyam/kamps/internal/model"
)

func progressBar(filled, total, width int) string {
	if total <= 0 || width <= 0 {
		return ""
	}
	filled = clampInt(filled, 0, total)
	on := filled * width / total
	if filled > 0 && on == 0 {
		on = 1
	}
	return strings.Repeat("█", on) + strings.Repeat("░", width-on)
}

func (m *App) tabSummary() string {
	if !m.loaded {
		return ""
	}
	if m.tab == TabMaterials {
		total := len(m.materials)
		local := 0
		for _, material := range m.materials {
			if material.Downloaded {
				local++
			}
		}
		if total == 0 {
			return ""
		}
		summary := stFaint.Render(fmt.Sprintf("%d local · %d new  ", local, total-local))
		if m.sizeClass() <= sizeCompact {
			return summary
		}
		return summary + stDone.Render(progressBar(local, total, 5))
	}
	now := m.currentTime()
	overdue, dueSoon, newCount := 0, 0, 0
	for _, assignment := range m.assignments {
		if assignment.Triage == model.Undecided {
			newCount++
		}
		if assignment.Overdue(now) {
			overdue++
		}
		if !assignment.HasDue {
			continue
		}
		dueDay := dateOnly(assignment.Due)
		today := dateOnly(now)
		offset := int(dueDay.Sub(today).Hours() / 24)
		if offset >= 0 && offset <= actNowLookAheadDays && !assignment.Overdue(now) {
			dueSoon++
		}
	}
	var parts []string
	if overdue > 0 {
		label := fmt.Sprintf("%d overdue", overdue)
		if stale := m.staleOverdueCount(); stale > 0 {
			label += fmt.Sprintf(" (%d stale)", stale)
		}
		parts = append(parts, stOverdue.Render(label))
	}
	if dueSoon > 0 {
		parts = append(parts, stSoon.Render(fmt.Sprintf("%d due soon", dueSoon)))
	}
	if newCount > 0 {
		parts = append(parts, stFlagNew.Render(fmt.Sprintf("%d new", newCount)))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, stFaint.Render(" · "))
}

func (m *App) overdueCount() int {
	now := m.currentTime()
	n := 0
	for _, assignment := range m.assignments {
		if assignment.Overdue(now) {
			n++
		}
	}
	return n
}

func (m *App) staleOverdueCount() int {
	now := m.currentTime()
	n := 0
	for _, assignment := range m.assignments {
		if classifyTask(assignment, now) == taskGroupStale {
			n++
		}
	}
	return n
}

func dateOnly(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func (m *App) taskCourseMeta(courseID int, h rowHighlight) string {
	total, decided := 0, 0
	for _, assignment := range m.assignments {
		if assignment.CourseID != courseID {
			continue
		}
		total++
		if assignment.Triage != model.Undecided {
			decided++
		}
	}
	return m.courseMeta(courseID, decided, total, h)
}

func (m *App) materialCourseMeta(courseID int, h rowHighlight) string {
	total, local := 0, 0
	for _, material := range m.materials {
		if material.CourseID != courseID {
			continue
		}
		total++
		if material.Downloaded {
			local++
		}
	}
	return m.courseMeta(courseID, local, total, h)
}

func (m *App) courseMeta(courseID, filled, total int, h rowHighlight) string {
	if total == 0 {
		return ""
	}
	style := m.courseStyle(courseID)
	if m.sizeClass() <= sizeCompact {
		return h.style(style).Render(fmt.Sprintf("%d/%d", filled, total))
	}
	bar := progressBar(filled, total, 5)
	return h.style(style).Render(bar) + h.style(stFaint).Render(fmt.Sprintf(" %d/%d", filled, total))
}
