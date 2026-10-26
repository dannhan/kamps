package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/source"
)

func TestTreePrefixSiblingConnectors(t *testing.T) {
	c1 := node{kind: nodeCourse, courseID: 1, key: courseKey(1)}
	b1 := node{kind: nodeBucket, courseID: 1, bucket: "needs decision", key: bucketKey(1, "needs decision"), parents: []string{courseKey(1)}}
	t1 := node{kind: nodeTask, courseID: 1, task: model.Assignment{ID: "A1"}, key: taskKey("A1"), parents: []string{courseKey(1), b1.key}}
	t2 := node{kind: nodeTask, courseID: 1, task: model.Assignment{ID: "A2"}, key: taskKey("A2"), parents: []string{courseKey(1), b1.key}}
	b2 := node{kind: nodeBucket, courseID: 1, bucket: "tracked", key: bucketKey(1, "tracked"), parents: []string{courseKey(1)}}
	t3 := node{kind: nodeTask, courseID: 1, task: model.Assignment{ID: "A3"}, key: taskKey("A3"), parents: []string{courseKey(1), b2.key}}
	c2 := node{kind: nodeCourse, courseID: 2, key: courseKey(2)}
	b3 := node{kind: nodeBucket, courseID: 2, bucket: "needs decision", key: bucketKey(2, "needs decision"), parents: []string{courseKey(2)}}
	t4 := node{kind: nodeTask, courseID: 2, task: model.Assignment{ID: "B1"}, key: taskKey("B1"), parents: []string{courseKey(2), b3.key}}
	nodes := []node{c1, b1, t1, t2, b2, t3, c2, b3, t4}

	want := []string{
		"",
		"├─ ",
		"│  ├─ ",
		"│  └─ ",
		"└─ ",
		"   └─ ",
		"",
		"└─ ",
		"   └─ ",
	}
	for i, expected := range want {
		if got := treePrefix(nodes, i); got != expected {
			t.Errorf("treePrefix(%d) = %q, want %q", i, got, expected)
		}
	}
}

type treeFixedSource struct{ snapshot source.Snapshot }

func (s treeFixedSource) Label() string { return "tree-test" }

func (s treeFixedSource) Load(context.Context) (source.Snapshot, error) {
	return s.snapshot, nil
}

func treeSampleApp(t *testing.T, now time.Time) *App {
	t.Helper()
	at := func(days int) time.Time {
		day := now.AddDate(0, 0, days)
		return time.Date(day.Year(), day.Month(), day.Day(), 12, 0, 0, 0, day.Location())
	}
	snapshot := source.Snapshot{
		Courses: []model.Course{
			{ID: 1, Code: "C1", Short: "c1", Name: "Course One", Term: "smt1"},
			{ID: 2, Code: "C2", Short: "c2", Name: "Course Two", Term: "smt1"},
		},
		Assignments: []model.Assignment{
			{ID: "A1", CourseID: 1, CourseCode: "c1", Name: "Alpha Assignment", Due: at(-35), HasDue: true, Triage: model.Undecided},
			{ID: "A2", CourseID: 1, CourseCode: "c1", Name: "Beta", Due: at(-2), HasDue: true, Triage: model.Undecided},
			{ID: "A3", CourseID: 1, CourseCode: "c1", Name: "Gamma Quiz", Due: at(4), HasDue: true, Triage: model.Undecided},
			{ID: "A4", CourseID: 1, CourseCode: "c1", Name: "Delta", Triage: model.Undecided},
			{ID: "T1", CourseID: 1, CourseCode: "c1", Name: "Tracked One", Due: at(10), HasDue: true, Triage: model.Tracked},
			{ID: "M1", CourseID: 1, CourseCode: "c1", Name: "Missed One", Due: at(-20), HasDue: true, Triage: model.Missed},
			{ID: "I1", CourseID: 1, CourseCode: "c1", Name: "Ignored One", Due: at(-30), HasDue: true, Triage: model.Ignored},
			{ID: "B1", CourseID: 2, CourseCode: "c2", Name: "Course Two Task", Due: at(1), HasDue: true, Triage: model.Undecided},
		},
		Materials: []model.Material{
			{ID: "MA1:slides.pdf", CourseID: 1, CourseCode: "c1", Name: "slides.pdf", Section: "week-1", Kind: "file"},
		},
	}
	app := New(treeFixedSource{snapshot: snapshot}, config.Config{}, "test")
	app.now = func() time.Time { return now }
	app.Update(app.Init()())
	app.tab = TabTasks
	for _, courseID := range []int{1, 2} {
		for _, bucket := range []string{"needs decision", "tracked", "closed"} {
			app.expanded[bucketKey(courseID, bucket)] = true
		}
	}
	app.taskCursor = -1
	return app
}

