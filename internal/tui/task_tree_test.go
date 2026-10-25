package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/ruyam/kamps/internal/model"
)

func TestTaskTreeGroupsSortsAndFolds(t *testing.T) {
	app := demoApp(t)
	nodes := app.taskNodes()
	if len(nodes) == 0 || nodes[0].kind != nodeCourse {
		t.Fatalf("courses should be the top-level rows: %#v", nodes)
	}

	// Courses are the only top-level rows and sort by their short name.
	var courseIDs []int
	for _, n := range nodes {
		if n.kind == nodeCourse {
			courseIDs = append(courseIDs, n.courseID)
		}
	}
	for i := 1; i < len(courseIDs); i++ {
		if app.shortCode(courseIDs[i-1]) > app.shortCode(courseIDs[i]) {
			t.Fatalf("courses are not sorted by short name: %v", courseIDs)
		}
	}

	course := findTaskNode(nodes, nodeCourse, "")
	if course < 0 {
		t.Fatal("ppl course row not found")
	}
	var buckets []string
	for _, n := range nodes[course+1:] {
		if n.kind == nodeCourse {
			break
		}
		if n.kind == nodeBucket {
			buckets = append(buckets, n.bucket)
		}
	}
	if len(buckets) != 3 || buckets[0] != "needs decision" || buckets[1] != "tracked" || buckets[2] != "closed" {
		t.Fatalf("ppl buckets = %v", buckets)
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

func TestTaskTreeIndentAndRowWidth(t *testing.T) {
	app := demoApp(t)
	nodes := app.taskNodes()
	course := findTaskNode(nodes, nodeCourse, "")
	if got := treeGuides(nodes, course); got != "" {
		t.Fatalf("top-level guide = %q, want empty", got)
	}
	app.taskCursor = course
	row := app.taskTreeRow(nodes, course, 98)
	if got := lipgloss.Width(stripANSI(row)); got != 98 {
		t.Fatalf("rendered row width = %d, want 98", got)
	}
	if !strings.Contains(row, "Proyek Perangkat Lunak A") {
		t.Fatalf("rendered course row missing course: %q", row)
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
		if !strings.Contains(view, "courses · all") || !strings.Contains(view, "Proyek Perangkat Lunak A") || strings.Contains(stripANSI(view), "other") {
			t.Fatalf("%d-column view is missing expected content:\n%s", width, stripANSI(view))
		}
	}
}

func TestHistoricalTreeLabelsRemainStateColored(t *testing.T) {
	if got := formatTaskCounts([5]int{6, 1, 0, 1, 1}, rowHighlight{}); stripANSI(got) != "○ 6  ◐ 1  ◆ 1  × 1" {
		t.Fatalf("task counters = %q", stripANSI(got))
	}
	if got := taskTreeRowForTest(t); !strings.Contains(stripANSI(got), "Latihan ERD") || !strings.Contains(stripANSI(got), "−50d") {
		t.Fatalf("assignment row missing name or due: %q", stripANSI(got))
	}
}

func TestTrackedAndClosedStartCollapsed(t *testing.T) {
	app := demoApp(t)
	app.expanded = map[string]bool{}
	nodes := app.taskNodes()
	for i, n := range nodes {
		if n.kind != nodeBucket || (n.bucket != "tracked" && n.bucket != "closed") {
			continue
		}
		for j := i + 1; j < len(nodes) && nodes[j].depth() > n.depth(); j++ {
			if nodes[j].kind == nodeTask {
				t.Fatalf("%s bucket should start collapsed", n.bucket)
			}
		}
	}
}
