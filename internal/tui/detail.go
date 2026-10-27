package tui

// detailMode is how the detail pane occupies the body.
type detailMode int

const (
	detailHidden detailMode = iota
	detailSplit
	detailReplace
)

// detailMode reserves a split only when both content and inspector stay readable.
// At narrower sizes the focused inspector replaces the list.
func (m *App) detailMode() detailMode {
	if m.tab != TabTasks && m.tab != TabMaterials {
		return detailHidden
	}
	wide := m.width >= 140 && m.height >= 24
	switch m.detailPane {
	case 0:
		return detailHidden
	case 1:
		if wide {
			return detailSplit
		}
		return detailReplace
	default:
		return detailHidden
	}
}

// detailVisible reports whether the selected item's detail pane should be
// drawn. The pane is opt-in at every terminal size.
func (m *App) detailVisible() bool {
	return m.detailMode() != detailHidden
}

// toggleDetailPane pins the detail pane open or closed.
func (m *App) toggleDetailPane() {
	if m.detailVisible() && !(m.detailMode() == detailReplace && m.paneFocus != 2) {
		m.detailPane = 0
		m.paneFocus = 0
	} else {
		m.detailPane = 1
		m.detailScroll = 0
		m.paneFocus = 2
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
	offset := clampInt(m.detailScroll, 0, maxInt(0, len(content)-height))
	content = content[offset:]
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
	if m.paneFocus == 2 {
		m.toggleDetailPane()
		return
	}
	if m.tab == TabTasks {
		n, ok := m.cursorTaskNode()
		if !ok {
			return
		}
		if n.kind == nodeTaskMore {
			m.expanded[taskShortlistKey] = !m.expanded[taskShortlistKey]
			for i, candidate := range m.taskNodes() {
				if candidate.key == n.key {
					m.taskCursor = i
					break
				}
			}
			m.clampCursor()
			return
		}
		if m.isTaskGroup(n) {
			m.expanded[n.key] = !m.taskGroupOpen(n)
			m.cancelTriageArm()
			m.visual = false
			return
		}
		if n.kind != nodeTask {
			return
		}
	}
	if m.tab == TabMaterials {
		nodes := m.matNodes()
		if m.matCursor < len(nodes) && m.materialHasChildren(nodes[m.matCursor]) {
			n := nodes[m.matCursor]
			m.expanded[n.key] = !m.isOpen(n.key)
			return
		}
	}
	if m.tab == TabTasks || m.tab == TabMaterials {
		m.toggleDetailPane()
	}
}

func (m *App) detailMaxScroll() int {
	width := m.width
	if width >= 100 {
		width -= workspaceSidebarWidth(width) + 1
	}
	if m.detailMode() == detailSplit {
		_, width = splitWidths(width)
	}
	nodes, cursor := m.taskNodes(), m.taskCursor
	if m.tab == TabMaterials {
		nodes, cursor = m.matNodes(), m.matCursor
	}
	if cursor < 0 || cursor >= len(nodes) {
		return 0
	}
	return maxInt(0, len(m.detailLines(nodes[cursor], maxInt(1, width-2)))-bodyHeightFor(m.width, m.height))
}
