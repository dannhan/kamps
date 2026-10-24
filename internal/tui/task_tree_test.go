package tui

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/ruyam/kamps/internal/model"
)

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

func TestHistoricalTreeLabelsRemainStateColored(t *testing.T) {
	if got := formatTaskCounts([5]int{6, 1, 0, 1, 1}, rowHighlight{}); stripANSI(got) != "○ 6  ◐ 1  ◆ 1  × 1" {
		t.Fatalf("task counters = %q", stripANSI(got))
	}
	if got := taskTreeRowForTest(t); !strings.Contains(stripANSI(got), "new · K901013 ·") {
		t.Fatalf("assignment metadata missing: %q", stripANSI(got))
	}
}
