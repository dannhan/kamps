package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/ruyam/kamps/internal/model"
)

func materialTreeState(material model.Material) (string, lipgloss.Style) {
	if material.Downloaded {
		return "downloaded", stDone
	}
	return "new", stFlagNew
}

// matRowRightPad mirrors taskRowRightPad: the status labels and long names
// stop two columns short of the divider and the terminal edge. The final pad
// carries the row band, so a cursor row stays one continuous strip.
const matRowRightPad = 2

func (m *App) matTreeRow(nodes []node, index, width int, selected bool) string {
	n := nodes[index]
	h := rowHighlight{cursor: index == m.matCursor, mark: selected}
	background := h.style
	prefix := h.gutter() + background(stFaint).Render(treeGuides(nodes, index))
	inner := maxInt(1, width-matRowRightPad)
	parent := m.materialHasChildren(n)
	open := n.key != "" && m.isOpen(n.key)
	lead := ""
	if parent {
		lead = h.containerLead(open, selected)
	}
	content := ""
	switch n.kind {
	case nodeCourse:
		name := m.shortCode(n.courseID)
		if course, ok := m.courseByID(n.courseID); ok && course.Name != "" {
			name = course.Name
		}
		content = lead + background(stCode.Bold(true)).Render(name)
	case nodeMaterialBucket:
		count := 0
		for _, material := range m.courseFiles(n.courseID) {
			if (n.bucket == "downloaded") == material.Downloaded {
				count++
			}
		}
		content = lead + background(stRow).Render(fmt.Sprintf("%s (%d)", n.bucket, count))
	case nodeFile:
		glyph, stateStyle := materialTreeState(n.mat)
		state := "○"
		if glyph == "downloaded" {
			state = "●"
		}
		caret := ""
		if parent {
			caret = h.style(stFaint).Render(caretGlyph(open)) + h.style(stRow).Render(" ")
		}
		nameWidth := maxInt(1, inner-lipgloss.Width(prefix)-12-lipgloss.Width(caret))
		content = background(stateStyle).Render(state) +
			background(stRow).Render(" ") + h.markCell(selected) + background(stRow).Render(" ") + caret +
			background(stRow).Render(ansi.Truncate(n.mat.Name, nameWidth, "…")) +
			background(stateStyle).Render(" "+glyph)
	case nodeMaterialAssignment:
		content = background(triageStyle(n.task.Triage)).Render(flagChar(n.task.Triage)) + background(stFaint).Render(" ⁝ ") + background(stRow).Render("assignment: "+n.task.Name) + background(stRowDim).Render(" · "+n.task.ID)
	}
	if (n.kind == nodeCourse || n.kind == nodeMaterialBucket || n.kind == nodeFile) && !open && parent {
		content += background(stRowDim).Render("  ⋯")
	}
	row := prefix + content
	row = ansiTruncate(row, inner)
	return row + background(stRow).Render(strings.Repeat(" ", maxInt(0, width-lipgloss.Width(row))))
}

func (m *App) materialHasChildren(n node) bool {
	switch n.kind {
	case nodeCourse:
		return len(m.courseFiles(n.courseID)) > 0
	case nodeMaterialBucket:
		for _, material := range m.courseFiles(n.courseID) {
			if n.bucket == "downloaded" && material.Downloaded || n.bucket == "new" && !material.Downloaded {
				return true
			}
		}
	case nodeFile:
		return m.assignmentForMaterial(n.mat) != nil
	}
	return false
}

func (m *App) materialsContent(width, height int) (string, string, []string, []string) {
	nodes := m.matNodes()
	matches := len(m.visibleMaterials())
	hint := scrollHint(len(nodes), m.matCursor, height)
	var title, right string
	if m.searchActive() {
		title = m.searchTitle()
		right = rightAligned(fmt.Sprintf("%d %s", matches, plural(matches, "match", "matches")), hint)
	} else {
		title = fmt.Sprintf("materials · %d files · %d local", len(m.materials), m.downloadedMaterials())
		right = hint
	}
	detailHint := "enter detail"
	if m.detailVisible() {
		detailHint = "enter hide"
	}
	hints := []string{"↑↓ move", "h/l fold", "space select", detailHint, "d download"}
	if m.filter != "all" {
		hints = append(hints, "esc clear")
	}
	leftWidth := width
	if m.detailVisible() {
		leftWidth, _ = splitWidths(width)
	}
	rows := m.materialRows(nodes, leftWidth, height)
	if m.detailVisible() {
		rows = m.splitTreeRows(rows, nodes, m.matCursor, width, height)
	}
	return title, right, hints, rows
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
