package tui

import (
	"os"
	"reflect"
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
	app.loaded = true
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

func TestSharedFooterKeepsQueueWhenHintsDoNotFit(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks
	line := stripANSI(app.sharedFooter(24, []string{"↑↓ move", "h/l fold", "space select", "t/m/i decide", "c commit", "/ filter"}))
	if !strings.HasSuffix(line, "queue empty") {
		t.Fatalf("shared footer dropped queue: %q", line)
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

func TestSharedShellAllTabsAtRequestedSizes(t *testing.T) {
	for _, size := range [][2]int{{100, 30}, {80, 24}} {
		var reference []int
		for active := TabTasks; active < tabCount; active++ {
			app := demoApp(t)
			app.width, app.height = size[0], size[1]
			app.tab = active
			if active == TabCommit {
				app.pending = map[string]model.Triage{"K901000": model.Tracked}
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
	if got := stripANSI(app.View()); !strings.Contains(got, "terminal too small") {
		t.Fatalf("small-terminal message missing: %q", got)
	}
}

func TestCommitPreviewContainsGroupedExactLinesAndMarkers(t *testing.T) {
	app := demoApp(t)
	app.width, app.height = 100, 30
	app.tab = TabCommit
	app.pending = map[string]model.Triage{
		"K901000": model.Tracked,
		"K901001": model.Missed,
		"K901002": model.Ignored,
		"K163827": model.Tracked,
	}
	view := stripANSI(app.View())
	for _, want := range []string{"done (track)", "missed", "ignore", "K901000 · ppl · Progress Report 1", "- [ ] [ppl] Progress Report 1 (K901000)", "will show as OVERDUE", `similar manual task: "Submit ppl Report 5"`} {
		if !strings.Contains(view, want) {
			t.Errorf("commit preview missing %q:\n%s", want, view)
		}
	}
}

func TestMaterialsTreeHasStateBucketsAndAttachedAssignments(t *testing.T) {
	app := demoApp(t)
	app.tab = TabMaterials
	nodes := app.matNodes()
	foundBucket, foundAssignment := false, false
	for _, n := range nodes {
		foundBucket = foundBucket || n.kind == nodeMaterialBucket
		foundAssignment = foundAssignment || n.kind == nodeMaterialAssignment && n.task.ID == "K165483"
	}
	if !foundBucket || !foundAssignment {
		t.Fatalf("Materials tree lacks state bucket or attached assignment: bucket=%v assignment=%v", foundBucket, foundAssignment)
	}
	app.filter = "downloaded"
	for _, n := range app.matNodes() {
		if n.kind == nodeFile && !n.mat.Downloaded {
			t.Fatalf("downloaded filter included new material %q", n.mat.Name)
		}
	}
}

func TestHistoricalTreeLabelsRemainStateColored(t *testing.T) {
	if got := formatTaskCounts([5]int{6, 1, 0, 1, 1}); stripANSI(got) != "○ 6  ◐ 1  ◆ 1  × 1" {
		t.Fatalf("task counters = %q", stripANSI(got))
	}
	if got := taskTreeRowForTest(t); !strings.Contains(stripANSI(got), "new · K901013 ·") {
		t.Fatalf("assignment metadata missing: %q", stripANSI(got))
	}
}

func taskTreeRowForTest(t *testing.T) string {
	t.Helper()
	app := demoApp(t)
	app.taskCursor = findTaskNode(app.taskNodes(), nodeTask, "K901013")
	return app.taskTreeRow(app.taskNodes(), app.taskCursor, 98)
}

func TestTabSwitchRestoresPerTabCursorAndFilter(t *testing.T) {
	app := demoApp(t)
	app.taskCursor = 7
	app.filter = "tracked"
	app.tabFilters[TabTasks] = "tracked"
	app.switchTab(TabMaterials)
	app.matCursor = 4
	app.switchTab(TabTasks)
	if app.taskCursor != 7 || app.filter != "tracked" {
		t.Fatalf("Tasks state not restored: cursor=%d filter=%q", app.taskCursor, app.filter)
	}
	app.switchTab(TabMaterials)
	if app.matCursor != 4 {
		t.Fatalf("Materials cursor not restored: %d", app.matCursor)
	}
}

func TestFoldingOpenGroupKeepsCursorOnIt(t *testing.T) {
	app := demoApp(t)
	app.layout = LayList
	app.tab = TabTasks

	nodes := app.taskNodes()
	tracked := findTaskNode(nodes, nodeBucket, "tracked")
	if tracked < 0 {
		t.Fatal("tracked bucket missing from the demo tree")
	}
	app.taskCursor = tracked
	app.moveX(-1)
	if app.taskCursor != tracked {
		t.Fatalf("folding an open bucket moved the cursor: got %d, want %d", app.taskCursor, tracked)
	}
	if findTaskNode(app.taskNodes(), nodeBucket, "tracked") < 0 {
		t.Fatal("the bucket the cursor stayed on vanished from the tree")
	}
}

func TestFoldOnLeafMovesCursorToParent(t *testing.T) {
	app := demoApp(t)
	app.layout = LayList
	app.tab = TabTasks

	nodes := app.taskNodes()
	task := findTaskNode(nodes, nodeTask, "K901000")
	if task < 0 {
		t.Fatal("assignment K901000 missing from the default tree")
	}
	expected := findTaskNode(nodes, nodeBucket, "needs decision")
	app.taskCursor = task
	app.moveX(-1)
	if app.taskCursor != expected {
		t.Fatalf("h on a leaf moved the cursor to %d, want parent bucket at %d", app.taskCursor, expected)
	}
}

func TestFoldOnAlreadyFoldedGroupMovesCursorToParent(t *testing.T) {
	app := demoApp(t)
	app.layout = LayList
	app.tab = TabTasks

	app.expanded[courseKey(18515)] = false
	nodes := app.taskNodes()
	course := findTaskNode(nodes, nodeCourse, "")
	expected := findTaskNode(nodes, nodeTerm, "smt11")
	if course < 0 || expected != 0 {
		t.Fatalf("unexpected folded tree: course=%d term=%d", course, expected)
	}
	app.taskCursor = course
	app.moveX(-1)
	if app.taskCursor != expected {
		t.Fatalf("h on an already-folded course moved the cursor to %d, want term at %d", app.taskCursor, expected)
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