func TestTreeSampleLayout(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct{ width, height int }{{80, 24}, {120, 40}} {
		app := treeSampleApp(t, now)
		app.width, app.height = tc.width, tc.height
		nodes := app.taskNodes()
		rows := app.taskRows(nodes, tc.width, tc.height)
		if len(rows) != len(nodes) {
			t.Fatalf("%dx%d rendered %d of %d rows", tc.width, tc.height, len(rows), len(nodes))
		}
		plain := make([]string, len(rows))
		for i, row := range rows {
			plain[i] = stripANSI(row)
		}

		// (a) exact connector sequence for the known sample tree.
		var prefix []string
		for i := range nodes {
			prefix = append(prefix, treePrefix(nodes, i))
		}
		wantPrefix := []string{
			"", "├─ ", "│  ├─ ", "│  ├─ ", "│  ├─ ", "│  └─ ",
			"├─ ", "│  └─ ",
			"└─ ", "   ├─ ", "   └─ ",
			"", // spacer above Course Two
			"", "└─ ", "   └─ ",
		}
		// The sample has 15 visible nodes including the spacer above Course Two.
		if len(prefix) != len(wantPrefix) {
			t.Fatalf("%dx%d node count %d, want %d: %v", tc.width, tc.height, len(prefix), len(wantPrefix), prefix)
		}
		for i := range wantPrefix {
			if prefix[i] != wantPrefix[i] {
				t.Errorf("%dx%d prefix[%d] = %q, want %q", tc.width, tc.height, i, prefix[i], wantPrefix[i])
			}
		}

		// (b) every row at the same depth starts its title at the same column.
		titleColumns := map[int]int{}
		for i, n := range nodes {
			title := treeNodeTitle(app, n)
			if title == "" {
				continue
			}
			byteIndex := strings.Index(plain[i], title)
			if byteIndex < 0 {
				t.Fatalf("%dx%d row %q is missing title %q", tc.width, tc.height, plain[i], title)
			}
			column := lipgloss.Width(plain[i][:byteIndex])
			depth := n.depth()
			if want, ok := titleColumns[depth]; ok && column != want {
				t.Errorf("%dx%d depth %d title %q starts at %d, want %d", tc.width, tc.height, depth, title, column, want)
			} else {
				titleColumns[depth] = column
			}
		}

		// (c) the due column has one right edge and a one-cell flag slot.
		rightEdge := -1
		for i, n := range nodes {
			if n.kind != nodeTask {
				continue
			}
			trimmed := strings.TrimRight(plain[i], " ")
			edge := lipgloss.Width(trimmed)
			if rightEdge == -1 {
				rightEdge = edge
			} else if edge != rightEdge {
				t.Errorf("%dx%d due row %q ends at %d, want %d", tc.width, tc.height, trimmed, edge, rightEdge)
			}
			if want := tc.width - treeRowRightPad; edge != want {
				t.Errorf("%dx%d due row ends at %d, want %d", tc.width, tc.height, edge, want)
			}
			cell := tailCells(trimmed, dueColumnWidth)
			if got := lipgloss.Width(cell); got != dueColumnWidth {
				t.Fatalf("%dx%d due cell %q width %d, want %d", tc.width, tc.height, cell, got, dueColumnWidth)
			}
			flag := " "
			if n.task.HasDue && n.task.Due.Before(now) {
				flag = "!"
			}
			if string([]rune(cell)[0]) != flag {
				t.Errorf("%dx%d due cell %q flag = %q, want %q", tc.width, tc.height, cell, string([]rune(cell)[0]), flag)
			}
		}

		// Golden snapshot of the tree at this size.
		assertTreeGolden(t, fmt.Sprintf("tasks-%dx%d.txt", tc.width, tc.height), trimRightLines(plain))
	}
}

func treeNodeTitle(app *App, n node) string {
	switch n.kind {
	case nodeCourse:
		if course, ok := app.courseByID(n.courseID); ok && course.Name != "" {
			return course.Name
		}
		return app.shortCode(n.courseID)
	case nodeBucket:
		return fmt.Sprintf("%s (%d)", n.bucket, len(app.taskGroupIDs(n)))
	case nodeTask:
		return taskDisplayName(n.task.Name)
	}
	return ""
}

