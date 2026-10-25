package tui

import (
	"slices"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ruyam/kamps/internal/model"
)

func TestEscapeClearsFilterBeforeSelections(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks
	app.filter = "undecided"
	app.tabFilters[TabTasks] = "undecided"
	app.taskCursor = 3
	selectedID := app.assignments[0].ID
	app.selected[selectedID] = true

	if _, _, hints, _ := app.tabContent(96, 20); !slices.Contains(hints, "esc clear") {
		t.Fatalf("an active filter should advertise esc clear, got %v", hints)
	}

	app.handleKey(tea.KeyMsg{Type: tea.KeyEsc})

	if app.filter != "all" || app.tabFilters[TabTasks] != "all" {
		t.Fatalf("esc should clear the filter, got filter=%q stored=%q", app.filter, app.tabFilters[TabTasks])
	}
	if !app.selected[selectedID] {
		t.Fatal("clearing a filter should not drop the selection")
	}
	if app.taskCursor != 0 {
		t.Fatalf("clearing a filter should reset the cursor, got %d", app.taskCursor)
	}

	app.handleKey(tea.KeyMsg{Type: tea.KeyEsc})

	if len(app.selected) != 0 {
		t.Fatalf("a second esc should clear the selection, got %v", app.selected)
	}
}

func TestTabSwitchRestoresPerTabCursorAndFilter(t *testing.T) {
	app := demoApp(t)
	app.taskCursor = 7
	app.filter = "tracked"
	app.tabFilters[TabTasks] = "tracked"
	app.switchTab(TabMaterials)
	app.matCursor = 4
	app.switchTab(TabTasks)
	if app.taskCursor != 7 || app.filter != "tracked" {
		t.Fatalf("Tasks state not restored: cursor=%d filter=%q", app.taskCursor, app.filter)
	}
	app.switchTab(TabMaterials)
	if app.matCursor != 4 {
		t.Fatalf("Materials cursor not restored: %d", app.matCursor)
	}
}

func TestFoldingOpenGroupKeepsCursorOnIt(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks

	nodes := app.taskNodes()
	tracked := findTaskNode(nodes, nodeBucket, "tracked")
	if tracked < 0 {
		t.Fatal("tracked bucket missing from the demo tree")
	}
	app.taskCursor = tracked
	app.expanded[nodes[tracked].key] = true
	app.moveX(-1)
	if app.taskCursor != tracked {
		t.Fatalf("folding an open bucket moved the cursor: got %d, want %d", app.taskCursor, tracked)
	}
	if findTaskNode(app.taskNodes(), nodeBucket, "tracked") < 0 {
		t.Fatal("the bucket the cursor stayed on vanished from the tree")
	}
}

func TestFoldOnLeafMovesCursorToParent(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks

	nodes := app.taskNodes()
	task := findTaskNode(nodes, nodeTask, "K901000")
	if task < 0 {
		t.Fatal("assignment K901000 missing from the default tree")
	}
	expected := findTaskNode(nodes, nodeBucket, "needs decision")
	app.taskCursor = task
	app.moveX(-1)
	if app.taskCursor != expected {
		t.Fatalf("h on a leaf moved the cursor to %d, want parent bucket at %d", app.taskCursor, expected)
	}
}

func TestFoldOnAlreadyFoldedGroupMovesCursorToParent(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks

	app.expanded[courseKey(18515)] = false
	nodes := app.taskNodes()
	course := findTaskNode(nodes, nodeCourse, "")
	expected := findTaskNode(nodes, nodeCourse, "")
	if course < 0 || expected != course {
		t.Fatalf("unexpected folded tree: course=%d expected=%d", course, expected)
	}
	app.taskCursor = course
	app.moveX(-1)
	if app.taskCursor != expected {
		t.Fatalf("h on an already-folded course moved the cursor to %d, want term at %d", app.taskCursor, expected)
	}
}

func TestNaturalNavigationKeys(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks
	nodes := app.taskNodes()
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if app.taskCursor != len(nodes)-1 {
		t.Fatalf("G moved to %d, want %d", app.taskCursor, len(nodes)-1)
	}
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if app.taskCursor != 0 {
		t.Fatalf("g moved to %d, want 0", app.taskCursor)
	}
}

func TestBracketKeysSwitchTabs(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	if app.tab != TabMaterials {
		t.Fatalf("] switched to %v", app.tab)
	}
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	if app.tab != TabTasks {
		t.Fatalf("[ switched to %v", app.tab)
	}
}

func TestNewAndUndoKeys(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks
	id := "K901017"
	app.taskCursor = findTaskNode(app.taskNodes(), nodeTask, id)
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if app.byID(id).Triage != model.Tracked {
		t.Fatalf("t triage = %v", app.byID(id).Triage)
	}
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if app.byID(id).Triage != model.Undecided {
		t.Fatalf("u did not undo: %v", app.byID(id).Triage)
	}
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if app.byID(id).Triage != model.Undecided {
		t.Fatalf("n triage = %v", app.byID(id).Triage)
	}
}
