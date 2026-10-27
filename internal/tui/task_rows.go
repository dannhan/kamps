package tui

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/ruyam/kamps/internal/model"
)

const actNowPrefixWidth = 8

func (m *App) actNowCoursePrefix(courseID int) string {
	course, ok := m.courseByID(courseID)
	if !ok {
		return m.shortCode(courseID)
	}
	if alias := strings.TrimSpace(course.Short); alias != "" {
		return alias
	}
	if initials := courseInitials(course.Name); initials != "" {
		return initials
	}
	return course.Name
}

func (m *App) actNowPrefixCell(h rowHighlight, courseID int) string {
	label := pad(m.actNowCoursePrefix(courseID), actNowPrefixWidth, false)
	return h.style(stFaint).Render(label + " · ")
}

func courseInitials(name string) string {
	fields := strings.Fields(name)
	if len(fields) == 0 {
		return ""
	}
	if len(fields) > 1 && isSectionLetter(fields[len(fields)-1]) {
		fields = fields[:len(fields)-1]
	}
	var initials strings.Builder
	for _, field := range fields {
		for _, r := range field {
			if unicode.IsUpper(r) {
				initials.WriteRune(r)
			}
			break
		}
	}
	return initials.String()
}

func isSectionLetter(value string) bool {
	runes := []rune(value)
	return len(runes) == 1 && runes[0] >= 'A' && runes[0] <= 'Z'
}

func (m *App) taskTreeRow(nodes []node, index, width int) string {
	return strings.Join(m.taskBlock(nodes[index], index, width), "\n")
}

func (m *App) isTaskGroup(n node) bool {
	return n.kind == nodeCourse || n.kind == nodeBucket
}

func (m *App) taskNodeCounts(group node) [5]int {
	assignments := make([]model.Assignment, 0, 8)
	for _, id := range m.taskGroupIDs(group) {
		if assignment := m.byID(id); assignment != nil {
			assignments = append(assignments, *assignment)
		}
	}
	return tallyTriage(assignments)
}

// tallyTriage buckets assignments by decision state. Index 2 ("done") is kept
// for the five-glyph legend shape but no triage value maps to it.
func tallyTriage(assignments []model.Assignment) [5]int {
	var counts [5]int
	for _, assignment := range assignments {
		switch assignment.Triage {
		case model.Tracked:
			counts[1]++
		case model.Ignored:
			counts[3]++
		case model.Missed:
			counts[4]++
		default:
			counts[0]++
		}
	}
	return counts
}

// formatTaskCounts renders the per-state counts of a container. It takes the
// row's highlight so every count carries the band background: rendering the
// styled counts under a background wrapper loses it to the inner colour resets.
func formatTaskCounts(counts [5]int, h rowHighlight) string {
	states := []struct {
		count int
		glyph string
		style lipgloss.Style
	}{{counts[0], "○", stFlagNew}, {counts[1], "◐", stFlagT}, {counts[2], "●", stDone}, {counts[3], "◆", stFlagI}, {counts[4], "×", stFlagM}}
	var parts []string
	for _, state := range states {
		if state.count > 0 {
			parts = append(parts, h.style(state.style).Render(fmt.Sprintf("%s %d", state.glyph, state.count)))
		}
	}
	return strings.Join(parts, h.blanks(2))
}

func (m *App) taskRows(nodes []node, width, height int) []string {
	rows, _, _ := m.taskListWindow(nodes, width, height)
	return rows
}

// taskTotals is the top-bar legend with the totals for every decision state
// across all assignments, so the numbers do not move with the active filter.
// It degrades to glyphs+counts and then to nothing as width runs out.
func (m *App) taskTotals(width int) string {
	totals := tallyTriage(m.assignments)
	states := []struct {
		count int
		glyph string
		label string
		style lipgloss.Style
	}{
		{totals[0], "○", "new", stFlagNew},
		{totals[1], "◐", "tracked", stFlagT},
		{totals[3], "◆", "ignored", stFlagI},
		{totals[4], "×", "missed", stFlagM},
	}
	var full, compact []string
	for _, state := range states {
		full = append(full, state.style.Render(state.glyph)+" "+stDim.Render(fmt.Sprintf("%s %d", state.label, state.count)))
		compact = append(compact, state.style.Render(fmt.Sprintf("%s %d", state.glyph, state.count)))
	}
	if legend := strings.Join(full, "  "); lipgloss.Width(legend) <= width {
		return legend
	}
	if legend := strings.Join(compact, "  "); lipgloss.Width(legend) <= width {
		return legend
	}
	return ""
}

func (m *App) tasksContent(width, height int) (string, string, []string, []string) {
	return m.workspaceContent(width, height)
}
