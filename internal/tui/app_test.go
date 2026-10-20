package tui

import (
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ruyam/kamps/internal/model"
)

func TestEscapeClearsFilterBeforeSelections(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks
	app.filter = "undecided"
	app.tabFilters[TabTasks] = "undecided"
	app.taskCursor = 3
	selectedID := app.assignments[0].ID
	app.selected[selectedID] = true

	if _, _, hints, _ := app.tabContent(96, 20); !slices.Contains(hints, "esc clear") {
		t.Fatalf("an active filter should advertise esc clear, got %v", hints)
	}

	app.handleKey(tea.KeyMsg{Type: tea.KeyEsc})

	if app.filter != "all" || app.tabFilters[TabTasks] != "all" {
		t.Fatalf("esc should clear the filter, got filter=%q stored=%q", app.filter, app.tabFilters[TabTasks])
	}
	if !app.selected[selectedID] {
		t.Fatal("clearing a filter should not drop the selection")
	}
	if app.taskCursor != 0 {
		t.Fatalf("clearing a filter should reset the cursor, got %d", app.taskCursor)
	}

	app.handleKey(tea.KeyMsg{Type: tea.KeyEsc})

	if len(app.selected) != 0 {
		t.Fatalf("a second esc should clear the selection, got %v", app.selected)
	}
}

func TestShellUsesRulesInsteadOfABox(t *testing.T) {
	app := demoApp(t)
	app.width, app.height = 100, 30
	lines := strings.Split(stripANSI(app.View()), "\n")
	if strings.ContainsAny(strings.Join(lines, ""), "╭╮") {
		t.Fatal("shell still draws rounded pane corners")
	}
	for i, line := range lines {
		if strings.HasSuffix(line, "│") {
			t.Fatalf("line %d still has a vertical pane border: %q", i, line)
		}
	}
	want := strings.Repeat("─", 100)
	rules := 0
	for _, line := range lines {
		if line == want {
			rules++
		}
	}
	if rules != 2 {
		t.Fatalf("want one rule above and one below the body, got %d", rules)
	}
	if lines[2] != want || lines[len(lines)-2] != want {
		t.Fatalf("rules are not fenced around the body: %q / %q", lines[2], lines[len(lines)-2])
	}
}

func TestDetailPaneFollowsWidthAndEnter(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks

	app.width = 100
	if !app.detailVisible() {
		t.Fatal("detail pane should show at 100 columns")
	}
	app.width = 80
	if app.detailVisible() {
		t.Fatal("detail pane should auto-hide at 80 columns")
	}

	app.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	if !app.detailVisible() {
		t.Fatal("enter should pin the detail pane open at 80 columns")
	}
	if app.tab != TabTasks {
		t.Fatal("enter must not jump to the commit tab")
	}
	app.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if app.detailVisible() {
		t.Fatal("a second enter should close the detail pane")
	}

	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if app.tab != TabCommit {
		t.Fatal("c should still open the commit tab")
	}
}

func TestDetailPaneRendersTheSelectedTask(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks
	nodes := app.taskNodes()
	cursor := -1
	for i, n := range nodes {
		if n.kind == nodeTask {
			cursor = i
			break
		}
	}
	if cursor < 0 {
		t.Fatal("demo data has no task rows")
	}
	task := nodes[cursor].task
	lines := stripANSI(strings.Join(app.detailLines(nodes[cursor], 40), "\n"))
	for _, want := range []string{task.ID, task.Name, "due", "state", "course"} {
		if !strings.Contains(lines, want) {
			t.Fatalf("detail pane %q is missing %q", lines, want)
		}
	}
}

func TestEnterTogglesDetailOnMaterials(t *testing.T) {
	app := demoApp(t)
	app.tab = TabMaterials
	app.width = 80

	if app.detailVisible() {
		t.Fatal("materials detail should auto-hide at 80 columns")
	}
	app.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !app.detailVisible() {
		t.Fatal("enter should open the materials detail pane")
	}
	if app.toast != "" {
		t.Fatalf("enter should not download on Materials, toast = %q", app.toast)
	}
	if view := stripANSI(app.View()); !strings.Contains(view, "local") {
		t.Fatal("materials view has no detail column after enter")
	}

	nodes := app.matNodes()
	for i, n := range nodes {
		if n.kind == nodeFile && !n.mat.Downloaded {
			app.matCursor = i
			break
		}
	}
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if !strings.Contains(app.toast, "downloaded") {
		t.Fatalf("d should still download on Materials, toast = %q", app.toast)
	}
}

