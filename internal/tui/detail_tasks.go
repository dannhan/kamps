package tui

import (
	"fmt"
	"time"

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
