package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ruyam/kamps/internal/model"
)

func TestFooterKeepsPriorityAndSafetyHint(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks
	full := renderHintItems(app.footerHintItems(), 200)
	want := "↑↓ move · enter detail · t/m/i decide · / search · f filter · ? all keys"
	if got := stripANSI(full); got != want {
		t.Fatalf("full footer = %q, want %q", got, want)
	}
	short := renderHintItems(app.footerHintItems(), 12)
	if got := stripANSI(short); got != "? all keys" {
		t.Fatalf("short footer = %q, want ? all keys", got)
	}
	medium := stripANSI(renderHintItems(app.footerHintItems(), 40))
	if !strings.HasSuffix(medium, "? all keys") || strings.Contains(medium, "f filter") {
		t.Fatalf("short footer did not drop hints from the right: %q", medium)
	}
}

func TestFooterKeepsQueueAtRightEdge(t *testing.T) {
	app := demoApp(t)
	app.width, app.height = 100, 24
	lines := app.footerLines(100, 24)
	if len(lines) != 2 {
		t.Fatalf("footer lines = %d, want 2", len(lines))
	}
	hintLine := strings.TrimSpace(stripANSI(lines[0]))
	if !strings.HasPrefix(hintLine, "↑↓ move") || strings.Contains(hintLine, "queue") {
		t.Fatalf("footer hint alignment = %q", hintLine)
	}
	if strings.Contains(hintLine, "[") || strings.Contains(hintLine, "v view") || strings.Contains(hintLine, "mouse") {
		t.Fatalf("footer contains secondary controls or brackets: %q", hintLine)
	}
	app.queue.Set("tracked", model.Tracked)
	status := stripANSI(app.footerLines(100, 24)[1])
	if !strings.HasSuffix(status, "queue: 1 track") {
		t.Fatalf("populated queue is not right aligned: %q", status)
	}
}

func TestFooterKeepsQueueWhenHintsDoNotFit(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks
	app.queue.Set("tracked", model.Tracked)
	lines := app.footerLines(24, 24)
	if len(lines) != 2 {
		t.Fatalf("footer lines = %d, want 2", len(lines))
	}
	if got := stripANSI(lines[0]); !strings.Contains(got, "? all keys") {
		t.Fatalf("narrow footer dropped the safety hint: %q", got)
	}
	if status := stripANSI(lines[1]); !strings.HasSuffix(status, "queue: 1 track") {
		t.Fatalf("footer dropped queue: %q", status)
	}
}

func TestFooterShowsToasts(t *testing.T) {
	app := demoApp(t)
	app.width, app.height = 100, 24
	app.toast = "copied 2 lines"
	app.toastTicks = 20
	for _, tab := range []Tab{TabTasks, TabMaterials} {
		app.tab = tab
		view := stripANSI(app.View())
		if !strings.Contains(view, "copied 2 lines") {
			t.Fatalf("%s footer hid the toast: %q", tab, view)
		}
	}
}

func TestTaskQueueLabels(t *testing.T) {
	app := demoApp(t)
	if got := stripANSI(app.taskQueueLabel()); got != "" {
		t.Fatalf("empty queue label = %q", got)
	}
	app.queue.Set("a", model.Tracked)
	app.queue.Set("b", model.Tracked)
	app.queue.Set("c", model.Missed)
	if got := stripANSI(app.taskQueueLabel()); got != "queue: 2 track · 1 missed" {
		t.Fatalf("populated queue label = %q", got)
	}
}