func TestMaterialDetailLines(t *testing.T) {
	app := demoApp(t)
	app.tab = TabMaterials
	nodes := app.matNodes()

	var file, course, bucket, assignment *node
	for i := range nodes {
		n := &nodes[i]
		switch n.kind {
		case nodeFile:
			if file == nil && n.mat.Size > 0 {
				file = n
			}
		case nodeCourse:
			if course == nil {
				course = n
			}
		case nodeMaterialBucket:
			if bucket == nil {
				bucket = n
			}
		case nodeMaterialAssignment:
			if assignment == nil {
				assignment = n
			}
		}
	}
	if file == nil || course == nil || bucket == nil {
		t.Fatalf("demo materials missing rows: file=%v course=%v bucket=%v", file != nil, course != nil, bucket != nil)
	}

	fileLines := stripANSI(strings.Join(app.materialDetailLines(*file, 40), "\n"))
	for _, want := range []string{file.mat.ID, "course", "section", "size", "state"} {
		if !strings.Contains(fileLines, want) {
			t.Fatalf("material detail %q is missing %q", fileLines, want)
		}
	}

	courseLines := stripANSI(strings.Join(app.materialDetailLines(*course, 40), "\n"))
	for _, want := range []string{"files", "local", "size"} {
		if !strings.Contains(courseLines, want) {
			t.Fatalf("course detail %q is missing %q", courseLines, want)
		}
	}
	if strings.Contains(courseLines, "states") || strings.Contains(courseLines, "tasks") {
		t.Fatalf("materials course detail leaked task fields: %q", courseLines)
	}

	bucketLines := stripANSI(strings.Join(app.materialDetailLines(*bucket, 40), "\n"))
	if !strings.Contains(bucketLines, bucket.bucket) || !strings.Contains(bucketLines, "files") {
		t.Fatalf("bucket detail = %q", bucketLines)
	}

	if assignment != nil {
		assignmentLines := stripANSI(strings.Join(app.materialDetailLines(*assignment, 40), "\n"))
		for _, want := range []string{assignment.task.ID, "course", "state", "due"} {
			if !strings.Contains(assignmentLines, want) {
				t.Fatalf("linked assignment detail %q is missing %q", assignmentLines, want)
			}
		}
	}
}

func TestDetailPaneClipsLongTextWithEllipsis(t *testing.T) {
	app := demoApp(t)
	app.taskCursor = 0
	long := strings.Repeat("a very long task description ", 40)
	nodes := []node{{kind: nodeTask, task: model.Assignment{ID: "K999999", Name: long, Triage: model.Undecided}}}
	rows := app.detailRows(nodes, 0, 34, 8)
	if len(rows) != 8 {
		t.Fatalf("detail rows = %d, want 8", len(rows))
	}
	for i, row := range rows {
		if got := lipgloss.Width(row); got != 34 {
			t.Fatalf("detail row %d width %d, want 34", i, got)
		}
	}
	if !strings.Contains(stripANSI(rows[7]), "…") {
		t.Fatalf("clipped detail pane has no ellipsis: %q", stripANSI(rows[7]))
	}
}

func TestCommitPreviewAndDrop(t *testing.T) {
	app := demoApp(t)
	app.tab = TabCommit
	app.pending = map[string]model.Triage{"K901000": model.Tracked, "K901001": model.Missed}
	app.byID("K901000").Triage = model.Tracked

	want := app.taskLine(*app.byID("K901000")) + "\n" + app.taskLine(*app.byID("K901001"))
	if got := app.commitPreviewText(); got != want {
		t.Fatalf("preview = %q, want %q", got, want)
	}

	// Items sort by action, so the tracked item is first.
	app.commitCursor = 0
	app.dropCommitItem()

	if _, ok := app.pending["K901000"]; ok {
		t.Fatal("drop should remove the focused decision from the queue")
	}
	if len(app.pending) != 1 {
		t.Fatalf("queue has %d items after drop, want 1", len(app.pending))
	}
	if got := app.byID("K901000"); got == nil || got.Triage != model.Tracked {
		t.Fatal("drop should not revert the decision itself")
	}
}

