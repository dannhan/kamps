package tui

import (
	"testing"

	"github.com/ruyam/kamps/internal/model"
)

// TestSnapshot renders every screen so the layout can be checked by eye with
// `go test -run TestSnapshot -v ./internal/tui`. It fails only on a panic.
func TestSnapshot(t *testing.T) {
	m := newTestApp(t, 96, 30)

	m.tab, m.layout = TabTasks, LayList
	m.taskCursor = 2
	t.Logf("\n== Tasks · list ==\n%s", m.View())

	m.tab, m.layout = TabTasks, LayBoard
	m.boardCol = 0
	t.Logf("\n== Tasks · board ==\n%s", m.View())

	// A wizard mid-flow: a few selected, an action chosen, review pending.
	m.tab, m.layout = TabTasks, LayWizard
	items := m.visible()
	for i := 0; i < 3 && i < len(items); i++ {
		m.selected[items[i].ID] = true
	}
	m.wizAction = model.Tracked
	m.wizStep = 1
	t.Logf("\n== Tasks · wizard (action) ==\n%s", m.View())

	m.wizStep = 2
	m.pending = map[string]model.Triage{}
	for i := 0; i < 3 && i < len(items); i++ {
		m.pending[items[i].ID] = model.Tracked
	}
	t.Logf("\n== Tasks · wizard (review) ==\n%s", m.View())

	m.tab, m.layout = TabMaterials, LayList
	m.matCursor = 1
	m.selectedM[m.materials[2].ID] = true
	t.Logf("\n== Materials ==\n%s", m.View())

	m.tab = TabToday
	t.Logf("\n== Today ==\n%s", m.View())

	m.tab = TabSync
	t.Logf("\n== Sync ==\n%s", m.View())
}
