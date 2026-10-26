package tui

func (m *App) switchTab(tab Tab) {
	if tab < 0 || tab >= tabCount {
		return
	}
	m.cancelTriageArm()
	m.visual = false
	m.dropAllArmed = false
	m.openAllArmed = false
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
		m.taskCursor = movedIndex(m.taskNodes(), m.taskCursor, delta)
		m.extendRange()
	case TabMaterials:
		m.matCursor = movedIndex(m.matNodes(), m.matCursor, delta)
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
		m.taskCursor = selectableIndex(m.taskNodes(), m.taskCursor)
	case m.tab == TabMaterials:
		m.matCursor = selectableIndex(m.matNodes(), m.matCursor)
	case m.tab == TabSync:
		m.syncCursor = clampInt(m.syncCursor, 0, 5)
	case m.tab == TabCommit:
		m.commitCursor = clampInt(m.commitCursor, 0, maxInt(0, len(m.commitItems())-1))
	}
}

// selectableNode reports whether a row can hold the cursor. Spacers and the
// dim "all in Act now" course placeholder are landmarks only.
func selectableNode(n node) bool {
	return n.kind != nodeSpacer && n.kind != nodeCoursePlaceholder
}

// selectableIndex clamps index into the list and, when it lands on a landmark,
// moves to the nearest selectable row.
func selectableIndex(nodes []node, index int) int {
	if len(nodes) == 0 {
		return 0
	}
	index = clampInt(index, 0, len(nodes)-1)
	if selectableNode(nodes[index]) {
		return index
	}
	for i := index + 1; i < len(nodes); i++ {
		if selectableNode(nodes[i]) {
			return i
		}
	}
	for i := index - 1; i >= 0; i-- {
		if selectableNode(nodes[i]) {
			return i
		}
	}
	return 0
}

// movedIndex applies delta and skips landmark rows in the direction of travel.
// Landmarks remain rendered but can never hold the cursor.
func movedIndex(nodes []node, index, delta int) int {
	if len(nodes) == 0 {
		return 0
	}
	if delta == 0 {
		return selectableIndex(nodes, index)
	}
	index = clampInt(index+delta, 0, len(nodes)-1)
	if delta > 0 {
		for index < len(nodes)-1 && !selectableNode(nodes[index]) {
			index++
		}
	} else {
		for index > 0 && !selectableNode(nodes[index]) {
			index--
		}
	}
	if !selectableNode(nodes[index]) {
		// Degenerate list: move the other way to find something selectable.
		if delta > 0 {
			for index > 0 && !selectableNode(nodes[index]) {
				index--
			}
		} else {
			for index < len(nodes)-1 && !selectableNode(nodes[index]) {
				index++
			}
		}
	}
	return index
}

// toggleShowAll switches between the summary tree (Act now + stale buckets) and
// the full course tree. The cursor follows the focused stable key when possible.
func (m *App) toggleShowAll() {
	if m.tab != TabTasks {
		return
	}
	m.cancelTriageArm()
	m.visual = false
	key := ""
	var fallbacks []string
	nodes := m.taskNodes()
	if m.taskCursor >= 0 && m.taskCursor < len(nodes) {
		n := nodes[m.taskCursor]
		key = n.key
		fallbacks = taskFallbackKeys(n)
	}
	m.showAll = !m.showAll
	keys := append([]string{key}, fallbacks...)
	for _, candidate := range keys {
		if candidate == "" {
			continue
		}
		for i, n := range m.taskNodes() {
			if n.key == candidate && selectableNode(n) {
				m.taskCursor = i
				return
			}
		}
	}
	m.clampCursor()
}

func (m *App) cancelTriageArm() {
	m.triageArmed = false
	m.armedTriageIDs = nil
	m.armedTriageHidden = 0
}

// moveTop and moveBottom jump to the first or last row in the active list.
func (m *App) moveTop() { m.move(-1 << 20) }

func (m *App) moveBottom() { m.move(1 << 20) }

// movePage moves by roughly half a screen, matching ctrl+d/ctrl+u conventions.
func (m *App) movePage(direction int) {
	step := maxInt(1, (m.height-5)/2)
	m.move(direction * step)
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
	if !selectableNode(n) {
		return
	}
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
	m.cancelTriageArm()
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
	m.cancelTriageArm()
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
	m.cancelTriageArm()
	m.selected = map[string]bool{}
	m.selectedM = map[string]bool{}
	m.visual = false
	m.openAllArmed = false
}
