package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ruyam/kamps/internal/model"
)

func TestLayoutBreakpoints(t *testing.T) {
	sizes := [][2]int{{40, 12}, {56, 16}, {60, 20}, {79, 30}, {80, 24}, {100, 23}, {100, 30}, {120, 40}, {160, 50}}
	for _, size := range sizes {
		for _, tab := range []Tab{TabTasks, TabMaterials, TabSync, TabCommit} {
			app := demoApp(t)
			app.width, app.height = size[0], size[1]
			app.tab = tab
			view := stripANSI(app.View())
			if strings.Contains(view, "terminal too small") {
				t.Fatalf("%dx%d tab %s fell back to the tiny message", size[0], size[1], tab)
			}
			lines := strings.Split(view, "\n")
			if len(lines) != size[1] {
				t.Fatalf("%dx%d tab %s has %d rows, want %d", size[0], size[1], tab, len(lines), size[1])
			}
			for i, line := range lines {
				if got := lipgloss.Width(line); got != size[0] {
					t.Fatalf("%dx%d tab %s line %d width %d: %q", size[0], size[1], tab, i+1, got, line)
				}
			}
			if !strings.Contains(view, "? all keys") {
				t.Fatalf("%dx%d tab %s lost the safety hint: %q", size[0], size[1], tab, view)
			}
		}
	}
}

func TestTaskStatusDoesNotReplaceDecisions(t *testing.T) {
	app := demoApp(t)
	app.selected["K901000"] = true
	app.queue.Set("K901001", model.Tracked)
	title, _, _, _ := app.workspaceContent(100, 24)
	for _, want := range []string{"1 selected", "queue: 1 track"} {
		if !strings.Contains(stripANSI(title), want) {
			t.Fatalf("missing %s: %s", want, title)
		}
	}
	for _, width := range []int{60, 80, 100} {
		app.width = width
		app.statusErr = "disk full"
		title, _, _, _ = app.workspaceContent(width, 24)
		if !strings.Contains(title, "disk full") || !strings.Contains(title, "esc dismiss") {
			t.Fatal("error missing")
		}
		footer := stripANSI(strings.Join(app.footerLines(width, sizeStandard), ""))
		for _, want := range []string{"t Track", "i Ignore", "m Missed"} {
			if !strings.Contains(footer, want) {
				t.Fatalf("footer lost %s", want)
			}
		}
	}
}

func TestHelpGroupsCoverEveryKey(t *testing.T) {
	app := demoApp(t)
	app.width, app.height = 200, 80
	content := stripANSI(strings.Join(app.helpContent(200), "\n"))
	for _, want := range []string{"NAVIGATION", "ACTIONS", "VIEW", "LEGEND", "! overdue", "▪ selected"} {
		if !strings.Contains(content, want) {
			t.Fatalf("help is missing %q:\n%s", want, content)
		}
	}
	if strings.Contains(content, "● done") {
		t.Fatalf("help still advertises the impossible done state:\n%s", content)
	}

	bindings := []key.Binding{app.keys.Confirm}
	for _, group := range app.keys.FullHelp() {
		bindings = append(bindings, group...)
	}
	for _, binding := range bindings {
		help := binding.Help()
		if !strings.Contains(content, help.Key) {
			t.Errorf("help is missing key %q (%q)", help.Key, help.Desc)
		}
		if !strings.Contains(content, help.Desc) {
			t.Errorf("help is missing description %q", help.Desc)
		}
	}

	app.width, app.height = 80, 24
	if app.helpMaxScroll() == 0 {
		t.Fatal("help should scroll at 80x24")
	}
	app.helpScroll = 0
	first, clipped := app.helpLines(80, 18)
	if !clipped || len(first) != 18 {
		t.Fatalf("help window = %d lines, clipped=%v", len(first), clipped)
	}
	app.helpScroll = 6
	shifted, _ := app.helpLines(80, 18)
	if stripANSI(shifted[0]) == stripANSI(first[0]) {
		t.Fatal("help scroll did not move the window")
	}
}

func TestResizePreservesCursorAndRecomputesPane(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks
	app.width, app.height = 140, 40

	nodes := app.taskNodes()
	index := -1
	for i, n := range nodes {
		if n.kind == nodeTask {
			index = i
			break
		}
	}
	if index < 0 {
		t.Fatal("demo data has no task row")
	}
	app.taskCursor = index
	key := nodes[index].key

	for _, size := range [][2]int{{140, 40}, {100, 30}, {80, 24}, {60, 20}, {40, 12}, {120, 34}} {
		app.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		nodes := app.taskNodes()
		if app.taskCursor < 0 || app.taskCursor >= len(nodes) || nodes[app.taskCursor].key != key {
			t.Fatalf("%dx%d lost the cursor: %d/%d", size[0], size[1], app.taskCursor, len(nodes))
		}
		if wantSplit := false; (app.detailMode() == detailSplit) != wantSplit {
			t.Fatalf("%dx%d detail mode = %v, want split=%v", size[0], size[1], app.detailMode(), wantSplit)
		}
		for i, line := range strings.Split(stripANSI(app.View()), "\n") {
			if got := lipgloss.Width(line); got != size[0] {
				t.Fatalf("%dx%d line %d width %d", size[0], size[1], i+1, got)
			}
		}
	}
}
