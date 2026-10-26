package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/ruyam/kamps/internal/model"
)

func (m *App) taskTreeRow(nodes []node, index, width int) string {
	n := nodes[index]
	group := m.isTaskGroup(n)
	ids := m.taskGroupIDs(n)
	selected := false
	if n.kind == nodeTask {
		selected = m.selected[n.task.ID]
	} else if group {
		selected = allSelected(m.selected, ids)
	}
	open := group && m.taskIsOpen(n.key, taskNodeDefaultOpen(n))
	spec := treeRowSpec{
		nodes:      nodes,
		index:      index,
		width:      width,
		cursor:     index == m.taskCursor,
		selected:   selected,
		partial:    !selected && partialSelected(m.selected, ids),
		expandable: group,
		open:       open,
	}
	switch n.kind {
	case nodeCourse:
		name := m.shortCode(n.courseID)
		if course, ok := m.courseByID(n.courseID); ok && course.Name != "" {
			name = course.Name
		}
		spec.title = func(h rowHighlight) string {
			return h.style(courseStyle(n.courseID).Bold(true)).Render(name)
		}
		spec.meta = func(h rowHighlight) string { return m.taskCourseMeta(n.courseID, h) }
	case nodeBucket:
		spec.title = func(h rowHighlight) string {
			return h.primaryStyle().Render(fmt.Sprintf("%s (%d)", n.bucket, len(ids)))
		}
	case nodeTask:
		now := m.currentTime()
		ageStyle := stRowDim
		if n.task.HasDue && n.task.Due.Before(now) {
			ageStyle = stOverdue
		}
		due := dueCell(n.task, now)
		name := taskDisplayName(n.task.Name)
		spec.title = func(h rowHighlight) string { return h.primaryStyle().Render(name) }
		spec.meta = func(h rowHighlight) string { return h.style(ageStyle).Render(due) }
	case nodeTaskMaterial:
		glyph := "○"
		if n.mat.Downloaded {
			glyph = "●"
		}
		_, style := materialTreeState(n.mat)
		spec.expandable = false
		spec.glyph = glyph
		spec.glyphStyle = style
		spec.prefix = func(h rowHighlight) string { return h.style(stFaint).Render("⁝ ") }
		spec.title = func(h rowHighlight) string { return h.primaryStyle().Render(n.mat.Name) }
	}
	return m.treeRow(spec)
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
		title = fmt.Sprintf("tasks › %s", m.filter)
		right = rightAligned(fmt.Sprintf("%d assignments", matches), hint)
	}
	filterHint := "f filter"
	if m.filter != "all" || m.search != "" {
		filterHint = "esc clear"
	}
	detailHint := "enter detail"
	switch m.detailMode() {
	case detailSplit:
		detailHint = "enter hide"
	case detailReplace:
		detailHint = "esc back"
	}
	hints := []string{"↑↓ move", "t/m/i decide", "c commit", filterHint, detailHint, "? more"}
	leftWidth := width
	if m.detailMode() == detailSplit {
		leftWidth, _ = splitWidths(width)
	}
	rows := m.taskRows(nodes, leftWidth, height)
	rows = m.detailBody(rows, nodes, m.taskCursor, width, height)
	return title, right, hints, rows
}
