package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/lipgloss"
	"github.com/ruyam/kamps/internal/model"
)

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
		if n.kind == nodeFile && !n.mat.Downloaded && n.mat.Kind == "file" {
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

	var file, course, assignment *node
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
		case nodeMaterialAssignment:
			if assignment == nil {
				assignment = n
			}
		}
	}
	if file == nil || course == nil {
		t.Fatalf("demo materials missing rows: file=%v course=%v", file != nil, course != nil)
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
