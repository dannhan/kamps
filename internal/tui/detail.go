package tui

// detailMode is how the detail pane occupies the body.
type detailMode int

const (
	detailHidden detailMode = iota
	detailSplit
	detailReplace
)

// detailMode resolves the pane against the current size class. The default
// (detailPane = -1) auto-opens only at Wide; enter pins it open, and on
// Standard/Compact the pinned pane takes the whole body instead of a cramped
// split.
func (m *App) detailMode() detailMode {
	if m.tab != TabTasks && m.tab != TabMaterials {
		return detailHidden
	}
	wide := m.sizeClass() == sizeWide
	switch m.detailPane {
	case 0:
		return detailHidden
	case 1:
		if wide {
			return detailSplit
		}
		return detailReplace
	default:
		if wide {
			return detailSplit
		}
		return detailHidden
	}
}

// detailVisible reports whether the selected item's detail pane should be
// drawn. The pane is opt-in on Standard/Compact and auto-open at Wide.
func (m *App) detailVisible() bool {
	return m.detailMode() != detailHidden
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
// single rule between them. Only called in Wide mode.
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

// detailBody applies the resolved pane mode to an already-rendered tree.
func (m *App) detailBody(tree []string, nodes []node, cursor, width, height int) []string {
	switch m.detailMode() {
	case detailSplit:
		return m.splitTreeRows(tree, nodes, cursor, width, height)
	case detailReplace:
		return m.detailRows(nodes, cursor, width, height)
	}
	return tree
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
	if m.tab == TabTasks {
		if n, ok := m.cursorTaskNode(); ok {
			if n.kind == nodeActNow || n.kind == nodeBucket && n.bucket == "stale overdue" {
				m.cancelTriageArm()
				if m.expanded == nil {
					m.expanded = make(map[string]bool)
				}
				m.expanded[n.key] = !m.taskIsOpen(n.key, taskNodeDefaultOpen(n))
				m.tabExpanded[m.tab] = m.expanded
				return
			}
		}
	}
	switch m.tab {
	case TabTasks, TabMaterials:
		m.toggleDetailPane()
	}
}
