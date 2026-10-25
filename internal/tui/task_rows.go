package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/ruyam/kamps/internal/model"
)

// taskRowRightPad keeps the tree's right edge (course counts, state labels)
// off the pane divider and the terminal edge. The padding is painted with the
// row highlight, so a cursor row still reaches the edge as one band.
const taskRowRightPad = 2

func (m *App) taskTreeRow(nodes []node, index, width int) string {
	n := nodes[index]
	selected := false
	if n.kind == nodeTask {
		selected = m.selected[n.task.ID]
	} else if m.isTaskGroup(n) {
		selected = allSelected(m.selected, m.taskGroupIDs(n))
	}
	h := rowHighlight{cursor: index == m.taskCursor, mark: selected}
	background := h.style
	prefix := h.gutter() + background(stFaint).Render(treeGuides(nodes, index))
	available := maxInt(1, width-lipgloss.Width(prefix)-taskRowRightPad)
	group := m.isTaskGroup(n)
	open := group && m.taskIsOpen(n.key, taskNodeDefaultOpen(n))
	content := ""
	switch n.kind {
	case nodeCourse:
		name := m.shortCode(n.courseID)
		if course, ok := m.courseByID(n.courseID); ok && course.Name != "" {
			name = course.Name
		}
		content = h.containerLead(open, selected) + background(stCode.Bold(true)).Render(name)
	case nodeBucket:
		content = h.containerLead(open, selected) + background(stRow).Render(fmt.Sprintf("%s (%d)", n.bucket, len(m.taskGroupIDs(n))))
	case nodeTask:
		marker := h.markCell(selected)
		caret := ""
		if group {
			caret = background(stFaint).Render(caretGlyph(open)) + background(stRow).Render(" ")
		}
		now := time.Now()
		age := dueAge(n.task, now)
		ageStyle := stRowDim
		if n.task.HasDue && n.task.Due.Before(now) {
			ageStyle = stOverdue
		}
		ageText := background(ageStyle).Render(age)
		lead := marker + background(stRow).Render(" ") + caret
		nameWidth := maxInt(1, available-lipgloss.Width(lead)-lipgloss.Width(ageText)-1)
		name := background(stRow).Render(ansi.Truncate(n.task.Name, nameWidth, "…"))
		gap := maxInt(1, available-lipgloss.Width(lead)-lipgloss.Width(name)-lipgloss.Width(ageText))
		content = lead + name + background(stRow).Render(strings.Repeat(" ", gap)) + ageText
	case nodeTaskMaterial:
		_, style := materialTreeState(n.mat)
		glyph := "○"
		if n.mat.Downloaded {
			glyph = "●"
		}
		label := ansi.Truncate(n.mat.Name, maxInt(1, available-5), "…")
		space := background(stRow).Render(" ")
		content = background(style).Render(glyph) + space + background(stFaint).Render("⁝") + space + background(stRow).Render(label)
	}

	content = ansiTruncate(content, available)
	row := prefix + content
	padding := strings.Repeat(" ", maxInt(0, width-lipgloss.Width(row)))
	if h.band() {
		padding = background(stRow).Render(padding)
	}
	return row + padding
}

func (m *App) isTaskGroup(n node) bool {
	return n.kind == nodeCourse || n.kind == nodeBucket || n.kind == nodeTask && len(m.assignmentMaterials(n.task)) > 0
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

func dueAge(assignment model.Assignment, now time.Time) string {
	if !assignment.HasDue {
		return "—"
	}
	dueDay := time.Date(assignment.Due.Year(), assignment.Due.Month(), assignment.Due.Day(), 0, 0, 0, 0, assignment.Due.Location())
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	days := int(dueDay.Sub(today).Hours() / 24)
	if days < 0 {
		return fmt.Sprintf("−%dd", -days)
	}
	return fmt.Sprintf("+%dd", days)
}

func (m *App) taskRows(nodes []node, width, height int) []string {
	start, end := windowRange(len(nodes), m.taskCursor, height)
	rows := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		rows = append(rows, m.taskTreeRow(nodes, i, width))
	}
	if len(rows) == 0 {
		rows = append(rows, stEmpty.Render("nothing matches this filter"))
	}
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
	nodes := m.taskNodes()
	matches := len(m.visible())
	hint := scrollHint(len(nodes), m.taskCursor, height)
	var title, right string
	if m.searchActive() {
		title = m.searchTitle()
		right = rightAligned(fmt.Sprintf("%d %s", matches, plural(matches, "match", "matches")), hint)
	} else {
		title = fmt.Sprintf("courses · %s · %d assignments", m.filter, matches)
		right = hint
	}
	filterHint := "f filter"
	if m.filter != "all" || m.search != "" {
		filterHint = "esc clear"
	}
	detailHint := "enter detail"
	if m.detailVisible() {
		detailHint = "enter hide"
	}
	hints := []string{"↑↓ move", "t/m/i decide", "c commit", filterHint, detailHint, "? more"}
	leftWidth := width
	if m.detailVisible() {
		leftWidth, _ = splitWidths(width)
	}
	rows := m.taskRows(nodes, leftWidth, height)
	if m.detailVisible() {
		rows = m.splitTreeRows(rows, nodes, m.taskCursor, width, height)
	}
	return title, right, hints, rows
}
