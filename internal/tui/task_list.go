package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ruyam/kamps/internal/model"
)

func (m *App) fullCourseName(id int) string {
	if course, ok := m.courseByID(id); ok {
		if course.Name != "" {
			return course.Name
		}
		if course.Short != "" {
			return course.Short
		}
	}
	return "Unnamed course"
}

// taskBlock is an indivisible visual item. Long names wrap, with the complete
// name available in details. Course and deadline remain next to the task.
func (m *App) taskBlock(n node, index, width int) []string {
	if n.kind == nodeSpacer {
		return []string{fitLine("", width)}
	}
	if n.kind == nodeTaskHeading {
		style := stDim
		if n.bucket == "Needs attention" || n.bucket == "Courses" {
			style = stSectionTitle
		}
		indent := 5 + 2*len(n.parents)
		return []string{fitLine(strings.Repeat(" ", indent)+style.Render(n.bucket), width)}
	}
	h := rowHighlight{cursor: index == m.taskCursor && m.paneFocus != 1}
	if n.kind == nodeTask {
		h.mark = m.selected[n.task.ID]
		indent := 5 + 2*len(n.parents)
		inner := maxInt(1, width-indent-1)
		deadline := readableDeadline(n.task, m)
		deadlineStyle := stDim
		if n.task.Triage == model.Undecided {
			deadlineStyle = dueStyleFor(dueToneForDays(dueDays(n.task, m.currentTime()), n.task.HasDue))
		}
		if n.task.Triage != model.Undecided {
			deadline += " · " + n.task.Triage.String()
		}
		inCourseGroup := len(n.parents) > 0 && n.parents[0] == courseKey(n.courseID)
		titleWidth := inner
		inlineDeadline := inCourseGroup && inner-lipgloss.Width(deadline)-3 >= 12
		if inlineDeadline {
			titleWidth -= lipgloss.Width(deadline) + 3
		}
		titles := wrapCell(taskDisplayName(n.task.Name), titleWidth)
		if len(titles) > 3 {
			titles = titles[:3]
			titles[2] = truncate(titles[2], maxInt(1, titleWidth-1)) + "…"
		}
		var rows []string
		for i, title := range titles {
			prefix := h.blanks(indent)
			if i == 0 {
				prefix = h.gutter() + h.markCell() + h.blanks(indent-2)
			}
			content := h.primaryStyle().Render(title)
			if inlineDeadline && i == len(titles)-1 {
				content += h.style(stDim).Render(" · ") + h.style(deadlineStyle).Render(deadline)
			}
			rows = append(rows, fitLine(prefix+content+h.blanks(maxInt(0, inner-lipgloss.Width(content))+1), width))
		}
		if inlineDeadline {
			return rows
		}
		metadata := ""
		if !m.courseScoped && (len(n.parents) == 0 || n.parents[0] != courseKey(n.courseID)) {
			metadata = m.fullCourseName(n.courseID)
		}
		metaLines := wrapCell(metadata, inner)
		if metadata == "" {
			metaLines = nil
		}
		if len(metaLines) > 2 {
			metaLines = metaLines[:2]
			metaLines[1] = truncate(metaLines[1], maxInt(1, inner-1)) + "…"
		}
		if len(metaLines) > 0 && lipgloss.Width(metaLines[len(metaLines)-1])+3+lipgloss.Width(deadline) <= inner {
			last := len(metaLines) - 1
			for _, line := range metaLines[:last] {
				rows = append(rows, fitLine(h.blanks(indent)+h.style(m.courseStyle(n.courseID)).Render(pad(line, inner, false))+h.gap(), width))
			}
			content := h.style(m.courseStyle(n.courseID)).Render(metaLines[last]+" · ") + h.style(deadlineStyle).Render(deadline)
			rows = append(rows, fitLine(h.blanks(indent)+content+h.blanks(inner-lipgloss.Width(content)+1), width))
		} else {
			for _, line := range metaLines {
				rows = append(rows, fitLine(h.blanks(indent)+h.style(m.courseStyle(n.courseID)).Render(pad(line, inner, false))+h.gap(), width))
			}
			rows = append(rows, fitLine(h.blanks(indent)+h.style(deadlineStyle).Render(pad(deadline, inner, false))+h.gap(), width))
		}
		return rows
	}
	label := n.bucket
	if n.kind == nodeTaskMore {
		if m.expanded[taskShortlistKey] {
			label = "Show fewer"
		}
		return []string{paneRow("  "+label+" · Enter", width, h)}
	}
	if n.kind == nodeCourse {
		label = m.fullCourseName(n.courseID)
	}
	ids := m.taskGroupIDs(n)
	h.mark = allSelected(m.selected, ids)
	h.partial = !h.mark && partialSelected(m.selected, ids)
	prefix := h.gutter() + h.markCell() + h.blanks(1+2*len(n.parents)) + h.style(stDim).Render(caretGlyph(m.taskGroupOpen(n))+" ")
	label += fmt.Sprintf(" · %d", len(ids))
	if n.kind == nodeCourse {
		undecided := 0
		for _, id := range ids {
			if a := m.byID(id); a != nil && a.Triage == model.Undecided {
				undecided++
			}
		}
		label = fmt.Sprintf("%s · %d undecided", m.fullCourseName(n.courseID), undecided)
	}
	style := h.primaryStyle()
	if n.kind == nodeCourse {
		style = h.style(m.courseStyle(n.courseID)).Bold(true)
	}
	return []string{fitLine(prefix+style.Render(pad(label, maxInt(1, width-lipgloss.Width(prefix)), false)), width)}
}

