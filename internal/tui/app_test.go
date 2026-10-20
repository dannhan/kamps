package tui

import (
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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

func TestDetailPaneFollowsWidthAndEnter(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks

	app.width = 100
	if !app.detailVisible() {
		t.Fatal("detail pane should show at 100 columns")
	}
	app.width = 80
	if app.detailVisible() {
		t.Fatal("detail pane should auto-hide at 80 columns")
	}

	app.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	if !app.detailVisible() {
		t.Fatal("enter should pin the detail pane open at 80 columns")
	}
	if app.tab != TabTasks {
		t.Fatal("enter must not jump to the commit tab")
	}
	app.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if app.detailVisible() {
		t.Fatal("a second enter should close the detail pane")
	}

	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if app.tab != TabCommit {
		t.Fatal("c should still open the commit tab")
	}
}

func TestDetailPaneRendersTheSelectedTask(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks
	nodes := app.taskNodes()
	cursor := -1
	for i, n := range nodes {
		if n.kind == nodeTask {
			cursor = i
			break
		}
	}
	if cursor < 0 {
		t.Fatal("demo data has no task rows")
	}
	task := nodes[cursor].task
	lines := stripANSI(strings.Join(app.detailLines(nodes[cursor], 40), "\n"))
	for _, want := range []string{task.ID, task.Name, "due", "state", "course"} {
		if !strings.Contains(lines, want) {
			t.Fatalf("detail pane %q is missing %q", lines, want)
		}
	}
}

func TestDetailPaneClipsLongTextWithEllipsis(t *testing.T) {
	app := demoApp(t)
	app.taskCursor = 0
	long := strings.Repeat("a very long task description ", 40)
	nodes := []node{{kind: nodeTask, task: model.Assignment{ID: "K999999", Name: long, Triage: model.Undecided}}}
	rows := app.detailRows(nodes, 34, 8)
	if len(rows) != 8 {
		t.Fatalf("detail rows = %d, want 8", len(rows))
	}
	for i, row := range rows {
		if got := lipgloss.Width(row); got != 34 {
			t.Fatalf("detail row %d width %d, want 34", i, got)
		}
	}
	if !strings.Contains(stripANSI(rows[7]), "…") {
		t.Fatalf("clipped detail pane has no ellipsis: %q", stripANSI(rows[7]))
	}
}

func TestSplitPaneKeepsEveryLineAtWidth(t *testing.T) {
	for _, width := range []int{100, 80} {
		app := demoApp(t)
		app.width, app.height = width, 24
		app.tab = TabTasks
		app.detailPane = 1 // force the pane on, even at 80
		for i, line := range strings.Split(stripANSI(app.View()), "\n") {
			if got := lipgloss.Width(line); got != width {
				t.Fatalf("%d-column split line %d width %d: %q", width, i, got, line)
			}
		}
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
