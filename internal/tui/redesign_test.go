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

func TestFooterStatusShowsSelectionQueueFilterAndError(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks
	app.width, app.height = 100, 30
	app.selected["K901000"] = true
	app.queue.Set("K901001", model.Tracked)
	app.filter = "overdue"

	lines := app.footerLines(100, sizeStandard)
	if len(lines) != 2 {
		t.Fatalf("standard footer lines = %d, want 2", len(lines))
	}
	status := stripANSI(lines[1])
	for _, want := range []string{"TASKS", "1 selected", "queue: 1 track", "filter: overdue"} {
		if !strings.Contains(status, want) {
			t.Fatalf("footer status %q is missing %q", status, want)
		}
	}

	app.statusErr = "disk full"
	lines = app.footerLines(100, sizeStandard)
	if !strings.Contains(stripANSI(lines[1]), "err: disk full") || !strings.Contains(stripANSI(lines[1]), "esc dismiss") {
		t.Fatalf("durable error missing from footer: %q", stripANSI(lines[1]))
	}

	app.width, app.height = 60, 20
	compact := app.footerLines(60, sizeCompact)
	if len(compact) != 1 || !strings.Contains(stripANSI(compact[0]), "err: disk full") {
		t.Fatalf("compact footer hid the error: %#v", compact)
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

func TestCursorSelectionAndPartialAreDistinct(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks
	nodes := app.taskNodes()

	bucket := -1
	for i, n := range nodes {
		if n.kind == nodeBucket && len(app.taskGroupIDs(n)) > 1 {
			bucket = i
			break
		}
	}
	if bucket < 0 {
		t.Fatal("demo data has no multi-task bucket")
	}
	ids := app.taskGroupIDs(nodes[bucket])
	taskIndex := -1
	for i, n := range nodes {
		if n.kind == nodeTask && n.task.ID == ids[0] {
			taskIndex = i
			break
		}
	}
	if taskIndex < 0 {
		t.Fatal("could not find the first task in the bucket")
	}

	app.taskCursor = taskIndex
	cursor := stripANSI(app.taskTreeRow(nodes, taskIndex, 98))
	if !strings.Contains(cursor, "▌") || strings.Contains(cursor, "▪") {
		t.Fatalf("cursor row = %q", cursor)
	}

	app.taskCursor = -1
	app.selected[ids[0]] = true
	selected := stripANSI(app.taskTreeRow(nodes, taskIndex, 98))
	if !strings.Contains(selected, "▪") || strings.Contains(selected, "▌") {
		t.Fatalf("selected row = %q", selected)
	}

	app.taskCursor = taskIndex
	both := stripANSI(app.taskTreeRow(nodes, taskIndex, 98))
	if !strings.Contains(both, "▌") || !strings.Contains(both, "▪") {
		t.Fatalf("cursor+selected row = %q", both)
	}

	partial := stripANSI(app.taskTreeRow(nodes, bucket, 98))
	if !strings.Contains(partial, "▫") {
		t.Fatalf("partially selected bucket = %q", partial)
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
		if wantSplit := sizeClassFor(size[0], size[1]) == sizeWide; (app.detailMode() == detailSplit) != wantSplit {
			t.Fatalf("%dx%d detail mode = %v, want split=%v", size[0], size[1], app.detailMode(), wantSplit)
		}
		for i, line := range strings.Split(stripANSI(app.View()), "\n") {
			if got := lipgloss.Width(line); got != size[0] {
				t.Fatalf("%dx%d line %d width %d", size[0], size[1], i+1, got)
			}
		}
	}
}
