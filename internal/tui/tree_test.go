package tui

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/source"
)

func demoApp(t *testing.T) *App {
	t.Helper()
	app := New(source.NewDemo(), config.Config{}, "test")
	msg := app.Init()()
	loaded, ok := msg.(loadedMsg)
	if !ok {
		t.Fatalf("Init() returned %T, want loadedMsg", msg)
	}
	app.courses = loaded.courses
	app.assignments = loaded.assignments
	app.materials = loaded.materials
	return app
}

func findTaskNode(nodes []node, kind nodeKind, value string) int {
	for i, n := range nodes {
		if n.kind != kind {
			continue
		}
		switch kind {
		case nodeTerm:
			if n.term == value {
				return i
			}
		case nodeCourse:
			if n.courseID == 18515 {
				return i
			}
		case nodeBucket:
			if n.bucket == value && n.courseID == 18515 {
				return i
			}
		case nodeTask:
			if n.task.ID == value {
				return i
			}
		case nodeTaskMaterial:
			if n.mat.ID == value {
				return i
			}
		}
	}
	return -1
}

func TestTaskTreeGroupsSortsAndFolds(t *testing.T) {
	app := demoApp(t)
	nodes := app.taskNodes()
	term := findTaskNode(nodes, nodeTerm, "smt11")
	course := findTaskNode(nodes, nodeCourse, "")
	needsDecision := findTaskNode(nodes, nodeBucket, "needs decision")
	tracked := findTaskNode(nodes, nodeBucket, "tracked")
	closed := findTaskNode(nodes, nodeBucket, "closed")
	if term != 0 || course < term || needsDecision < course || tracked < needsDecision || closed < tracked {
		t.Fatalf("unexpected tree ordering: term=%d course=%d needs=%d tracked=%d closed=%d", term, course, needsDecision, tracked, closed)
	}
	if findTaskNode(nodes, nodeTask, "K150009") >= 0 {
		t.Fatal("closed assignments should be collapsed by default")
	}
	app.expanded[bucketKey(18515, "closed")] = true
	nodes = app.taskNodes()
	if findTaskNode(nodes, nodeTask, "K150009") < 0 {
		t.Fatal("opening closed bucket should reveal its assignments")
	}
	app.expanded[taskKey("K165483")] = true
	nodes = app.taskNodes()
	if findTaskNode(nodes, nodeTaskMaterial, "M165483:modul6.pdf") < 0 {
		t.Fatal("opening assignment should reveal its attached material")
	}
}

func TestTaskTreeAssignmentsSortOldestFirstAndAssociateAttachments(t *testing.T) {
	app := demoApp(t)
	var ids []string
	for _, n := range app.taskNodes() {
		if n.kind == nodeTask && n.courseID == 18515 && n.bucket == "needs decision" {
			ids = append(ids, n.task.ID)
		}
	}
	if len(ids) < 2 || ids[0] != "K901000" || ids[1] != "K901001" {
		t.Fatalf("oldest undecided assignments not first: %v", ids[:minInt(2, len(ids))])
	}
	materials := app.assignmentMaterials(model.Assignment{ID: "K165483"})
	if len(materials) != 2 || materials[0].ID != "M165483:modul6.pdf" {
		t.Fatalf("unexpected assignment materials: %#v", materials)
	}
}

func TestTaskTreeConnectorsAndRowWidth(t *testing.T) {
	oldTerm := os.Getenv("TERM")
	t.Cleanup(func() { _ = os.Setenv("TERM", oldTerm) })
	_ = os.Setenv("TERM", "xterm-256color")
	app := demoApp(t)
	nodes := app.taskNodes()
	term := findTaskNode(nodes, nodeTerm, "smt11")
	course := findTaskNode(nodes, nodeCourse, "")
	if got := treeGuides(nodes, course); got != "├─ " {
		t.Fatalf("course connector = %q, want %q", got, "├─ ")
	}
	_ = os.Setenv("TERM", "dumb")
	if vertical, branch, last := treeConnectors(); vertical != "|  " || branch != "|-- " || last != "`-- " {
		t.Fatalf("ASCII connectors = %q, %q, %q", vertical, branch, last)
	}
	app.taskCursor = term
	row := app.taskTreeRow(nodes, term, 98)
	if got := lipgloss.Width(stripANSI(row)); got != 98 {
		t.Fatalf("rendered row width = %d, want 98", got)
	}
	if !strings.Contains(row, "smt11") {
		t.Fatalf("rendered term row missing term: %q", row)
	}
}

