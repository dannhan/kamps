package tui

import (
	"slices"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/source"
)

func workFirstNow() time.Time {
	return time.Date(2026, 10, 26, 10, 0, 0, 0, time.UTC)
}

func dueOn(now time.Time, days int) time.Time {
	day := now.AddDate(0, 0, days)
	return time.Date(day.Year(), day.Month(), day.Day(), 12, 0, 0, 0, time.UTC)
}

func workFirstApp(t *testing.T, now time.Time, courses []model.Course, assignments []model.Assignment) *App {
	t.Helper()
	app := New(treeFixedSource{snapshot: source.Snapshot{Courses: courses, Assignments: assignments}}, config.Config{}, "test")
	app.now = func() time.Time { return now }
	app.Update(app.Init()())
	app.tab = TabTasks
	app.taskCursor = -1
	return app
}

func findNodeByKey(nodes []node, kind nodeKind, key string) int {
	for i, n := range nodes {
		if n.kind == kind && n.key == key {
			return i
		}
	}
	return -1
}

func findBucketNode(nodes []node, courseID int, bucket string) int {
	for i, n := range nodes {
		if n.kind == nodeBucket && n.courseID == courseID && n.bucket == bucket {
			return i
		}
	}
	return -1
}

func findTaskByID(nodes []node, id string) int {
	for i, n := range nodes {
		if n.kind == nodeTask && n.task.ID == id {
			return i
		}
	}
	return -1
}

func actNowIDs(app *App) []string {
	var ids []string
	for _, n := range app.taskNodes() {
		if n.kind == nodeTask && n.bucket == "act now" {
			ids = append(ids, n.task.ID)
		}
	}
	return ids
}

func visibleIDs(app *App) map[string]bool {
	ids := make(map[string]bool)
	for _, a := range app.visible() {
		ids[a.ID] = true
	}
	return ids
}

func taskNodeIDs(nodes []node) []string {
	var ids []string
	for _, n := range nodes {
		if n.kind == nodeTask {
			ids = append(ids, n.task.ID)
		}
	}
	return ids
}