func TestCommitEmptyQueueExplainsAndSkipsCopy(t *testing.T) {
	app := demoApp(t)
	app.tab = TabCommit
	app.pending = map[string]model.Triage{}

	if got := app.commitPreviewText(); got != "" {
		t.Fatalf("empty queue preview = %q", got)
	}
	if cmd := app.copyCommitPreview(); cmd != nil {
		t.Fatal("copying an empty queue should not emit a command")
	}
	if !strings.Contains(app.toast, "nothing to copy") {
		t.Fatalf("empty copy toast = %q", app.toast)
	}
	body := stripANSI(strings.Join(app.commitBody(80, 10), "\n"))
	if !strings.Contains(body, "no decisions queued") || strings.Contains(body, "mark items in Tasks first") {
		t.Fatalf("empty commit body = %q", body)
	}
}

func TestCommitRowsClipWithEllipsis(t *testing.T) {
	app := demoApp(t)
	app.tab = TabCommit
	id := app.assignments[0].ID
	app.byID(id).Name = strings.Repeat("a very long assignment name ", 10)
	app.pending = map[string]model.Triage{id: model.Tracked}

	body := stripANSI(strings.Join(app.commitBody(60, 20), "\n"))
	if !strings.Contains(body, "…") {
		t.Fatalf("clipped commit row has no ellipsis: %q", body)
	}
}

func TestSplitPaneKeepsEveryLineAtWidth(t *testing.T) {
	for _, tab := range []Tab{TabTasks, TabMaterials} {
		for _, width := range []int{100, 80} {
			app := demoApp(t)
			app.width, app.height = width, 24
			app.tab = tab
			app.detailPane = 1 // force the pane on, even at 80
			for i, line := range strings.Split(stripANSI(app.View()), "\n") {
				if got := lipgloss.Width(line); got != width {
					t.Fatalf("%s %d-column split line %d width %d: %q", tab, width, i, got, line)
				}
			}
		}
	}
}

