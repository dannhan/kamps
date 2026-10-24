package tui

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

func (m *App) enter() {
	switch m.tab {
	case TabTasks, TabMaterials:
		m.toggleDetailPane()
	}
}
