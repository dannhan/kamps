package tui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/ruyam/kamps/internal/model"
)

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

func TestSharedShellAllTabsAtRequestedSizes(t *testing.T) {
	for _, size := range [][2]int{{100, 30}, {80, 24}} {
		var reference []int
		for active := TabTasks; active < tabCount; active++ {
			app := demoApp(t)
			app.width, app.height = size[0], size[1]
			app.tab = active
			if active == TabCommit {
				setPendingForTest(app, map[string]model.Triage{"K901000": model.Tracked})
			}
			view := stripANSI(app.View())
			lines := strings.Split(view, "\n")
			if len(lines) != size[1] {
				t.Fatalf("%dx%d tab %s has %d rows", size[0], size[1], active, len(lines))
			}
			for i, line := range lines {
				if got := lipgloss.Width(line); got != size[0] {
					t.Fatalf("%dx%d tab %s line %d width %d", size[0], size[1], active, i+1, got)
				}
			}
			if strings.TrimSpace(lines[1]) == "" {
				t.Fatalf("%dx%d tab %s title row is empty", size[0], size[1], active)
			}
			positions := make([]int, tabCount)
			for tab, label := range []string{"Tasks", "Materials", "Sync", "Commit"} {
				positions[tab] = strings.Index(lines[0], label)
			}
			if active == TabTasks {
				reference = positions
			} else if !reflect.DeepEqual(reference, positions) {
				t.Fatalf("tab cell starts differ at %dx%d with active %s: got %v want %v", size[0], size[1], active, positions, reference)
			}
		}
	}
}

func TestSharedShellShowsSmallTerminalMessage(t *testing.T) {
	app := demoApp(t)
	app.width, app.height = 79, 23
	view := stripANSI(app.View())
	if !strings.Contains(view, "terminal too small") {
		t.Fatalf("small-terminal message missing: %q", view)
	}
	if strings.Count(view, "? more") != 1 {
		t.Fatalf("small-terminal footer repeats the ? more hint: %q", view)
	}
}
