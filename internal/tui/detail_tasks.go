package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/ruyam/kamps/internal/format"
	"github.com/ruyam/kamps/internal/model"
)

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
	case nodeCourse:
		addWrapped(m.fullCourseName(n.courseID), stPaneTitle)
		add("")
		addKV("tasks", fmt.Sprint(len(m.taskGroupIDs(n))), stRow)
		addStates(n)
	case nodeBucket:
		add(stPaneTitle.Render(truncate(n.bucket, width)))
		addKV("tasks", fmt.Sprint(len(m.taskGroupIDs(n))), stRow)
		addStates(n)
	case nodeTask:
		task := n.task
		addWrapped(task.Name, stPaneTitle.Bold(true))
		add("")
		if course, ok := m.courseByID(task.CourseID); ok {
			addWrapped(course.Name, stDim)
		} else {
			addWrapped(m.fullCourseName(task.CourseID), stDim)
		}
		now := m.currentTime()
		due, dueStyle := "none", stDueNone
		if task.HasDue {
			days := dueDays(task, now)
			due = task.Due.Format("2 Jan 2006 15:04") + " · " + readableDeadline(task, m)
			dueStyle = dueStyleFor(dueToneForDays(days, true))
			if task.Overdue(now) {
				due = "! " + due
			}
		}
		addKV("due", due, dueStyle)
		state := task.Triage.String()
		if task.Triage == model.Undecided {
			state = "new"
		}
		addKV("state", flagChar(task.Triage)+" "+state, triageStyle(task.Triage))

		if task.Section != "" {
			addKV("section", sectionLabel(task.Section), stRow)
		}
		if task.Praktikum {
			addKV("kind", "praktikum", stRow)
		}
		if task.EstMinutes > 0 {
			addKV("estimate", fmt.Sprintf("%d min", task.EstMinutes), stRow)
		}
		if task.Goal != "" {
			addWrapped(task.Goal, stRow)
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
			addKV("size", format.HumanSize(material.Size), stRow)
		}
		if material.URL != "" {
			add("")
			add(stFaint.Render("url"))
			add(stRow.Render(truncate(material.URL, width)))
		}
	}
	return lines
}
