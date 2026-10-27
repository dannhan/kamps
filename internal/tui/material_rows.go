package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/ruyam/kamps/internal/model"
)

func materialTreeState(material model.Material) (string, lipgloss.Style) {
	if material.Downloaded {
		return "downloaded", materialStateStyle(true)
	}
	return "new", materialStateStyle(false)
}

func (m *App) matTreeRow(nodes []node, index, width int, selected bool) string {
	n := nodes[index]
	if n.kind == nodeSpacer {
		return m.treeRow(treeRowSpec{nodes: nodes, index: index, width: width, blank: true})
	}
	ids := m.materialNodeIDs(n)
	parent := m.materialHasChildren(n)
	open := n.key != "" && m.isOpen(n.key)
	spec := treeRowSpec{
		nodes:      nodes,
		index:      index,
		width:      width,
		cursor:     index == m.matCursor,
		selected:   selected,
		partial:    !selected && partialSelected(m.selectedM, ids),
		expandable: parent,
		open:       open,
	}
	switch n.kind {
	case nodeCourse:
		name := m.shortCode(n.courseID)
		if n.courseID == localCourseID {
			name = "local library"
		} else if course, ok := m.courseByID(n.courseID); ok && course.Name != "" {
			name = course.Name
		}
		spec.title = func(h rowHighlight) string {
			return h.style(m.courseStyle(n.courseID).Bold(true)).Render(name + " ")
		}
		spec.meta = func(h rowHighlight) string { return m.materialCourseMeta(n.courseID, h) }
		spec.rule = width >= courseRuleMinWidth
	case nodeMaterialWeek, nodeMaterialSection:
		count := len(m.materialNodeFiles(n))
		spec.title = func(h rowHighlight) string {
			return h.primaryStyle().Render(fmt.Sprintf("%s (%d)", n.section, count))
		}
	case nodeMaterialFolder:
		count := len(m.materialNodeFiles(n))
		spec.reserveGlyph = true
		spec.title = func(h rowHighlight) string {
			return h.primaryStyle().Render(fmt.Sprintf("%s/ (%d)", n.mat.Folder, count))
		}
	case nodeFile:
		glyph := "○"
		if n.mat.Downloaded {
			glyph = "●"
		}
		_, style := materialTreeState(n.mat)
		spec.reserveGlyph = true
		spec.glyph = glyph
		spec.glyphStyle = style
		spec.title = func(h rowHighlight) string { return h.primaryStyle().Render(n.mat.Name) }
	case nodeMaterialAssignment:
		spec.reserveGlyph = true
		spec.glyph = flagChar(n.task.Triage)
		spec.glyphStyle = triageStyle(n.task.Triage)
		spec.prefix = func(h rowHighlight) string { return h.style(stFaint).Render("⁝ ") }
		name := taskDisplayName(n.task.Name)
		spec.title = func(h rowHighlight) string {
			return h.primaryStyle().Render("assignment: "+name) + h.style(stRowDim).Render(" · "+n.task.ID)
		}
	}
	return m.treeRow(spec)
}

func (m *App) materialHasChildren(n node) bool {
	switch n.kind {
	case nodeCourse:
		return len(m.materialNodeFiles(n)) > 0
	case nodeMaterialWeek, nodeMaterialSection, nodeMaterialFolder:
		return len(m.materialNodeFiles(n)) > 0
	}
	return false
}

func (m *App) materialsContent(width, height int) (string, string, []string, []string) {
	return m.workspaceContent(width, height)
}

func (m *App) materialRows(nodes []node, width, height int) []string {
	start, end := windowRange(len(nodes), m.matCursor, height)
	rows := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		n := nodes[i]
		rows = append(rows, m.matTreeRow(nodes, i, width, m.materialNodeSelected(n)))
	}
	if len(rows) == 0 {
		rows = append(rows, stEmpty.Render("no materials"))
	}
	return rows
}