// taskListWindow windows complete blocks by terminal lines, never splitting a
// task from its course/deadline. The opening window starts at the section title.
func (m *App) taskListWindow(nodes []node, width, height int) ([]string, int, int) {
	if height <= 0 {
		return nil, 0, 0
	}
	blocks := make([][]string, len(nodes))
	for i, n := range nodes {
		blocks[i] = m.taskBlock(n, i, width)
		if len(blocks[i]) > height { // Very short terminals: retain metadata and indicate omitted title lines.
			blocks[i] = append([]string{fitLine(truncate(blocks[i][0], maxInt(1, width-1))+"…", width)}, blocks[i][len(blocks[i])-height+1:]...)
		}
	}
	return m.courseContextWindow(nodes, blocks, m.taskCursor, width, height)
}

// Keep ownership visible when a course heading has scrolled above the window.
func (m *App) courseContextWindow(nodes []node, blocks [][]string, cursor, width, height int) ([]string, int, int) {
	rows, start, end := blockWindow(blocks, cursor, height)
	needsContext := func(index int) bool {
		if index <= 0 || index >= len(nodes) {
			return false
		}
		for _, parent := range nodes[index].parents {
			for _, candidate := range nodes[:index] {
				if candidate.kind == nodeCourse && candidate.key == parent {
					return true
				}
			}
		}
		return false
	}
	if height < 4 || !needsContext(start) {
		return rows, start, end
	}
	contextRows, contextStart, contextEnd := blockWindow(blocks, cursor, height-1)
	if !needsContext(contextStart) {
		return contextRows, contextStart, contextEnd
	}
	courseID := nodes[contextStart].courseID
	context := fitLine("     "+m.courseStyle(courseID).Render(m.fullCourseName(courseID)+" · continued"), width)
	return append([]string{context}, contextRows...), contextStart, contextEnd
}

// blockWindow keeps focus visible and never detaches metadata from its title.
func blockWindow(blocks [][]string, cursor, height int) ([]string, int, int) {
	if height <= 0 {
		return nil, 0, 0
	}
	cursor = clampInt(cursor, 0, maxInt(0, len(blocks)-1))
	start, used := 0, 0
	for i := 0; i <= cursor && i < len(blocks); i++ {
		used += len(blocks[i])
	}
	for used > height && start < cursor {
		used -= len(blocks[start])
		start++
	}
	rows := []string{}
	end := start
	for end < len(blocks) && len(rows)+len(blocks[end]) <= height {
		rows = append(rows, blocks[end]...)
		end++
	}
	return rows, start, end
}

// Prefer the same appearance after a decision. If it disappears, move to the
// next task in that section, then the preceding task, before falling back.
func (m *App) restoreAfterDecision(before []node, cursor int) {
	after := m.taskNodes()
	indexes := map[string]int{}
	for i, n := range after {
		indexes[n.key] = i
	}
	if cursor < 0 || cursor >= len(before) {
		m.clampCursor()
		return
	}
	focused := before[cursor]
	sameParents := func(n node) bool { return strings.Join(n.parents, "/") == strings.Join(focused.parents, "/") }
	if index, ok := indexes[focused.key]; ok && sameParents(after[index]) {
		m.taskCursor = index
		return
	}
	sameSection := func(n node) bool {
		return n.kind == nodeTask && strings.Join(n.parents, "/") == strings.Join(focused.parents, "/")
	}
	for i := cursor + 1; i < len(before); i++ {
		if sameSection(before[i]) {
			if index, ok := indexes[before[i].key]; ok && sameParents(after[index]) {
				m.taskCursor = index
				return
			}
		}
	}
	for i := cursor - 1; i >= 0; i-- {
		if sameSection(before[i]) {
			if index, ok := indexes[before[i].key]; ok && sameParents(after[index]) {
				m.taskCursor = index
				return
			}
		}
	}
	for i, n := range after {
		if n.kind == nodeTask && sameSection(n) {
			m.taskCursor = i
			return
		}
	}
	m.clampCursor()
}
