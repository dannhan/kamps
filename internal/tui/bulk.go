package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ruyam/kamps/internal/model"
)

func (m *App) selectionSet() map[string]bool {
	if m.tab == TabTasks {
		return m.selected
	}
	return m.selectedM
}

// partialSelected reports whether some, but not all, ids are selected.
func partialSelected(set map[string]bool, ids []string) bool {
	if len(ids) < 2 {
		return false
	}
	on := 0
	for _, id := range ids {
		if set[id] {
			on++
		}
	}
	return on > 0 && on < len(ids)
}

func (m *App) selectedCount() int {
	switch m.tab {
	case TabTasks:
		return len(m.selected)
	case TabMaterials:
		return len(m.selectedM)
	}
	return 0
}

// selectionLabel is the persistent "N selected" readout. It is rendered in the
// footer so a bulk action always has a visible count.
func (m *App) selectionLabel() string {
	n := m.selectedCount()
	if n == 0 {
		return ""
	}
	return stSelMark.Render(fmt.Sprintf("%d selected", n))
}

func (m *App) selectionNodes() ([]node, int) {
	if m.tab == TabTasks {
		return m.taskNodes(), m.taskCursor
	}
	if m.tab == TabMaterials {
		return m.matNodes(), m.matCursor
	}
	return nil, 0
}

// nodeSelectionIDs resolves the assignment/material IDs a row stands for.
func (m *App) nodeSelectionIDs(n node) []string {
	switch n.kind {
	case nodeTask:
		return []string{n.task.ID}
	case nodeTaskMaterial:
		return []string{n.task.ID}
	case nodeFile:
		return []string{n.mat.ID}
	case nodeCourse, nodeBucket:
		if m.tab == TabTasks {
			return m.taskGroupIDs(n)
		}
		return idsOf(m.materialNodeFiles(n))
	case nodeMaterialSection, nodeMaterialFolder:
		return idsOf(m.materialNodeFiles(n))
	}
	return nil
}

func (m *App) startRange() {
	nodes, cursor := m.selectionNodes()
	if cursor < 0 || cursor >= len(nodes) {
		return
	}
	m.visual = true
	m.rangeAnchor = cursor
	m.selectionSet() // ensure non-nil
	for id := range m.selectionSet() {
		delete(m.selectionSet(), id)
	}
	for _, id := range m.nodeSelectionIDs(nodes[cursor]) {
		m.selectionSet()[id] = true
	}
	m.toast = "range select · j/k extend · esc clear"
	m.toastTicks = 24
}

func (m *App) extendRange() {
	if !m.visual {
		return
	}
	nodes, cursor := m.selectionNodes()
	if cursor < 0 || cursor >= len(nodes) || m.rangeAnchor < 0 || m.rangeAnchor >= len(nodes) {
		return
	}
	set := m.selectionSet()
	for id := range set {
		delete(set, id)
	}
	start, end := m.rangeAnchor, cursor
	if start > end {
		start, end = end, start
	}
	for i := start; i <= end; i++ {
		for _, id := range m.nodeSelectionIDs(nodes[i]) {
			set[id] = true
		}
	}
}

func (m *App) invertVisible() {
	if m.tab == TabTasks {
		for _, a := range m.visible() {
			if m.selected[a.ID] {
				delete(m.selected, a.ID)
			} else {
				m.selected[a.ID] = true
			}
		}
		return
	}
	if m.tab == TabMaterials {
		for _, material := range m.visibleMaterials() {
			if m.selectedM[material.ID] {
				delete(m.selectedM, material.ID)
			} else {
				m.selectedM[material.ID] = true
			}
		}
	}
}

// undoTriage restores the triage values from the last applyTriage call and
// returns the persistence command.
func (m *App) undoTriage() tea.Cmd {
	if m.tab != TabTasks || len(m.undoDecisions) == 0 {
		m.toast = "nothing to undo"
		m.toastTicks = 20
		return nil
	}
	ids := make([]string, 0, len(m.undoDecisions))
	for id, previous := range m.undoDecisions {
		ids = append(ids, id)
		m.queue.Set(id, previous)
		m.pendingDecisions[id] = previous
	}
	m.service.ApplyDecisions(m.undoDecisions)
	m.assignments = m.service.Snapshot().Assignments
	m.state = m.service.State()
	m.statusErr = ""
	m.undoDecisions = map[string]model.Triage{}
	m.toast = fmt.Sprintf("undid %d %s", len(ids), plural(len(ids), "decision", "decisions"))
	m.toastTicks = 24
	return m.queueStatePersist()
}

// dropAllCommit requires a second press so a stray key cannot empty the queue.
func (m *App) dropAllCommit() {
	count := m.queue.Len()
	if count == 0 {
		m.toast = "queue empty"
		m.toastTicks = 20
		return
	}
	if !m.dropAllArmed {
		m.dropAllArmed = true
		m.toast = fmt.Sprintf("press X again to drop %d queued %s", count, plural(count, "item", "items"))
		m.toastTicks = 40
		return
	}
	m.queue.Clear()
	m.dropAllArmed = false
	m.clampCursor()
	m.toast = fmt.Sprintf("dropped %d queued %s", count, plural(count, "item", "items"))
	m.toastTicks = 24
}
