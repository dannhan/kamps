package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/ruyam/kamps/internal/model"
)

// visible returns the Tasks-tab assignments after the active filter and
// search, sorted.
func (m *App) visible() []model.Assignment {
	now := time.Now()
	out := make([]model.Assignment, 0, len(m.assignments))
	for _, a := range m.assignments {
		if !m.matchesFilter(a, now) || !m.matchesTaskSearch(a) {
			continue
		}
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return lessAssignment(out[i], out[j]) })
	return out
}

func (m *App) matchesFilter(a model.Assignment, now time.Time) bool {
	switch m.filter {
	case "undecided":
		return a.Triage == model.Undecided
	case "overdue":
		return a.Overdue(now)
	case "tracked":
		return a.Triage == model.Tracked
	case "missed":
		return a.Triage == model.Missed
	case "ignored":
		return a.Triage == model.Ignored
	}
	return true
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
		nodes := m.taskNodes()
		if m.taskCursor >= 0 && m.taskCursor < len(nodes) && m.isTaskGroup(nodes[m.taskCursor]) {
			return m.taskGroupIDs(nodes[m.taskCursor])
		}
		return nil
	}
	if id := m.cursorAssignmentID(); id != "" {
		return []string{id}
	}
	return ids
}

func (m *App) byID(id string) *model.Assignment {
	for i := range m.assignments {
		if m.assignments[i].ID == id {
			return &m.assignments[i]
		}
	}
	return nil
}

// applyTriage sets the decision on the target items and queues them for commit.
func (m *App) applyTriage(t model.Triage) {
	if m.tab != TabTasks {
		return
	}
	ids := m.targetIDs()
	if len(ids) == 0 {
		return
	}
	for _, id := range ids {
		if a := m.byID(id); a != nil {
			a.Triage = t
		}
		m.state.SetDecision(id, t)
		if t == model.Undecided {
			delete(m.pending, id)
		} else {
			m.pending[id] = t
		}
	}
	m.selected = map[string]bool{}
	m.toast = fmt.Sprintf("%d %s %s", len(ids), plural(len(ids), "item", "items"), t.String())
	m.toastTicks = 24
	m.saveState()
}

func (m *App) taskFooter(width int) string {
	queue := m.taskQueueLabel()
	if f := m.footer(); f != "" {
		queue = f
	}
	queueWidth := lipgloss.Width(queue)
	if queueWidth >= width {
		return fitLine(queue, width)
	}
	hintWidth := maxInt(0, width-queueWidth-2)
	hints := taskFooterHints(hintWidth)
	gap := maxInt(1, width-lipgloss.Width(hints)-queueWidth)
	return fitLine(hints+strings.Repeat(" ", gap)+queue, width)
}

func (m *App) taskQueueLabel() string {
	tracked, missed, ignored := m.pendingCounts()
	if tracked+missed+ignored == 0 {
		return stDim.Render("queue empty")
	}
	var parts []string
	if tracked > 0 {
		parts = append(parts, fmt.Sprintf("%d track", tracked))
	}
	if missed > 0 {
		parts = append(parts, fmt.Sprintf("%d missed", missed))
	}
	if ignored > 0 {
		parts = append(parts, fmt.Sprintf("%d ignore", ignored))
	}
	return stKey.Render("queue: " + strings.Join(parts, " · "))
}
