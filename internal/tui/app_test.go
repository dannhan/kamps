package tui

import (
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

func TestShellUsesRulesInsteadOfABox(t *testing.T) {
	app := demoApp(t)
	app.width, app.height = 100, 30
	lines := strings.Split(stripANSI(app.View()), "\n")
	if strings.ContainsAny(strings.Join(lines, ""), "╭╮") {
		t.Fatal("shell still draws rounded pane corners")
	}
	for i, line := range lines {
		if strings.HasSuffix(line, "│") {
			t.Fatalf("line %d still has a vertical pane border: %q", i, line)
		}
	}
	want := strings.Repeat("─", 100)
	rules := 0
	for _, line := range lines {
		if line == want {
			rules++
		}
	}
	if rules != 2 {
		t.Fatalf("want one rule above and one below the body, got %d", rules)
	}
	if lines[2] != want || lines[len(lines)-2] != want {
		t.Fatalf("rules are not fenced around the body: %q / %q", lines[2], lines[len(lines)-2])
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

func TestScrollHintReportsHiddenRows(t *testing.T) {
	cases := []struct {
		total, cursor, height int
		want                  string
	}{
		{20, 10, 5, "▲ 8 ▼ 7"},
		{10, 0, 3, "▼ 7"},
		{10, 9, 3, "▲ 7"},
		{5, 4, 5, ""},
		{0, 0, 5, ""},
	}
	for _, tc := range cases {
		if got := stripANSI(scrollHint(tc.total, tc.cursor, tc.height)); got != tc.want {
			t.Errorf("scrollHint(%d, %d, %d) = %q, want %q", tc.total, tc.cursor, tc.height, got, tc.want)
		}
	}
}

func TestPanesAdvertiseHiddenRows(t *testing.T) {
	app := demoApp(t)

	app.tab = TabTasks
	if _, right, _, _ := app.tabContent(96, 1); !strings.Contains(stripANSI(right), "▼") {
		t.Fatalf("a clipped task tree should show a down hint, got %q", stripANSI(right))
	}
	if _, right, _, _ := app.tabContent(96, 500); strings.ContainsAny(stripANSI(right), "▲▼") {
		t.Fatalf("a tree that fits should not show a scroll hint, got %q", stripANSI(right))
	}

	app.tab = TabMaterials
	if _, right, _, _ := app.tabContent(96, 1); !strings.Contains(stripANSI(right), "▼") {
		t.Fatalf("a clipped material tree should show a down hint, got %q", stripANSI(right))
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