func TestCoursesTreeViewsAt100And80Columns(t *testing.T) {
	for _, width := range []int{100, 80} {
		app := demoApp(t)
		app.width = width
		app.height = 40
		app.layout = LayList
		app.taskCursor = 0
		view := app.View()
		for i, line := range strings.Split(view, "\n") {
			if got := lipgloss.Width(line); got != width {
				t.Fatalf("width %d line %d has width %d: %q", width, i+1, got, stripANSI(line))
			}
		}
		if !strings.Contains(view, "courses · all") || !strings.Contains(view, "smt11") || !strings.Contains(stripANSI(view), "queue empty") {
			t.Fatalf("%d-column view is missing expected content:\n%s", width, stripANSI(view))
		}
	}
}

func TestQuietTreeLineColorTracksForegroundAndBackground(t *testing.T) {
	for _, pair := range [][2]string{{"#1B1B26", "#E4E4E4"}, {"#EDEDED", "#333355"}} {
		got := quietHex(pair[0], pair[1])
		if got == pair[0] || got == pair[1] || len(got) != 7 {
			t.Fatalf("quietHex(%q, %q) = %q", pair[0], pair[1], got)
		}
	}
	if got := quietHex("bad", "#000000"); got != "#8A8A8A" {
		t.Fatalf("invalid colors fallback = %q", got)
	}
}

func TestTaskFooterKeepsPriorityAndMoreHint(t *testing.T) {
	full := taskFooterHints(200)
	want := "↑↓ move · h/l fold · space select · t/m/i decide · c commit · / filter · ? more"
	if got := stripANSI(full); got != want {
		t.Fatalf("full footer = %q, want %q", got, want)
	}
	short := taskFooterHints(12)
	if got := stripANSI(short); got != "? more" {
		t.Fatalf("short footer = %q, want ? more", got)
	}
	medium := stripANSI(taskFooterHints(40))
	if !strings.HasSuffix(medium, "? more") || strings.Contains(medium, "/ filter") {
		t.Fatalf("short footer did not drop hints from the right: %q", medium)
	}
}

func TestTaskTitleLegendFallback(t *testing.T) {
	full := stripANSI(taskTitleLine("courses · all · 42 assignments", 100))
	if !strings.Contains(full, "○ new") || !strings.Contains(full, "× missed") {
		t.Fatalf("full title legend missing: %q", full)
	}
	glyphs := stripANSI(taskTitleLine("courses · all · 42 assignments", 45))
	if !strings.Contains(glyphs, "○  ◐  ●  ◆  ×") || strings.Contains(glyphs, "new") {
		t.Fatalf("glyph-only title legend not selected: %q", glyphs)
	}
	hidden := stripANSI(taskTitleLine("courses · all · 42 assignments", 10))
	if strings.ContainsAny(hidden, "○◐●◆×") {
		t.Fatalf("title legend should be hidden: %q", hidden)
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

func TestDueAgeUsesCalendarDays(t *testing.T) {
	now := time.Date(2026, 10, 18, 23, 30, 0, 0, time.UTC)
	if got := dueAge(model.Assignment{HasDue: true, Due: time.Date(2026, 10, 17, 23, 55, 0, 0, time.UTC)}, now); got != "−1d" {
		t.Fatalf("past due age = %q, want −1d", got)
	}
	if got := dueAge(model.Assignment{HasDue: true, Due: time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)}, now); got != "+2d" {
		t.Fatalf("upcoming age = %q, want +2d", got)
	}
}
