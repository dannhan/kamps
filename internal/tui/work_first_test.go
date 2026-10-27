package tui

import (
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

func TestHiddenSelectionConfirm(t *testing.T) {
	now := workFirstNow()
	course := model.Course{ID: 1, Short: "c1", Name: "Course One"}
	assignments := []model.Assignment{
		{ID: "S1", CourseID: 1, Name: "Stale", HasDue: true, Due: dueOn(now, -8)},
		{ID: "A1", CourseID: 1, Name: "Today", HasDue: true, Due: dueOn(now, 0)},
	}
	app := workFirstApp(t, now, []model.Course{course}, assignments)
	app.search = "Today"
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

	app.search = "Today"
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
	app.selected["S1"] = true
	app.selected["S2"] = true
	app.expanded[taskOverdueKey] = true
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