func TestClassifyTask(t *testing.T) {
	now := time.Date(2026, 10, 26, 10, 0, 0, 0, time.UTC)
	due := func(days int) time.Time {
		return time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC).AddDate(0, 0, days)
	}
	cases := []struct {
		name string
		task model.Assignment
		want taskGroup
	}{
		{"stale -8", model.Assignment{Name: "stale", HasDue: true, Due: due(-8)}, taskGroupStale},
		{"look-behind -7", model.Assignment{Name: "edge", HasDue: true, Due: due(-7)}, taskGroupActNow},
		{"today 0", model.Assignment{Name: "today", HasDue: true, Due: due(0)}, taskGroupActNow},
		{"look-ahead +7", model.Assignment{Name: "edge", HasDue: true, Due: due(7)}, taskGroupActNow},
		{"beyond +8", model.Assignment{Name: "later", HasDue: true, Due: due(8)}, taskGroupNeeds},
		{"no due", model.Assignment{Name: "someday"}, taskGroupNeeds},
		{"tracked due today", model.Assignment{Name: "done", HasDue: true, Due: due(0), Triage: model.Tracked}, taskGroupTracked},
		{"missed very late", model.Assignment{Name: "missed", HasDue: true, Due: due(-100), Triage: model.Missed}, taskGroupClosed},
		{"ignored very late", model.Assignment{Name: "ignored", HasDue: true, Due: due(-100), Triage: model.Ignored}, taskGroupClosed},
	}
	for _, tc := range cases {
		if got := classifyTask(tc.task, now); got != tc.want {
			t.Errorf("%s: classifyTask = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestActNowStaleBoundary(t *testing.T) {
	now := time.Date(2026, 10, 26, 10, 0, 0, 0, time.UTC)
	at := func(days int) time.Time {
		return time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC).AddDate(0, 0, days)
	}
	if got := classifyTask(model.Assignment{HasDue: true, Due: at(-8)}, now); got != taskGroupStale {
		t.Fatalf("-8 days = %v, want stale", got)
	}
	if got := classifyTask(model.Assignment{HasDue: true, Due: at(8)}, now); got != taskGroupNeeds {
		t.Fatalf("+8 days = %v, want needs decision", got)
	}
}

func TestActNowSortOrder(t *testing.T) {
	now := workFirstNow()
	course := model.Course{ID: 1, Short: "c1", Name: "Course One"}
	items := []struct {
		id   string
		days int
	}{
		{"m7", -7}, {"m2", -2}, {"d0", 0}, {"p1", 1}, {"p7", 7},
	}
	var assignments []model.Assignment
	for _, item := range items {
		assignments = append(assignments, model.Assignment{
			ID: item.id, CourseID: 1, Name: item.id, HasDue: true, Due: dueOn(now, item.days),
		})
	}
	app := workFirstApp(t, now, []model.Course{course}, assignments)
	if got, want := actNowIDs(app), []string{"m7", "m2", "d0", "p1", "p7"}; !slices.Equal(got, want) {
		t.Fatalf("Act-now order = %v, want %v", got, want)
	}
}

func TestActNowStaleBoundarySections(t *testing.T) {
	now := workFirstNow()
	course := model.Course{ID: 1, Short: "c1", Name: "Course One"}
	assignments := []model.Assignment{
		{ID: "stale", CourseID: 1, Name: "stale", HasDue: true, Due: dueOn(now, -8)},
		{ID: "next", CourseID: 1, Name: "next", HasDue: true, Due: dueOn(now, 8)},
		{ID: "actLate", CourseID: 1, Name: "actLate", HasDue: true, Due: dueOn(now, -7)},
		{ID: "actEarly", CourseID: 1, Name: "actEarly", HasDue: true, Due: dueOn(now, 7)},
	}
	app := workFirstApp(t, now, []model.Course{course}, assignments)
	nodes := app.taskNodes()
	if got := actNowIDs(app); !slices.Equal(got, []string{"actLate", "actEarly"}) {
		t.Fatalf("Act-now IDs = %v", got)
	}
	stale := findBucketNode(nodes, 1, "stale overdue")
	needs := findBucketNode(nodes, 1, "needs decision")
	if stale < 0 || needs < 0 {
		t.Fatalf("missing stale/needs buckets: stale=%d needs=%d", stale, needs)
	}
	if got := app.taskGroupIDs(nodes[stale]); !slices.Equal(got, []string{"stale"}) {
		t.Fatalf("stale IDs = %v", got)
	}
	if got := app.taskGroupIDs(nodes[needs]); !slices.Equal(got, []string{"next"}) {
		t.Fatalf("needs IDs = %v", got)
	}
	if findTaskByID(nodes, "stale") >= 0 {
		t.Fatal("stale task should start collapsed")
	}
	if findTaskByID(nodes, "next") < 0 {
		t.Fatal("+8 task should be visible in needs decision")
	}
}

func TestActNowCoursePrefix(t *testing.T) {
	for _, tc := range []struct{ name, want string }{
		{"Interaksi Manusia dan Komputer A", "IMK"},
		{"Metode Numerik A", "MN"},
		{"Proyek Perangkat Lunak A", "PPL"},
		{"Basis Data C", "BD"},
		{"Struktur Data B", "SD"},
	} {
		if got := courseInitials(tc.name); got != tc.want {
			t.Errorf("courseInitials(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
	courses := []model.Course{
		{ID: 1, Short: "basdat", Name: "Basis Data C"},
		{ID: 2, Name: "Interaksi Manusia dan Komputer A"},
		{ID: 3, Name: "lowercase"},
	}
	app := workFirstApp(t, workFirstNow(), courses, nil)
	if got := app.actNowCoursePrefix(1); got != "basdat" {
		t.Fatalf("configured alias = %q, want basdat", got)
	}
	if got := app.actNowCoursePrefix(2); got != "IMK" {
		t.Fatalf("initials fallback = %q, want IMK", got)
	}
	if got := pad(app.actNowCoursePrefix(3), actNowPrefixWidth, false); got != "lowerca…" {
		t.Fatalf("truncate fallback = %q, want lowerca…", got)
	}
}

func TestActNowHeaderMarker(t *testing.T) {
	app := treeSampleApp(t, workFirstNow())
	nodes := app.taskNodes()
	actNow := findNodeByKey(nodes, nodeActNow, actNowKey)
	course := findNodeByKey(nodes, nodeCourse, courseKey(1))
	if actNow < 0 || course < 0 {
		t.Fatalf("missing act-now or course row: %d/%d", actNow, course)
	}
	openAct := stripANSI(app.taskTreeRow(nodes, actNow, 120))
	openCourse := stripANSI(app.taskTreeRow(nodes, course, 120))
	if !strings.Contains(openAct, "▾") || !strings.Contains(openCourse, "▾") {
		t.Fatalf("open headers should both use ▾: act=%q course=%q", openAct, openCourse)
	}
	app.expanded[actNowKey] = false
	app.expanded[courseKey(1)] = false
	nodes = app.taskNodes()
	actNow = findNodeByKey(nodes, nodeActNow, actNowKey)
	course = findNodeByKey(nodes, nodeCourse, courseKey(1))
	closedAct := stripANSI(app.taskTreeRow(nodes, actNow, 120))
	closedCourse := stripANSI(app.taskTreeRow(nodes, course, 120))
	if !strings.Contains(closedAct, "▸") || !strings.Contains(closedCourse, "▸") {
		t.Fatalf("folded headers should both use ▸: act=%q course=%q", closedAct, closedCourse)
	}
}

func TestActNowPartition(t *testing.T) {
	app := treeSampleApp(t, workFirstNow())
	nodes := app.taskNodes()
	visible := visibleIDs(app)
	partition := make(map[string]int)
	for _, n := range nodes {
		switch n.kind {
		case nodeActNow, nodeCourse:
			for _, id := range app.taskGroupIDs(n) {
				partition[id]++
			}
		case nodeCoursePlaceholder:
			if ids := app.taskGroupIDs(n); len(ids) != 0 {
				t.Fatalf("placeholder %s has task IDs %v", n.key, ids)
			}
		}
	}
	if len(partition) != len(visible) {
		t.Fatalf("partition has %d IDs, visible has %d", len(partition), len(visible))
	}
	for id := range visible {
		if partition[id] != 1 {
			t.Fatalf("visible ID %s appears %d times in the partition", id, partition[id])
		}
	}
	rendered := make(map[string]bool)
	for _, n := range nodes {
		if n.kind != nodeTask {
			continue
		}
		if rendered[n.task.ID] {
			t.Fatalf("task %s rendered twice", n.task.ID)
		}
		if !visible[n.task.ID] {
			t.Fatalf("task %s is rendered but not visible", n.task.ID)
		}
		rendered[n.task.ID] = true
	}
	if len(rendered) != len(visible) {
		t.Fatalf("rendered %d tasks, visible %d", len(rendered), len(visible))
	}
	placeholder := findNodeByKey(nodes, nodeCoursePlaceholder, courseKey(2))
	if placeholder < 0 {
		t.Fatal("course with only Act-now items lost its placeholder header")
	}
	if selectableNode(nodes[placeholder]) {
		t.Fatal("the all-in-Act-now placeholder must not be selectable")
	}
	row := stripANSI(app.taskTreeRow(nodes, placeholder, 120))
	if !strings.Contains(row, "all in Act now") || !strings.Contains(row, "0/1") {
		t.Fatalf("placeholder header lost its suffix or progress: %q", row)
	}
}

func TestStaleCollapsedByDefault(t *testing.T) {
	now := workFirstNow()
	course := model.Course{ID: 1, Short: "c1", Name: "Course One"}
	assignments := []model.Assignment{
		{ID: "S1", CourseID: 1, Name: "Stale One", HasDue: true, Due: dueOn(now, -8)},
		{ID: "S2", CourseID: 1, Name: "Stale Two", HasDue: true, Due: dueOn(now, -10)},
		{ID: "N1", CourseID: 1, Name: "Later", HasDue: true, Due: dueOn(now, 8)},
	}
	app := workFirstApp(t, now, []model.Course{course}, assignments)
	nodes := app.taskNodes()
	stale := findBucketNode(nodes, 1, "stale overdue")
	if stale < 0 {
		t.Fatal("stale bucket missing")
	}
	if findTaskByID(nodes, "S1") >= 0 {
		t.Fatal("stale child should start collapsed")
	}
	if findTaskByID(nodes, "N1") < 0 {
		t.Fatal("needs-decision child should be visible")
	}
	app.taskCursor = stale
	app.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	nodes = app.taskNodes()
	if findTaskByID(nodes, "S1") < 0 || findTaskByID(nodes, "S2") < 0 {
		t.Fatal("enter did not expand the stale bucket")
	}
	if nodes[app.taskCursor].key != bucketKey(1, "stale overdue") {
		t.Fatalf("cursor left the stale header: %s", nodes[app.taskCursor].key)
	}
	app.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	nodes = app.taskNodes()
	if findTaskByID(nodes, "S1") >= 0 {
		t.Fatal("enter did not collapse the stale bucket")
	}
}

func TestStaleHeaderBulk(t *testing.T) {
	now := workFirstNow()
	course := model.Course{ID: 1, Short: "c1", Name: "Course One"}
	assignments := []model.Assignment{
		{ID: "S1", CourseID: 1, Name: "Stale One", HasDue: true, Due: dueOn(now, -8)},
		{ID: "S2", CourseID: 1, Name: "Stale Two", HasDue: true, Due: dueOn(now, -9)},
	}
	app := workFirstApp(t, now, []model.Course{course}, assignments)
	nodes := app.taskNodes()
	stale := findBucketNode(nodes, 1, "stale overdue")
	app.taskCursor = stale
	app.handleKey(tea.KeyMsg{Type: tea.KeySpace})
	if len(app.selected) != 2 {
		t.Fatalf("space selected %d stale IDs, want 2", len(app.selected))
	}
	if got := stripANSI(app.selectionLabel()); !strings.Contains(got, "2 selected, 2 hidden") {
		t.Fatalf("selection label = %q", got)
	}
	hints := stripANSI(renderHintItems(app.footerHintItems(), 200))
	for _, want := range []string{"enter expand", "space select", "t/m/i decide 2"} {
		if !strings.Contains(hints, want) {
			t.Fatalf("stale header footer %q is missing %q", hints, want)
		}
	}
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if !app.triageArmed || app.byID("S1").Triage != model.Undecided {
		t.Fatalf("first t should only arm: armed=%v triage=%v", app.triageArmed, app.byID("S1").Triage)
	}
	if prompt := app.triagePrompt(); !strings.Contains(prompt, "press t again") || !strings.Contains(prompt, "2 hidden") {
		t.Fatalf("confirmation prompt = %q", prompt)
	}
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if app.triageArmed || app.byID("S1").Triage != model.Tracked || app.byID("S2").Triage != model.Tracked {
		t.Fatalf("second t did not apply to the whole stale header: armed=%v S1=%v S2=%v", app.triageArmed, app.byID("S1").Triage, app.byID("S2").Triage)
	}
}

func TestHiddenSelectionConfirm(t *testing.T) {
	now := workFirstNow()
	course := model.Course{ID: 1, Short: "c1", Name: "Course One"}
	assignments := []model.Assignment{
		{ID: "S1", CourseID: 1, Name: "Stale", HasDue: true, Due: dueOn(now, -8)},
		{ID: "A1", CourseID: 1, Name: "Today", HasDue: true, Due: dueOn(now, 0)},
	}
	app := workFirstApp(t, now, []model.Course{course}, assignments)
	app.selected["S1"] = true
	if cmd := app.applyTriage(model.Tracked); cmd != nil {
		t.Fatal("hidden selection should arm, not write")
	}
	if !app.triageArmed || app.armedTriage != model.Tracked || app.armedTriageHidden != 1 {
		t.Fatalf("armed state = %v/%v/%d", app.triageArmed, app.armedTriage, app.armedTriageHidden)
	}
	if got := stripANSI(app.selectionLabel()); !strings.Contains(got, "1 selected, 1 hidden") {
		t.Fatalf("selection label = %q", got)
	}
	app.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if app.triageArmed || app.byID("S1").Triage != model.Undecided {
		t.Fatalf("esc should cancel without applying: armed=%v triage=%v", app.triageArmed, app.byID("S1").Triage)
	}

	app.selected["S1"] = true
	if cmd := app.applyTriage(model.Tracked); cmd != nil {
		t.Fatal("hidden selection should arm again")
	}
	if cmd := app.applyTriage(model.Tracked); cmd == nil {
		t.Fatal("the same triage key should apply")
	}
	if app.byID("S1").Triage != model.Tracked {
		t.Fatalf("hidden confirm did not apply: %v", app.byID("S1").Triage)
	}

	app.selected["S1"] = true
	if cmd := app.applyTriage(model.Missed); cmd != nil {
		t.Fatal("a different key should re-arm")
	}
	if !app.triageArmed || app.armedTriage != model.Missed {
		t.Fatalf("different key did not re-arm: %v", app.armedTriage)
	}
	app.applyTriage(model.Missed)
	if app.byID("S1").Triage != model.Missed {
		t.Fatalf("re-armed decision = %v", app.byID("S1").Triage)
	}

	app.selected["A1"] = true
	app.applyTriage(model.Missed)
	if app.triageArmed || app.byID("A1").Triage != model.Missed {
		t.Fatalf("visible decision armed unexpectedly: armed=%v triage=%v", app.triageArmed, app.byID("A1").Triage)
	}
}

func TestUndoBulkTriage(t *testing.T) {
	now := workFirstNow()
	course := model.Course{ID: 1, Short: "c1", Name: "Course One"}
	assignments := []model.Assignment{
		{ID: "S1", CourseID: 1, Name: "Stale One", HasDue: true, Due: dueOn(now, -8)},
		{ID: "S2", CourseID: 1, Name: "Stale Two", HasDue: true, Due: dueOn(now, -9)},
		{ID: "A1", CourseID: 1, Name: "Today", HasDue: true, Due: dueOn(now, 0)},
	}
	app := workFirstApp(t, now, []model.Course{course}, assignments)
	nodes := app.taskNodes()
	app.taskCursor = findBucketNode(nodes, 1, "stale overdue")
	app.handleKey(tea.KeyMsg{Type: tea.KeySpace})
	app.applyTriage(model.Tracked)
	app.applyTriage(model.Tracked)
	if app.byID("S1").Triage != model.Tracked {
		t.Fatalf("bulk triage did not apply: %v", app.byID("S1").Triage)
	}
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if app.byID("S1").Triage != model.Undecided || app.byID("S2").Triage != model.Undecided {
		t.Fatalf("u did not undo the bulk action: S1=%v S2=%v", app.byID("S1").Triage, app.byID("S2").Triage)
	}
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if app.toast != "nothing to undo" {
		t.Fatalf("second undo toast = %q", app.toast)
	}

	app.selected["A1"] = true
	app.applyTriage(model.Missed)
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if app.byID("A1").Triage != model.Undecided {
		t.Fatalf("single undo = %v", app.byID("A1").Triage)
	}
}

func TestShowAllToggle(t *testing.T) {
	app := treeSampleApp(t, workFirstNow())
	if title, _, _, _ := app.tasksContent(120, 30); title != "tasks › all · summary view" {
		t.Fatalf("summary title = %q", title)
	}
	before := visibleIDs(app)
	nodes := app.taskNodes()
	if findNodeByKey(nodes, nodeActNow, actNowKey) < 0 {
		t.Fatal("summary view is missing Act now")
	}
	if findBucketNode(nodes, 1, "stale overdue") < 0 {
		t.Fatal("summary view is missing the stale bucket")
	}
	if findNodeByKey(nodes, nodeCoursePlaceholder, courseKey(2)) < 0 {
		t.Fatal("summary view is missing the all-in-Act-now placeholder")
	}

	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	if title, _, _, _ := app.tasksContent(120, 30); title != "tasks › all · full view" {
		t.Fatalf("full title = %q", title)
	}
	nodes = app.taskNodes()
	if findNodeByKey(nodes, nodeActNow, actNowKey) >= 0 {
		t.Fatal("full view still has Act now")
	}
	if findBucketNode(nodes, 1, "stale overdue") >= 0 {
		t.Fatal("full view still has a stale bucket")
	}
	if findNodeByKey(nodes, nodeCoursePlaceholder, courseKey(2)) >= 0 {
		t.Fatal("full view still has the all-in-Act-now placeholder")
	}
	if findTaskByID(nodes, "A1") < 0 || findTaskByID(nodes, "B1") < 0 {
		t.Fatal("full view did not return stale/Act-now tasks to courses")
	}
	after := visibleIDs(app)
	if len(before) != len(after) {
		t.Fatalf("visible count changed: %d -> %d", len(before), len(after))
	}
	for id := range before {
		if !after[id] {
			t.Fatalf("visible ID %s disappeared", id)
		}
	}

	app.search = "course"
	if title, _, _, _ := app.tasksContent(120, 30); !strings.Contains(title, "full view") {
		t.Fatalf("search title lost the mode: %q", title)
	}
	app.search = ""
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	if title, _, _, _ := app.tasksContent(120, 30); title != "tasks › all · summary view" {
		t.Fatalf("summary title after toggle back = %q", title)
	}
	if findNodeByKey(app.taskNodes(), nodeActNow, actNowKey) < 0 {
		t.Fatal("toggle back did not restore Act now")
	}
}

func TestCursorStabilityAcrossCollapse(t *testing.T) {
	app := treeSampleApp(t, workFirstNow())
	nodes := app.taskNodes()
	a2 := findTaskByID(nodes, "A2")
	app.taskCursor = a2
	app.toggleShowAll()
	nodes = app.taskNodes()
	if app.taskCursor < 0 || app.taskCursor >= len(nodes) || nodes[app.taskCursor].kind != nodeTask || nodes[app.taskCursor].task.ID != "A2" {
		t.Fatalf("show-all lost the focused task: %#v", nodes[app.taskCursor])
	}
	app.toggleShowAll()
	nodes = app.taskNodes()
	if nodes[app.taskCursor].task.ID != "A2" {
		t.Fatalf("toggle back lost the focused task: %#v", nodes[app.taskCursor])
	}

	nodes = app.taskNodes()
	a1 := findTaskByID(nodes, "A1")
	if a1 < 0 {
		t.Fatal("sample tree has no stale A1")
	}
	app.taskCursor = a1
	app.captureCursors()
	app.expanded[bucketKey(1, "stale overdue")] = false
	app.restoreCursors()
	nodes = app.taskNodes()
	if app.taskCursor < 0 || app.taskCursor >= len(nodes) || nodes[app.taskCursor].kind != nodeBucket || nodes[app.taskCursor].bucket != "stale overdue" {
		t.Fatalf("collapsed stale child did not land on its header: %#v", nodes[app.taskCursor])
	}

	placeholder := findNodeByKey(nodes, nodeCoursePlaceholder, courseKey(2))
	if placeholder < 0 {
		t.Fatal("sample tree has no placeholder")
	}
	app.taskCursor = placeholder
	app.clampCursor()
	nodes = app.taskNodes()
	if nodes[app.taskCursor].kind == nodeCoursePlaceholder {
		t.Fatal("cursor landed on the non-selectable placeholder")
	}
}

func TestSelectionAcrossSectionsAndRefresh(t *testing.T) {
	app := treeSampleApp(t, workFirstNow())
	app.selected["B1"] = true
	app.toggleShowAll()
	if !app.selected["B1"] || findTaskByID(app.taskNodes(), "B1") < 0 {
		t.Fatal("selection did not follow the task into the course tree")
	}
	app.toggleShowAll()
	app.Update(app.Init()())
	if !app.selected["B1"] || findTaskByID(app.taskNodes(), "B1") < 0 {
		t.Fatal("selection did not survive refresh in Act now")
	}
}

func TestOverdueCountUnaffectedByGrouping(t *testing.T) {
	app := treeSampleApp(t, workFirstNow())
	overdue := app.overdueCount()
	stale := app.staleOverdueCount()
	if overdue != 2 || stale != 1 {
		t.Fatalf("sample counts = %d overdue, %d stale", overdue, stale)
	}
	summary := stripANSI(app.tabSummary())
	if !strings.Contains(summary, "2 overdue (1 stale)") {
		t.Fatalf("top bar summary = %q", summary)
	}
	app.toggleShowAll()
	if app.overdueCount() != overdue || app.staleOverdueCount() != stale {
		t.Fatal("show-all changed the global overdue counts")
	}
	if got := stripANSI(app.tabSummary()); !strings.Contains(got, "2 overdue (1 stale)") {
		t.Fatalf("full-view top bar summary = %q", got)
	}
	app.expanded[courseKey(1)] = false
	app.clampCursor()
	if app.overdueCount() != overdue || app.staleOverdueCount() != stale {
		t.Fatal("collapsing a course changed the global overdue counts")
	}
}

func TestFooterShowAllHint(t *testing.T) {
	app := treeSampleApp(t, workFirstNow())
	app.expanded[bucketKey(1, "stale overdue")] = false
	nodes := app.taskNodes()
	app.taskCursor = findNodeByKey(nodes, nodeCourse, courseKey(1))
	full := stripANSI(renderHintItems(app.footerHintItems(), 200))
	for _, want := range []string{"f filter", "A show all"} {
		if !strings.Contains(full, want) {
			t.Fatalf("normal footer %q is missing %q", full, want)
		}
	}
	if strings.Contains(full, "enter detail") {
		t.Fatalf("Tasks footer should drop enter: %q", full)
	}

	app.taskCursor = findBucketNode(nodes, 1, "stale overdue")
	context := stripANSI(renderHintItems(app.footerHintItems(), 200))
	for _, want := range []string{"enter expand", "space select", "t/m/i decide 1", "A show all"} {
		if !strings.Contains(context, want) {
			t.Fatalf("stale header footer %q is missing %q", context, want)
		}
	}

	app.showAll = true
	app.taskCursor = findNodeByKey(app.taskNodes(), nodeCourse, courseKey(1))
	full = stripANSI(renderHintItems(app.footerHintItems(), 200))
	if !strings.Contains(full, "A summary") || strings.Contains(full, "A show all") || !strings.Contains(full, "f filter") {
		t.Fatalf("full-view footer = %q", full)
	}

	help := stripANSI(strings.Join(app.helpContent(200), "\n"))
	for _, want := range []string{"enter", "filter", "show all"} {
		if !strings.Contains(help, want) {
			t.Fatalf("help is missing %q", want)
		}
	}
	if app.keys.All.Help().Key == app.keys.ShowAll.Help().Key {
		t.Fatalf("show-all key %q clashes with select-all", app.keys.ShowAll.Help().Key)
	}
}