func TestCursorAndMarksAreVisuallyDistinct(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks

	nodes := app.taskNodes()
	marked := findTaskNode(nodes, nodeTask, "K901000")
	unmarked := findTaskNode(nodes, nodeTask, "K901001")
	if marked < 0 || unmarked < 0 {
		t.Fatal("demo assignments missing from the default tree")
	}
	app.selected["K901000"] = true
	app.taskCursor = unmarked

	markedRow := stripANSI(app.taskTreeRow(nodes, marked, 98))
	unmarkedRow := stripANSI(app.taskTreeRow(nodes, unmarked, 98))
	if !strings.Contains(markedRow, markFull) || strings.Contains(unmarkedRow, markFull) || strings.Contains(unmarkedRow, "▫") {
		t.Fatalf("mark glyphs = %q / %q", markedRow, unmarkedRow)
	}
	if !strings.Contains(unmarkedRow, "▌") {
		t.Fatalf("cursor row lacks its accent gutter: %q", unmarkedRow)
	}
	if strings.Contains(markedRow, "▌") {
		t.Fatalf("a marked-only row must not look like the cursor: %q", markedRow)
	}
	if themeTokens.Surface == themeTokens.Marked {
		t.Fatal("cursor and marked rows share the same band colour")
	}
}

func TestParentRowsShowCaretAndMark(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks

	nodes := app.taskNodes()
	app.taskCursor = findTaskNode(nodes, nodeTask, "K901000")

	// Course rows are the top-level parents, without a placeholder term row.
	course := findTaskNode(nodes, nodeCourse, "")
	if row := stripANSI(app.taskTreeRow(nodes, course, 98)); !strings.Contains(row, "▾") || !strings.Contains(row, "Proyek Perangkat Lunak A") || strings.Contains(row, "▫") {
		t.Fatalf("course should be top-level with a down caret and no placeholder mark: %q", row)
	}

	// The demo tree's "closed" bucket starts folded.
	closed := findTaskNode(nodes, nodeBucket, "closed")
	row := stripANSI(app.taskTreeRow(nodes, closed, 98))
	if !strings.Contains(row, "▸") || !strings.Contains(row, "closed") || strings.Contains(row, "▫") {
		t.Fatalf("folded bucket should show a right caret: %q", row)
	}

	// Marking every child turns the parent's mark on; a partial mark keeps it off.
	tracked := findTaskNode(nodes, nodeBucket, "tracked")
	ids := app.taskGroupIDs(nodes[tracked])
	for _, id := range ids {
		app.selected[id] = true
	}
	app.expanded[nodes[tracked].key] = true
	row = stripANSI(app.taskTreeRow(nodes, tracked, 98))
	if !strings.Contains(row, "▾ "+markFull+" tracked") {
		t.Fatalf("fully marked parent should show %q: %q", markFull, row)
	}
	delete(app.selected, ids[0])
	row = stripANSI(app.taskTreeRow(nodes, tracked, 98))
	if strings.Contains(row, markFull) || strings.Contains(row, "▫") {
		t.Fatalf("partially marked parent should show no filled mark: %q", row)
	}
}

func TestTaskTotalsShowEveryStateCount(t *testing.T) {
	app := demoApp(t)
	totals := tallyTriage(app.assignments)
	legend := stripANSI(app.taskTotals(200))
	for _, want := range []string{
		fmt.Sprintf("○ new %d", totals[0]),
		fmt.Sprintf("◐ tracked %d", totals[1]),
		fmt.Sprintf("◆ ignored %d", totals[3]),
		fmt.Sprintf("× missed %d", totals[4]),
	} {
		if !strings.Contains(legend, want) {
			t.Fatalf("legend %q missing %q", legend, want)
		}
	}
	if narrow := stripANSI(app.taskTotals(30)); strings.Contains(narrow, "new") || !strings.Contains(narrow, "○ ") {
		t.Fatalf("narrow legend should fall back to glyph and count: %q", narrow)
	}
}

func TestDueAgeUsesCalendarDays(t *testing.T) {
	now := time.Date(2026, 10, 18, 23, 30, 0, 0, time.UTC)
	if got := dueAge(model.Assignment{HasDue: true, Due: time.Date(2026, 10, 17, 23, 55, 0, 0, time.UTC)}, now); got != "−1d" {
		t.Fatalf("past due age = %q, want −1d", got)
	}
	if got := dueAge(model.Assignment{HasDue: true, Due: time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)}, now); got != "+2d" {
		t.Fatalf("upcoming age = %q, want +2d", got)
	}
}
