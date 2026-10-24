package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/ruyam/kamps/internal/format"
	"github.com/ruyam/kamps/internal/model"
)

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
			addKV("size", format.HumanSize(material.Size), stRow)
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
		parts = append(parts, format.HumanSize(total))
	}
	if unknown > 0 {
		parts = append(parts, fmt.Sprintf("%d unknown", unknown))
	}
	if len(parts) == 0 {
		return "—"
	}
	return strings.Join(parts, " · ")
}
