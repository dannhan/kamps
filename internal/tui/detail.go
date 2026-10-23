package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/ruyam/kamps/internal/model"
)

// detailMinWidth is where the tree detail pane starts showing on its own.
// Below this the tree keeps every column until the pane is toggled on.
const detailMinWidth = 100

// detailVisible reports whether the selected item's detail pane should be
// drawn: an explicit toggle wins, otherwise it follows the terminal width.
func (m *App) detailVisible() bool {
	if m.tab != TabTasks && m.tab != TabMaterials {
		return false
	}
	if m.detailPane >= 0 {
		return m.detailPane == 1
	}
	return m.width >= detailMinWidth
}

// toggleDetailPane pins the detail pane open or closed.
func (m *App) toggleDetailPane() {
	if m.detailVisible() {
		m.detailPane = 0
	} else {
		m.detailPane = 1
	}
}

// splitWidths divides the body into a tree column and a detail column with a
// single rule between them.
func splitWidths(width int) (left, right int) {
	left = width * 62 / 100
	right = width - left - 1
	if right < 24 {
		right = 24
		left = width - right - 1
	}
	if left < 1 {
		left = 1
	}
	return left, right
}

// splitTreeRows draws the tree on the left and the focused node's details on
// the right.
func (m *App) splitTreeRows(tree []string, nodes []node, cursor, width, height int) []string {
	leftWidth, rightWidth := splitWidths(width)
	detail := m.detailRows(nodes, cursor, rightWidth, height)
	divider := stTreeLine.Render("│")
	rows := make([]string, 0, height)
	for i := 0; i < height; i++ {
		left, right := "", ""
		if i < len(tree) {
			left = tree[i]
		}
		if i < len(detail) {
			right = detail[i]
		}
		rows = append(rows, fitLine(left, leftWidth)+divider+fitLine(right, rightWidth))
	}
	return rows
}

// detailRows renders the right column, padding it to the pane height and
// marking a clipped description with an ellipsis.
func (m *App) detailRows(nodes []node, cursor, width, height int) []string {
	if height <= 0 {
		return nil
	}
	inner := maxInt(1, width-2)
	var content []string
	if cursor >= 0 && cursor < len(nodes) {
		content = m.detailLines(nodes[cursor], inner)
	}
	if len(content) > height {
		content = content[:height]
		content[height-1] = ansiTruncate(content[height-1], maxInt(0, inner-1)) + stFaint.Render("…")
	}
	rows := make([]string, 0, height)
	for i := 0; i < height; i++ {
		line := ""
		if i < len(content) {
			line = content[i]
		}
		rows = append(rows, " "+fitLine(line, inner)+" ")
	}
	return rows
}

// detailLines is the content of the detail pane for one tree node.
func (m *App) detailLines(n node, width int) []string {
	if m.tab == TabMaterials {
		return m.materialDetailLines(n, width)
	}
	var lines []string
	add := func(line string) { lines = append(lines, line) }
	addKV := func(key, value string, style lipgloss.Style) {
		const keyWidth = 8
		value = truncate(value, maxInt(1, width-keyWidth-1))
		add(stFaint.Render(pad(key, keyWidth, false)) + " " + style.Render(value))
	}
	addStates := func(group node) {
		add("")
		add(stFaint.Render("states"))
		add(truncate(formatTaskCounts(m.taskNodeCounts(group), rowHighlight{}), width))
	}
	addWrapped := func(text string, style lipgloss.Style) {
		for _, line := range wrapCell(text, width) {
			add(style.Render(line))
		}
	}

	switch n.kind {
	case nodeTerm:
		add(stPaneTitle.Render(truncate(n.term, width)))
		addKV("tasks", fmt.Sprint(len(m.taskGroupIDs(n))), stRow)
		addStates(n)
	case nodeCourse:
		short := m.shortCode(n.courseID)
		if course, ok := m.courseByID(n.courseID); ok {
			add(stPaneTitle.Render(truncate(course.Name, width)))
			add(stFaint.Render(truncate(course.Code+" · "+course.Term, width)))
			if course.Short != "" {
				short = course.Short
			}
		} else {
			add(stPaneTitle.Render(short))
		}
		add("")
		addKV("short", short, stCode)
		addKV("tasks", fmt.Sprint(len(m.taskGroupIDs(n))), stRow)
		addStates(n)
	case nodeBucket:
		add(stPaneTitle.Render(truncate(n.bucket, width)))
		addKV("tasks", fmt.Sprint(len(m.taskGroupIDs(n))), stRow)
		addStates(n)
	case nodeTask:
		task := n.task
		add(stCode.Render(task.ID))
		addWrapped(task.Name, stRow)
		add("")
		if course, ok := m.courseByID(task.CourseID); ok {
			addKV("course", course.Name, stRow)
		} else {
			addKV("course", m.shortCode(task.CourseID), stRow)
		}
		state := task.Triage.String()
		if task.Triage == model.Undecided {
			state = "new"
		}
		addKV("state", flagChar(task.Triage)+" "+state, triageStyle(task.Triage))
		now := time.Now()
		due, dueStyle := "none", stRow
		if task.HasDue {
			due = task.Due.Format("2006-01-02") + " · " + dueAge(task, now)
			if task.Overdue(now) {
				dueStyle = stOverdue
			}
		}
		addKV("due", due, dueStyle)
		if task.Praktikum {
			addKV("kind", "praktikum", stRow)
		}
		if task.EstMinutes > 0 {
			addKV("estimate", fmt.Sprintf("%d min", task.EstMinutes), stRow)
		}
		if task.Goal != "" {
			addKV("goal", task.Goal, stRow)
		}
		materials := m.assignmentMaterials(task)
		if len(materials) > 0 {
			add("")
			add(stFaint.Render("materials"))
			for _, material := range materials {
				state, style := materialTreeState(material)
				nameWidth := maxInt(1, width-lipgloss.Width(state)-4)
				add(" " + stFaint.Render("⁝") + " " + stRow.Render(truncate(material.Name, nameWidth)) + "  " + style.Render(state))
			}
		}
	case nodeTaskMaterial:
		material := n.mat
		add(stCode.Render(material.ID))
		addWrapped(material.Name, stRow)
		add("")
		addKV("course", m.shortCode(material.CourseID), stRow)
		state, style := materialTreeState(material)
		addKV("state", state, style)
		if material.Kind != "" {
			addKV("kind", material.Kind, stRow)
		}
		if material.Section != "" {
			addKV("section", material.Section, stRow)
		}
		if material.Size > 0 {
			addKV("size", humanSize(material.Size), stRow)
		}
		if material.URL != "" {
			add("")
			add(stFaint.Render("url"))
			add(stRow.Render(truncate(material.URL, width)))
		}
	}
	return lines
}

