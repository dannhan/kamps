package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/ruyam/kamps/internal/model"
)

var sparkBlocks = []rune("▁▂▃▄▅▆▇█")

func sparkline(counts []int) string {
	max := 0
	for _, count := range counts {
		if count > max {
			max = count
		}
	}
	if max == 0 {
		return ""
	}
	var b strings.Builder
	for _, count := range counts {
		if count == 0 {
			b.WriteRune('▁')
			continue
		}
		index := (count*len(sparkBlocks) - 1) / max
		b.WriteRune(sparkBlocks[index])
	}
	return b.String()
}

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
		bar := progressBar(local, total, 5)
		return stFaint.Render(fmt.Sprintf("%d local · %d new  ", local, total-local)) + stDone.Render(bar)
	}
	now := time.Now()
	overdue, dueSoon, newCount := 0, 0, 0
	var days [7]int
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
		if offset >= 0 && offset < len(days) {
			days[offset]++
			if !assignment.Overdue(now) {
				dueSoon++
			}
		}
	}
	var parts []string
	if overdue > 0 {
		parts = append(parts, stOverdue.Render(fmt.Sprintf("%d overdue", overdue)))
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
	summary := strings.Join(parts, stFaint.Render(" · "))
	if spark := sparkline(days[:]); spark != "" {
		summary += "  " + stCode.Render(spark)
	}
	return summary
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
	return courseMeta(courseID, decided, total, h)
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
	return courseMeta(courseID, local, total, h)
}

func courseMeta(courseID, filled, total int, h rowHighlight) string {
	if total == 0 {
		return ""
	}
	bar := progressBar(filled, total, 5)
	style := courseStyle(courseID)
	return h.style(style).Render(bar) + h.style(stFaint).Render(fmt.Sprintf(" %d/%d", filled, total))
}

func courseStyle(courseID int) lipgloss.Style {
	index := courseID % len(courseAccents)
	if index < 0 {
		index = -index
	}
	return lipgloss.NewStyle().Foreground(courseAccents[index])
}