func TestSlashOpensSearchAndNarrowsTasks(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks
	app.width, app.height = 100, 24

	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !app.searching {
		t.Fatal("slash should open search")
	}
	for _, r := range "quiz" {
		app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if app.search != "quiz" {
		t.Fatalf("search = %q, want quiz", app.search)
	}

	visible := app.visible()
	if len(visible) == 0 || len(visible) == len(app.assignments) {
		t.Fatalf("search matched %d of %d assignments", len(visible), len(app.assignments))
	}
	for _, assignment := range visible {
		if !app.matchesTaskSearch(assignment) {
			t.Fatalf("search kept a non-matching task: %q", assignment.Name)
		}
	}

	view := stripANSI(app.View())
	if !strings.Contains(view, "search · quiz") || !strings.Contains(view, "matches") {
		t.Fatalf("search title missing from view: %q", view)
	}
	if !strings.Contains(view, "enter done") {
		t.Fatalf("search footer missing: %q", view)
	}
}

func TestSearchKeepsCompoundInputAndBackspace(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks

	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	app.handleKey(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if app.search != "q u" {
		t.Fatalf("space should be typed into the query, got %q", app.search)
	}

	app.handleKey(tea.KeyMsg{Type: tea.KeyBackspace})
	if app.search != "q " {
		t.Fatalf("backspace = %q, want %q", app.search, "q ")
	}
	app.handleKey(tea.KeyMsg{Type: tea.KeyCtrlU})
	if app.search != "" {
		t.Fatalf("ctrl+u should clear the query, got %q", app.search)
	}
}

func TestSearchCtrlWDeletesWords(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks
	app.searching = true
	app.search = "quiz basis data"

	for i, want := range []string{"quiz basis ", "quiz ", ""} {
		app.handleKey(tea.KeyMsg{Type: tea.KeyCtrlW})
		if app.search != want {
			t.Fatalf("ctrl+w #%d = %q, want %q", i+1, app.search, want)
		}
	}

	app.search = "quiz basis"
	app.handleKey(tea.KeyMsg{Type: tea.KeyBackspace, Alt: true})
	if app.search != "quiz " {
		t.Fatalf("alt+backspace = %q, want %q", app.search, "quiz ")
	}
}

func TestSearchEnterKeepsQueryAndEscClearsItFirst(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks
	app.search = "quiz"
	app.searching = true

	app.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if app.searching || app.search != "quiz" {
		t.Fatalf("enter should keep the query and leave edit mode: %q searching=%v", app.search, app.searching)
	}
	if len(app.visible()) == 0 {
		t.Fatal("query should still narrow the tree after enter")
	}

	app.filter = "undecided"
	app.tabFilters[TabTasks] = "undecided"
	app.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if app.search != "" || app.filter != "undecided" {
		t.Fatalf("esc should clear search first: search=%q filter=%q", app.search, app.filter)
	}
	app.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if app.filter != "all" {
		t.Fatalf("second esc should clear the filter, got %q", app.filter)
	}
}

func TestSearchNarrowsMaterialsAndComposesWithFilter(t *testing.T) {
	app := demoApp(t)
	app.tab = TabMaterials
	app.search = "modul"

	files := 0
	for _, n := range app.matNodes() {
		if n.kind != nodeFile {
			continue
		}
		files++
		if !app.matchesMaterialSearch(n.mat) {
			t.Fatalf("search kept a non-matching material: %q", n.mat.Name)
		}
	}
	if files == 0 {
		t.Fatal("materials search found no files")
	}

	app.filter = "new"
	for _, n := range app.matNodes() {
		if n.kind == nodeFile && n.mat.Downloaded {
			t.Fatalf("new filter let a downloaded file through: %q", n.mat.Name)
		}
	}
}

func TestSwitchTabClearsSearch(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks
	app.search = "quiz"
	app.searching = true

	app.switchTab(TabMaterials)
	if app.search != "" || app.searching {
		t.Fatalf("tab switch should drop the search: %q searching=%v", app.search, app.searching)
	}
}

func TestFilterCyclesOnFAndNotSlash(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks

	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	if app.filter != "undecided" {
		t.Fatalf("f should advance the filter, got %q", app.filter)
	}

	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if app.filter != "undecided" {
		t.Fatalf("slash should be reserved for search, got filter %q", app.filter)
	}
}

func TestScrollHintReportsHiddenRows(t *testing.T) {
	cases := []struct {
		total, cursor, height int
		want                  string
	}{
		{20, 10, 5, "▲ 8 ▼ 7"},
		{10, 0, 3, "▼ 7"},
		{10, 9, 3, "▲ 7"},
		{5, 4, 5, ""},
		{0, 0, 5, ""},
	}
	for _, tc := range cases {
		if got := stripANSI(scrollHint(tc.total, tc.cursor, tc.height)); got != tc.want {
			t.Errorf("scrollHint(%d, %d, %d) = %q, want %q", tc.total, tc.cursor, tc.height, got, tc.want)
		}
	}
}

func TestPanesAdvertiseHiddenRows(t *testing.T) {
	app := demoApp(t)

	app.tab = TabTasks
	if _, right, _, _ := app.tabContent(96, 1); !strings.Contains(stripANSI(right), "▼") {
		t.Fatalf("a clipped task tree should show a down hint, got %q", stripANSI(right))
	}
	if _, right, _, _ := app.tabContent(96, 500); strings.ContainsAny(stripANSI(right), "▲▼") {
		t.Fatalf("a tree that fits should not show a scroll hint, got %q", stripANSI(right))
	}

	app.tab = TabMaterials
	if _, right, _, _ := app.tabContent(96, 1); !strings.Contains(stripANSI(right), "▼") {
		t.Fatalf("a clipped material tree should show a down hint, got %q", stripANSI(right))
	}
}

func TestEscapeClearsMaterialFilter(t *testing.T) {
	app := demoApp(t)
	app.tab = TabMaterials
	app.filter = "new"
	app.tabFilters[TabMaterials] = "new"
	app.matCursor = 2

	app.handleKey(tea.KeyMsg{Type: tea.KeyEsc})

	if app.filter != "all" || app.tabFilters[TabMaterials] != "all" {
		t.Fatalf("esc should clear the material filter, got filter=%q stored=%q", app.filter, app.tabFilters[TabMaterials])
	}
	if app.matCursor != 0 {
		t.Fatalf("clearing a filter should reset the cursor, got %d", app.matCursor)
	}
}