// materialDetailLines is the right-hand pane on the Materials tab.
func (m *App) materialDetailLines(n node, width int) []string {
	var lines []string
	add := func(line string) { lines = append(lines, line) }
	addKV := func(key, value string, style lipgloss.Style) {
		const keyWidth = 8
		value = truncate(value, maxInt(1, width-keyWidth-1))
		add(stFaint.Render(pad(key, keyWidth, false)) + " " + style.Render(value))
	}
	addWrapped := func(text string, style lipgloss.Style) {
		for _, line := range wrapCell(text, width) {
			add(style.Render(line))
		}
	}
	addFiles := func(files []model.Material) {
		local := 0
		for _, material := range files {
			if material.Downloaded {
				local++
			}
		}
		addKV("files", fmt.Sprint(len(files)), stRow)
		addKV("local", fmt.Sprintf("%d of %d", local, len(files)), stRow)
		addKV("size", materialSizeSummary(files), stRow)
	}

	switch n.kind {
	case nodeCourse:
		if course, ok := m.courseByID(n.courseID); ok {
			add(stPaneTitle.Render(truncate(course.Name, width)))
			add(stFaint.Render(truncate(course.Code+" · "+course.Term, width)))
		} else {
			add(stPaneTitle.Render(m.shortCode(n.courseID)))
		}
		add("")
		addFiles(m.courseFiles(n.courseID))
	case nodeMaterialBucket:
		add(stPaneTitle.Render(n.bucket + " files"))
		add("")
		addFiles(m.materialNodeFiles(n))
	case nodeFile:
		material := n.mat
		add(stCode.Render(material.ID))
		addWrapped(material.Name, stRow)
		add("")
		if course, ok := m.courseByID(material.CourseID); ok {
			addKV("course", course.Name, stRow)
		} else {
			addKV("course", m.shortCode(material.CourseID), stRow)
		}
		addKV("section", sectionLabel(material.Section), stRow)
		if material.Kind != "" {
			addKV("kind", material.Kind, stRow)
		}
		if material.Size > 0 {
			addKV("size", humanSize(material.Size), stRow)
		}
		state, style := materialTreeState(material)
		addKV("state", state, style)
		if assignment := m.assignmentForMaterial(material); assignment != nil {
			addKV("linked", assignment.ID+" · "+assignment.Name, stRow)
		}
		if material.URL != "" {
			add("")
			add(stFaint.Render("url"))
			add(stRow.Render(truncate(material.URL, width)))
		}
	case nodeMaterialAssignment:
		task := n.task
		add(stCode.Render(task.ID))
		addWrapped(task.Name, stRow)
		add("")
		if course, ok := m.courseByID(task.CourseID); ok {
			addKV("course", course.Name, stRow)
		} else {
			addKV("course", m.shortCode(task.CourseID), stRow)
		}
		state := task.Triage.String()
		if task.Triage == model.Undecided {
			state = "new"
		}
		addKV("state", flagChar(task.Triage)+" "+state, triageStyle(task.Triage))
		now := time.Now()
		due, dueStyle := "none", stRow
		if task.HasDue {
			due = task.Due.Format("2006-01-02") + " · " + dueAge(task, now)
			if task.Overdue(now) {
				dueStyle = stOverdue
			}
		}
		addKV("due", due, dueStyle)
	}
	return lines
}

// materialSizeSummary adds up the sizes we know about and says how many files
// have no size, instead of pretending the total is complete.
func materialSizeSummary(files []model.Material) string {
	var total int64
	unknown := 0
	for _, material := range files {
		if material.Size > 0 {
			total += material.Size
		} else {
			unknown++
		}
	}
	parts := []string{}
	if total > 0 {
		parts = append(parts, humanSize(total))
	}
	if unknown > 0 {
		parts = append(parts, fmt.Sprintf("%d unknown", unknown))
	}
	if len(parts) == 0 {
		return "—"
	}
	return strings.Join(parts, " · ")
}