func TestCourseSpacersAreNotSelectable(t *testing.T) {
	app := treeSampleApp(t, time.Now())
	nodes := app.taskNodes()
	spacer, courseTwo := -1, -1
	for i, n := range nodes {
		if n.kind == nodeSpacer {
			spacer = i
		}
		if n.kind == nodeCourse && n.courseID == 2 {
			courseTwo = i
		}
	}
	if spacer < 0 || courseTwo != spacer+1 {
		t.Fatalf("spacer=%d courseTwo=%d nodes=%d", spacer, courseTwo, len(nodes))
	}
	if nodes[0].kind != nodeCourse {
		t.Fatal("the first course should not have a spacer above it")
	}
	if row := stripANSI(app.taskTreeRow(nodes, spacer, 80)); strings.TrimSpace(row) != "" {
		t.Fatalf("spacer row is not blank: %q", row)
	}

	app.taskCursor = spacer
	app.clampCursor()
	if nodes[app.taskCursor].kind == nodeSpacer {
		t.Fatal("clampCursor left the cursor on a spacer")
	}

	app.taskCursor = spacer - 1
	app.move(1)
	if app.taskCursor != courseTwo {
		t.Fatalf("move down from course one ended at %d, want course two %d", app.taskCursor, courseTwo)
	}

	app.taskCursor = courseTwo
	app.move(-1)
	if nodes[app.taskCursor].kind == nodeSpacer {
		t.Fatal("move up from course two landed on the spacer")
	}

	app.moveTop()
	if nodes[app.taskCursor].kind == nodeSpacer {
		t.Fatal("moveTop landed on a spacer")
	}
	app.moveBottom()
	if nodes[app.taskCursor].kind == nodeSpacer {
		t.Fatal("moveBottom landed on a spacer")
	}

	app.taskCursor = courseTwo
	app.toggleSelect()
	if !app.selected["B1"] {
		t.Fatal("course header is no longer selectable")
	}
	app.moveX(-1)
	if app.taskIsOpen(courseKey(2), true) {
		t.Fatal("course header is no longer collapsible")
	}
	app.moveX(1)
	if !app.taskIsOpen(courseKey(2), true) {
		t.Fatal("course header is no longer expandable")
	}
}

func TestCourseRuleDropsBelow60(t *testing.T) {
	app := treeSampleApp(t, time.Now())
	nodes := app.taskNodes()
	course := -1
	for i, n := range nodes {
		if n.kind == nodeCourse && n.courseID == 1 {
			course = i
			break
		}
	}
	if course < 0 {
		t.Fatal("sample tree has no first course")
	}

	narrow := stripANSI(app.taskTreeRow(nodes, course, 59))
	if strings.Contains(narrow, "─") {
		t.Fatalf("59-column course header kept the rule: %q", narrow)
	}
	if !strings.Contains(narrow, "Course One") || !strings.Contains(narrow, "3/7") {
		t.Fatalf("59-column course header lost content: %q", narrow)
	}

	wide := stripANSI(app.taskTreeRow(nodes, course, 60))
	if !strings.Contains(wide, "─") || !strings.Contains(wide, "3/7") {
		t.Fatalf("60-column course header is missing the rule or progress: %q", wide)
	}
}

func TestTreeHidesTypeSuffixAndDetailKeepsIt(t *testing.T) {
	app := treeSampleApp(t, time.Now())
	nodes := app.taskNodes()
	for i, n := range nodes {
		if n.kind != nodeTask || n.task.ID != "A1" {
			continue
		}
		row := stripANSI(app.taskTreeRow(nodes, i, 120))
		if strings.Contains(row, "Alpha Assignment") || !strings.Contains(row, "Alpha") {
			t.Fatalf("tree row kept the type suffix: %q", row)
		}
		detail := stripANSI(strings.Join(app.detailLines(n, 60), "\n"))
		if !strings.Contains(detail, "Alpha Assignment") {
			t.Fatalf("detail pane lost the full name: %q", detail)
		}
		return
	}
	t.Fatal("sample tree has no A1 task")
}

func TestTaskDisplayName(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"Submission Praktikum 4 Assignment", "Submission Praktikum 4"},
		{"QUIS 01 Quiz", "QUIS 01"},
		{"Midterm Assignment Review", "Midterm Assignment Review"},
		{"Assignment", "Assignment"},
		{"  Spaced Assignment  ", "Spaced"},
	} {
		if got := taskDisplayName(tc.in); got != tc.want {
			t.Errorf("taskDisplayName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func tailCells(s string, width int) string {
	runes := []rune(s)
	used, start := 0, len(runes)
	for i := len(runes) - 1; i >= 0 && used < width; i-- {
		used += lipgloss.Width(string(runes[i]))
		start = i
	}
	return string(runes[start:])
}

func trimRightLines(lines []string) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = strings.TrimRight(line, " ")
	}
	return out
}

func assertTreeGolden(t *testing.T, name string, lines []string) {
	t.Helper()
	path := filepath.Join("testdata", name)
	got := strings.Join(lines, "\n") + "\n"
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v", path, err)
	}
	if got != string(want) {
		t.Errorf("%s snapshot mismatch\n--- got ---\n%s--- want ---\n%s", name, got, want)
	}
}
