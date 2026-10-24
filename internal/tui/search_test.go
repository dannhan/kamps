package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSlashOpensSearchAndNarrowsTasks(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks
	app.width, app.height = 100, 24

	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !app.searching {
		t.Fatal("slash should open search")
	}
	for _, r := range "quiz" {
		app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if app.search != "quiz" {
		t.Fatalf("search = %q, want quiz", app.search)
	}

	visible := app.visible()
	if len(visible) == 0 || len(visible) == len(app.assignments) {
		t.Fatalf("search matched %d of %d assignments", len(visible), len(app.assignments))
	}
	for _, assignment := range visible {
		if !app.matchesTaskSearch(assignment) {
			t.Fatalf("search kept a non-matching task: %q", assignment.Name)
		}
	}

	view := stripANSI(app.View())
	if !strings.Contains(view, "search · quiz") || !strings.Contains(view, "matches") {
		t.Fatalf("search title missing from view: %q", view)
	}
	if !strings.Contains(view, "enter done") {
		t.Fatalf("search footer missing: %q", view)
	}
}

func TestSearchKeepsCompoundInputAndBackspace(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks

	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	app.handleKey(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if app.search != "q u" {
		t.Fatalf("space should be typed into the query, got %q", app.search)
	}

	app.handleKey(tea.KeyMsg{Type: tea.KeyBackspace})
	if app.search != "q " {
		t.Fatalf("backspace = %q, want %q", app.search, "q ")
	}
	app.handleKey(tea.KeyMsg{Type: tea.KeyCtrlU})
	if app.search != "" {
		t.Fatalf("ctrl+u should clear the query, got %q", app.search)
	}
}

func TestSearchCtrlWDeletesWords(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks
	app.searching = true
	app.search = "quiz basis data"

	for i, want := range []string{"quiz basis ", "quiz ", ""} {
		app.handleKey(tea.KeyMsg{Type: tea.KeyCtrlW})
		if app.search != want {
			t.Fatalf("ctrl+w #%d = %q, want %q", i+1, app.search, want)
		}
	}

	app.search = "quiz basis"
	app.handleKey(tea.KeyMsg{Type: tea.KeyBackspace, Alt: true})
	if app.search != "quiz " {
		t.Fatalf("alt+backspace = %q, want %q", app.search, "quiz ")
	}
}

func TestSearchEnterKeepsQueryAndEscClearsItFirst(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks
	app.search = "quiz"
	app.searching = true

	app.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if app.searching || app.search != "quiz" {
		t.Fatalf("enter should keep the query and leave edit mode: %q searching=%v", app.search, app.searching)
	}
	if len(app.visible()) == 0 {
		t.Fatal("query should still narrow the tree after enter")
	}

	app.filter = "undecided"
	app.tabFilters[TabTasks] = "undecided"
	app.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if app.search != "" || app.filter != "undecided" {
		t.Fatalf("esc should clear search first: search=%q filter=%q", app.search, app.filter)
	}
	app.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if app.filter != "all" {
		t.Fatalf("second esc should clear the filter, got %q", app.filter)
	}
}

func TestSearchNarrowsMaterialsAndComposesWithFilter(t *testing.T) {
	app := demoApp(t)
	app.tab = TabMaterials
	app.search = "modul"

	files := 0
	for _, n := range app.matNodes() {
		if n.kind != nodeFile {
			continue
		}
		files++
		if !app.matchesMaterialSearch(n.mat) {
			t.Fatalf("search kept a non-matching material: %q", n.mat.Name)
		}
	}
	if files == 0 {
		t.Fatal("materials search found no files")
	}

	app.filter = "new"
	for _, n := range app.matNodes() {
		if n.kind == nodeFile && n.mat.Downloaded {
			t.Fatalf("new filter let a downloaded file through: %q", n.mat.Name)
		}
	}
}

func TestSwitchTabClearsSearch(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks
	app.search = "quiz"
	app.searching = true

	app.switchTab(TabMaterials)
	if app.search != "" || app.searching {
		t.Fatalf("tab switch should drop the search: %q searching=%v", app.search, app.searching)
	}
}

func TestFilterCyclesOnFAndNotSlash(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks

	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	if app.filter != "undecided" {
		t.Fatalf("f should advance the filter, got %q", app.filter)
	}

	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if app.filter != "undecided" {
		t.Fatalf("slash should be reserved for search, got filter %q", app.filter)
	}
}

func TestEscapeClearsMaterialFilter(t *testing.T) {
	app := demoApp(t)
	app.tab = TabMaterials
	app.filter = "new"
	app.tabFilters[TabMaterials] = "new"
	app.matCursor = 2

	app.handleKey(tea.KeyMsg{Type: tea.KeyEsc})

	if app.filter != "all" || app.tabFilters[TabMaterials] != "all" {
		t.Fatalf("esc should clear the material filter, got filter=%q stored=%q", app.filter, app.tabFilters[TabMaterials])
	}
	if app.matCursor != 0 {
		t.Fatalf("clearing a filter should reset the cursor, got %d", app.matCursor)
	}
}
