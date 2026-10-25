package tui

func (m *App) switchTab(tab Tab) {
	if tab < 0 || tab >= tabCount {
		return
	}
	m.visual = false
	m.dropAllArmed = false
	m.searching = false
	m.search = ""
	m.tabFilters[m.tab] = m.filter
	m.tabExpanded[m.tab] = m.expanded
	if tab == TabCommit && m.tab != TabCommit {
		m.returnTab = m.tab
	}
	m.tab = tab
	m.filter = m.tabFilters[tab]
	if m.filter == "" {
		m.filter = "all"
	}
	if m.tabExpanded[tab] != nil {
		m.expanded = m.tabExpanded[tab]
	} else {
		m.tabExpanded[tab] = m.expanded
	}
	m.clampCursor()
}

func (m *App) move(delta int) {
	switch m.tab {
	case TabTasks:
		m.taskCursor = clampInt(m.taskCursor+delta, 0, maxInt(0, len(m.taskNodes())-1))
		m.extendRange()
	case TabMaterials:
		m.matCursor = clampInt(m.matCursor+delta, 0, maxInt(0, len(m.matNodes())-1))
		m.extendRange()
	case TabSync:
		m.syncCursor = clampInt(m.syncCursor+delta, 0, 5)
	case TabCommit:
		m.commitCursor = clampInt(m.commitCursor+delta, 0, maxInt(0, len(m.commitItems())-1))
	}
}

// clampCursor keeps the per-tab cursors inside whatever the tree currently
// draws, so a stale index never points past the end.
func (m *App) clampCursor() {
	switch {
	case m.tab == TabTasks:
		m.taskCursor = clampInt(m.taskCursor, 0, maxInt(0, len(m.taskNodes())-1))
	case m.tab == TabMaterials:
		m.matCursor = clampInt(m.matCursor, 0, maxInt(0, len(m.matNodes())-1))
	case m.tab == TabSync:
		m.syncCursor = clampInt(m.syncCursor, 0, 5)
	case m.tab == TabCommit:
		m.commitCursor = clampInt(m.commitCursor, 0, maxInt(0, len(m.commitItems())-1))
	}
}

// moveX is horizontal: folding everywhere the trees are. `l` opens the cursor's
// container, `h` closes it.
func (m *App) moveX(delta int) {
	var nodes []node
	var cursor *int
	switch m.tab {
	case TabTasks:
		nodes, cursor = m.taskNodes(), &m.taskCursor
	case TabMaterials:
		nodes, cursor = m.matNodes(), &m.matCursor
	default:
		return
	}
	if *cursor < 0 || *cursor >= len(nodes) {
		return
	}
	n := nodes[*cursor]
	if delta > 0 {
		if n.key != "" && (m.isTaskGroup(n) || m.materialHasChildren(n)) {
			m.expanded[n.key] = true
			m.tabExpanded[m.tab] = m.expanded
		}
		return
	}
	// h/left: fold the cursor's container while it actually shows children,
	// keeping the highlight on it; on leaves and already-folded containers,
	// walk up to the parent row instead.
	if n.key != "" && (m.isTaskGroup(n) || m.materialHasChildren(n)) && hasVisibleChildren(nodes, n.key) {
		m.expanded[n.key] = false
		m.tabExpanded[m.tab] = m.expanded
		return
	}
	if len(n.parents) == 0 {
		return
	}
	parentKey := n.parents[len(n.parents)-1]
	for i, candidate := range nodes {
		if candidate.key == parentKey {
			*cursor = i
			return
		}
	}
}

// hasVisibleChildren reports whether the flat node list currently renders any
// child rows for the node with the given key, i.e. whether folding it would
// hide anything at all right now.
func hasVisibleChildren(nodes []node, key string) bool {
	for _, n := range nodes {
		for _, parent := range n.parents {
			if parent == key {
				return true
			}
		}
	}
	return false
}

func (m *App) toggleSelect() {
	switch m.tab {
	case TabTasks:
		nodes := m.taskNodes()
		if m.taskCursor < 0 || m.taskCursor >= len(nodes) {
			return
		}
		n := nodes[m.taskCursor]
		if n.kind == nodeTaskMaterial {
			return
		}
		if n.kind == nodeTask {
			toggleSet(m.selected, []string{n.task.ID})
			return
		}
		toggleSet(m.selected, m.taskGroupIDs(n))
	case TabMaterials:
		nodes := m.matNodes()
		if m.matCursor < 0 || m.matCursor >= len(nodes) {
			return
		}
		n := nodes[m.matCursor]
		var ids []string
		if n.kind == nodeFile {
			ids = []string{n.mat.ID}
		} else {
			ids = idsOf(m.materialNodeFiles(n))
		}
		toggleSet(m.selectedM, ids)
	}
}

func (m *App) selectAll() {
	switch m.tab {
	case TabTasks:
		for _, n := range m.taskNodes() {
			if n.kind == nodeTask {
				m.selected[n.task.ID] = true
			}
		}
	case TabMaterials:
		for _, mt := range m.materials {
			m.selectedM[mt.ID] = true
		}
	}
}

func (m *App) back() {
	m.selected = map[string]bool{}
	m.selectedM = map[string]bool{}
	m.visual = false
}
