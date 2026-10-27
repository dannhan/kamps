package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ruyam/kamps/internal/core"
	"github.com/ruyam/kamps/internal/model"
)

// visible returns the Tasks-tab assignments after the active filter and
// search, sorted.
func (m *App) visible() []model.Assignment {
	viewFilter, search := m.taskQueryState()
	filter := viewFilter
	if filter == "attention" || filter == "upcoming" || filter == "closed" {
		filter = "all"
	}
	items := m.service.Tasks(core.Query{Filter: filter, Search: search, CourseID: m.courseID})
	out := make([]model.Assignment, 0, len(items))
	for _, a := range items {
		if m.courseScoped && a.CourseID != m.courseID {
			continue
		}
		undecided := a.Triage == model.Undecided
		if viewFilter == "attention" && !undecided {
			continue
		}
		if viewFilter == "upcoming" && !m.currentTask(a) {
			continue
		}
		if viewFilter == "closed" && !(a.Triage == model.Missed || a.Triage == model.Ignored) {
			continue
		}
		out = append(out, a)
	}
	return out
}

// cursorAssignmentID is the item the keyboard is pointing at, or "". On the
// tree, the cursor may rest on a course row — that is no single assignment.
func (m *App) cursorAssignmentID() string {
	if m.tab != TabTasks {
		return ""
	}
	nodes := m.taskNodes()
	if m.taskCursor >= 0 && m.taskCursor < len(nodes) && nodes[m.taskCursor].kind == nodeTask {
		return nodes[m.taskCursor].task.ID
	}
	return ""
}

// targetIDs is what a triage key acts on: the selection, else the cursor item.
// On the tree, a course row targets all its visible children.
func (m *App) targetIDs() []string {
	var ids []string
	for _, a := range m.assignments {
		if m.selected[a.ID] {
			ids = append(ids, a.ID)
		}
	}
	if len(ids) > 0 {
		return ids
	}
	if m.tab == TabTasks {
		if id := m.cursorAssignmentID(); id != "" {
			return []string{id}
		}

		return nil
	}
	if id := m.cursorAssignmentID(); id != "" {
		return []string{id}
	}
	return ids
}

func sameIDs(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	set := make(map[string]bool, len(left))
	for _, id := range left {
		set[id] = true
	}
	for _, id := range right {
		if !set[id] {
			return false
		}
	}
	return true
}

// hiddenTaskCount counts target IDs that exist but are not rendered as task
// rows, either because a section is folded or the active filter hides them.
func (m *App) hiddenTaskCount(ids []string) int {
	if len(ids) == 0 {
		return 0
	}
	rendered := make(map[string]bool)
	for _, n := range m.taskNodes() {
		if n.kind == nodeTask {
			rendered[n.task.ID] = true
		}
	}
	existing := make(map[string]bool, len(m.assignments))
	for _, assignment := range m.assignments {
		existing[assignment.ID] = true
	}
	hidden := 0
	for _, id := range ids {
		if existing[id] && !rendered[id] {
			hidden++
		}
	}
	return hidden
}

func (m *App) hiddenSelectedCount() int {
	if m.tab == TabMaterials {
		visible := map[string]bool{}
		for _, n := range m.matNodes() {
			if n.kind == nodeFile {
				visible[n.mat.ID] = true
			}
		}
		hidden := 0
		for id := range m.selectedM {
			if !visible[id] {
				hidden++
			}
		}
		return hidden
	}
	if m.tab != TabTasks {
		return 0
	}
	ids := make([]string, 0, len(m.selected))
	for id := range m.selected {
		ids = append(ids, id)
	}
	return m.hiddenTaskCount(ids)
}

func (m *App) byID(id string) *model.Assignment {
	for i := range m.assignments {
		if m.assignments[i].ID == id {
			return &m.assignments[i]
		}
	}
	return nil
}

// applyTriage sets the decision on the target items, updates the in-memory
// model, and returns the single persistence command.
func (m *App) applyTriage(t model.Triage) tea.Cmd {
	if m.tab != TabTasks {
		return nil
	}
	ids := m.targetIDs()
	if len(ids) == 0 {
		return nil
	}
	hidden := m.hiddenTaskCount(ids)
	if hidden > 0 && !(m.triageArmed && m.armedTriage == t && sameIDs(m.armedTriageIDs, ids)) {
		m.triageArmed = true
		m.armedTriage = t
		m.armedTriageIDs = append([]string(nil), ids...)
		m.armedTriageHidden = hidden
		return nil
	}
	beforeNodes := m.taskNodes()
	beforeCursor := m.taskCursor
	m.cancelTriageArm()
	m.undoDecisions = map[string]model.Triage{}
	for _, id := range ids {
		if a := m.byID(id); a != nil {
			m.undoDecisions[id] = a.Triage
		}
	}
	decisions := make(map[string]model.Triage, len(ids))
	for _, id := range ids {
		decisions[id] = t
		m.queue.Set(id, t)
		m.pendingDecisions[id] = t
	}
	m.service.ApplyDecisions(decisions)
	m.assignments = m.service.Snapshot().Assignments
	m.state = m.service.State()
	m.statusErr = ""
	m.toast = fmt.Sprintf("%d %s %s", len(ids), plural(len(ids), "item", "items"), t.String())
	m.toastTicks = 24
	m.selected = map[string]bool{}
	m.restoreAfterDecision(beforeNodes, beforeCursor)
	m.detailScroll = 0
	return m.queueStatePersist()
}

func (m *App) taskQueueLabel() string {
	return m.queueSummary()
}

// Other views can refresh the source too. Task focus restoration must use the
// saved Tasks query rather than the active Materials filter/search.
func (m *App) taskQueryState() (string, string) {
	if m.tab == TabTasks {
		return m.filter, m.search
	}
	return m.tabFilters[TabTasks], m.tabSearch[TabTasks]
}
