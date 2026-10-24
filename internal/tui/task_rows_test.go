package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/ruyam/kamps/internal/model"
)

func TestTaskFooterKeepsPriorityAndMoreHint(t *testing.T) {
	full := taskFooterHints(200)
	want := "↑↓ move · h/l fold · space select · t/m/i decide · c commit · f filter · enter detail · ? more"
	if got := stripANSI(full); got != want {
		t.Fatalf("full footer = %q, want %q", got, want)
	}
	short := taskFooterHints(12)
	if got := stripANSI(short); got != "? more" {
		t.Fatalf("short footer = %q, want ? more", got)
	}
	medium := stripANSI(taskFooterHints(40))
	if !strings.HasSuffix(medium, "? more") || strings.Contains(medium, "f filter") {
		t.Fatalf("short footer did not drop hints from the right: %q", medium)
	}
}

func TestTaskFooterKeepsQueueAtRightEdge(t *testing.T) {
	app := demoApp(t)
	app.width = 100
	line := stripANSI(app.taskFooter(99))
	if !strings.HasPrefix(line, "↑↓ move") || !strings.HasSuffix(line, "queue empty") {
		t.Fatalf("footer alignment = %q", line)
	}
	if strings.Contains(line, "[") || strings.Contains(line, "v view") || strings.Contains(line, "mouse") {
		t.Fatalf("footer contains secondary controls or brackets: %q", line)
	}
	app.pending["tracked"] = model.Tracked
	line = stripANSI(app.taskFooter(99))
	if !strings.HasSuffix(line, "queue: 1 track") {
		t.Fatalf("populated queue is not right aligned: %q", line)
	}
}

func TestFooterKeepsQueueWhenHintsDoNotFit(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks
	line := stripANSI(app.helpBar(24, []string{"↑↓ move", "h/l fold", "space select", "t/m/i decide", "c commit", "f filter"}))
	if !strings.HasSuffix(line, "queue empty") {
		t.Fatalf("footer dropped queue: %q", line)
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
	if got := stripANSI(app.taskQueueLabel()); got != "queue empty" {
		t.Fatalf("empty queue label = %q", got)
	}
	app.pending["a"] = model.Tracked
	app.pending["b"] = model.Tracked
	app.pending["c"] = model.Missed
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
	if !strings.Contains(markedRow, markFull) || strings.Contains(markedRow, markEmpty) {
		t.Fatalf("marked row should show %q only: %q", markFull, markedRow)
	}
	if !strings.Contains(unmarkedRow, markEmpty) || strings.Contains(unmarkedRow, markFull) {
		t.Fatalf("unmarked row should show %q only: %q", markEmpty, unmarkedRow)
	}
	if !strings.Contains(unmarkedRow, "▌") {
		t.Fatalf("cursor row lacks its accent gutter: %q", unmarkedRow)
	}
	if strings.Contains(markedRow, "▌") {
		t.Fatalf("a marked-only row must not look like the cursor: %q", markedRow)
	}
	if themeTokens.Surface.Light == themeTokens.Marked.Light || themeTokens.Surface.Dark == themeTokens.Marked.Dark {
		t.Fatal("cursor and marked rows share the same band colour")
	}
}

func TestParentRowsShowCaretAndMark(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks

	nodes := app.taskNodes()
	app.taskCursor = findTaskNode(nodes, nodeTask, "K901000")

	// Open term: guide, then caret, mark column, name.
	term := findTaskNode(nodes, nodeTerm, "smt11")
	if row := stripANSI(app.taskTreeRow(nodes, term, 98)); !strings.Contains(row, "▾ "+markEmpty+" smt11") {
		t.Fatalf("open term should show a down caret and hollow mark: %q", row)
	}

	// The demo tree's "closed" bucket starts folded.
	closed := findTaskNode(nodes, nodeBucket, "closed")
	row := stripANSI(app.taskTreeRow(nodes, closed, 98))
	if !strings.Contains(row, "▸ "+markEmpty+" closed") {
		t.Fatalf("folded bucket should show a right caret: %q", row)
	}

	// Marking every child turns the parent's mark on; a partial mark keeps it off.
	tracked := findTaskNode(nodes, nodeBucket, "tracked")
	ids := app.taskGroupIDs(nodes[tracked])
	for _, id := range ids {
		app.selected[id] = true
	}
	row = stripANSI(app.taskTreeRow(nodes, tracked, 98))
	if !strings.Contains(row, "▾ "+markFull+" tracked") {
		t.Fatalf("fully marked parent should show %q: %q", markFull, row)
	}
	delete(app.selected, ids[0])
	row = stripANSI(app.taskTreeRow(nodes, tracked, 98))
	if !strings.Contains(row, "▾ "+markEmpty+" tracked") {
		t.Fatalf("partially marked parent should show %q: %q", markEmpty, row)
	}
}

func TestCourseCountsKeepTheCursorBand(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	app := demoApp(t)
	app.tab = TabTasks
	nodes := app.taskNodes()
	course := findTaskNode(nodes, nodeCourse, "")
	app.taskCursor = course

	row := app.taskTreeRow(nodes, course, 98)
	if !strings.Contains(stripANSI(row), "○ 6") {
		t.Fatalf("course row is missing its state counts: %q", stripANSI(row))
	}
	// Every cell of a cursor row carries the band background. A reset followed
	// by literal spaces means the counts broke the band, which is the bug this
	// guards against.
	if strings.Contains(row, "\x1b[0m ") || strings.Contains(row, "\x1b[0m  ") {
		t.Fatalf("counts punched holes in the cursor band: %q", row)
	}
	if strings.Count(row, "48;") < 5 {
		t.Fatalf("cursor band is not painted across every cell: %q", row)
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
