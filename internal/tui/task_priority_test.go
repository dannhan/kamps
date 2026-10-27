package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ruyam/kamps/internal/model"
)

func priorityApp(t *testing.T) *App {
	now := workFirstNow()
	assignments := []model.Assignment{
		{ID: "old", CourseID: 1, Name: "Old overdue", HasDue: true, Due: dueOn(now, -30)},
		{ID: "recent", CourseID: 1, Name: "Recent overdue", HasDue: true, Due: dueOn(now, -1)},
		{ID: "elapsed", CourseID: 1, Name: "Earlier today", HasDue: true, Due: now.Add(-time.Hour)},
		{ID: "later", CourseID: 1, Name: "Later work", HasDue: true, Due: dueOn(now, 8)},
		{ID: "undated", CourseID: 1, Name: "No date"},
		{ID: "tracked", CourseID: 1, Name: "Already handled", Triage: model.Tracked},
		{ID: "missed", CourseID: 1, Name: "Marked missed", Triage: model.Missed},
		{ID: "ignored", CourseID: 1, Name: "Ignored template", Triage: model.Ignored},
	}
	for i := 0; i < 8; i++ {
		assignments = append(assignments, model.Assignment{ID: fmt.Sprintf("current-%d", i), CourseID: 1, Name: fmt.Sprintf("Current task %d", i), HasDue: true, Due: dueOn(now, i)})
	}
	app := workFirstApp(t, now, []model.Course{{ID: 1, Name: "Numerical Methods"}, {ID: 2, Name: "Empty course"}}, assignments)
	app.width, app.height = 80, 24
	app.taskCursor = 1
	return app
}

func taskAppearance(app *App, id, parent string) int {
	for i, n := range app.taskNodes() {
		if n.kind == nodeTask && n.task.ID == id && len(n.parents) > 0 && n.parents[0] == parent {
			return i
		}
	}
	return -1
}

func TestPriorityOpeningAndExpansion(t *testing.T) {
	app := priorityApp(t)
	nodes := app.taskNodes()
	visible := taskNodeIDs(nodes)
	if len(visible) != 5 || visible[0] != "current-0" || visible[4] != "current-4" {
		t.Fatalf("shortlist: %v", visible)
	}
	view := stripANSI(app.View())
	for _, want := range []string{"Needs attention", "Numerical Methods", "Overdue · 3", "Show all 8"} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %s:\n%s", want, view)
		}
	}
	for _, n := range nodes {
		if n.kind == nodeTaskMore {
			app.taskCursor = findNodeByKey(nodes, nodeTaskMore, n.key)
			press(app, "enter")
			break
		}
	}
	if len(taskNodeIDs(app.taskNodes())) != 8 {
		t.Fatal("show all did not expand shortlist")
	}
	if app.filter != "all" {
		t.Fatal("expansion changed filter")
	}
	app.expanded[taskOverdueKey] = true
	var overdue []string
	for _, n := range app.taskNodes() {
		if n.kind == nodeTask && n.parents[0] == taskOverdueKey {
			overdue = append(overdue, n.task.ID)
		}
	}
	if strings.Join(overdue, ",") != "elapsed,recent,old" {
		t.Fatalf("overdue order: %v", overdue)
	}
	app.filter = "upcoming"
	for _, a := range app.visible() {
		if a.ID == "elapsed" || a.ID == "later" || a.Triage != model.Undecided {
			t.Fatalf("incorrect upcoming task: %s", a.ID)
		}
	}
}

func TestPriorityCoursesAndDuplicateMarks(t *testing.T) {
	app := priorityApp(t)
	app.expanded[courseKey(1)] = true
	ids := taskNodeIDs(app.taskNodes())
	if len(ids) != 18 {
		t.Fatalf("expected five shortlist plus thirteen undecided course tasks: %v", ids)
	}
	first := taskAppearance(app, "current-0", "tasks:attention")
	second := taskAppearance(app, "current-0", courseKey(1))
	if first < 0 || second < 0 {
		t.Fatal("missing duplicate appearance")
	}
	nodes := app.taskNodes()
	if nodes[first].key == nodes[second].key {
		t.Fatal("appearance keys collide")
	}
	app.taskCursor = first
	press(app, " ")
	if app.selectedCount() != 1 {
		t.Fatal("duplicate counted twice")
	}
	for _, index := range []int{first, second} {
		if !strings.Contains(stripANSI(strings.Join(app.taskBlock(nodes[index], index, 80), "\n")), "▪") {
			t.Fatal("mark not synchronized")
		}
	}
	app.taskCursor = second
	press(app, " ")
	if app.selectedCount() != 0 {
		t.Fatal("unmark not synchronized")
	}
	app.taskCursor = findNodeByKey(app.taskNodes(), nodeCourse, courseKey(1))
	press(app, "t")
	if app.queue.Len() != 0 {
		t.Fatal("unmarked heading implicitly triaged children")
	}
	press(app, " ")
	press(app, "i")
	if !app.triageArmed {
		t.Fatal("hidden marked tasks did not require confirmation")
	}
	press(app, "i")
	if app.queue.Len() != len(app.assignments) {
		t.Fatal("batch queue lost or duplicated assignments")
	}
}

func TestPriorityDecisionFocusAndRefresh(t *testing.T) {
	app := priorityApp(t)
	app.taskCursor = taskAppearance(app, "current-0", "tasks:attention")
	press(app, "t")
	if app.cursorAssignmentID() != "current-1" {
		t.Fatalf("next focus: %s", app.cursorAssignmentID())
	}
	app.taskCursor = taskAppearance(app, "current-4", "tasks:attention")
	press(app, "i")
	if app.cursorAssignmentID() != "current-5" {
		t.Fatalf("next in shortlist: %s", app.cursorAssignmentID())
	}
	app.selected["current-2"] = true
	key := app.taskNodes()[app.taskCursor].key
	app.captureCursors()
	app.Update(app.Init()())
	if app.taskNodes()[app.taskCursor].key != key || !app.selected["current-2"] {
		t.Fatal("refresh lost identity/mark")
	}
	app.switchTab(TabMaterials)
	app.filter = "unknown"
	app.search = "no materials"
	app.captureCursors()
	app.Update(app.Init()())
	app.switchTab(TabTasks)
	if app.taskNodes()[app.taskCursor].key != key {
		t.Fatal("refresh from Materials lost task focus")
	}
	for _, size := range [][2]int{{80, 24}, {120, 35}, {160, 45}} {
		app.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		if app.taskNodes()[app.taskCursor].key != key {
			t.Fatal("resize lost focus")
		}
	}
}

func TestPriorityLastTaskFallsBackToPrevious(t *testing.T) {
	app := priorityApp(t)
	app.expanded[taskShortlistKey] = true
	app.taskCursor = taskAppearance(app, "current-7", "tasks:attention")
	press(app, "m")
	if app.cursorAssignmentID() != "current-6" {
		t.Fatalf("previous focus: %s", app.cursorAssignmentID())
	}
}

func TestPriorityWholeBlockScrollingAndDetails(t *testing.T) {
	app := priorityApp(t)
	app.expanded[courseKey(1)] = true
	app.expanded[taskShortlistKey] = true
	for _, size := range [][2]int{{80, 24}, {120, 35}, {160, 45}} {
		app.width, app.height = size[0], size[1]
		for index, n := range app.taskNodes() {
			if n.kind != nodeTask {
				continue
			}
			app.taskCursor = index
			rows, start, end := app.taskListWindow(app.taskNodes(), size[0], size[1]-5)
			if index < start || index >= end {
				t.Fatal("focused block not in window")
			}
			for _, line := range rows {
				if lipgloss.Width(line) != size[0] {
					t.Fatalf("bad width %d", lipgloss.Width(line))
				}
			}
			view := stripANSI(strings.Join(rows, "\n"))
			if !strings.Contains(view, n.task.Name) || !strings.Contains(view, readableDeadline(n.task, app)) {
				t.Fatal("task separated from deadline")
			}
		}
	}
	app.taskCursor = taskAppearance(app, "current-0", courseKey(1))
	press(app, "enter")
	detail := stripANSI(strings.Join(app.detailRows(app.taskNodes(), app.taskCursor, 80, 20), "\n"))
	if !strings.HasPrefix(strings.TrimSpace(detail), "Current task 0") || strings.Contains(detail, "current-0") {
		t.Fatalf("details expose ID or wrong hierarchy:\n%s", detail)
	}
	app.toast = "decision saved"
	for _, width := range []int{80, 120, 160} {
		footer := stripANSI(strings.Join(app.footerLines(width, sizeStandard), ""))
		for _, hint := range []string{"t Track", "i Ignore", "m Missed", "space Mark"} {
			if !strings.Contains(footer, hint) {
				t.Fatalf("missing %s at %d", hint, width)
			}
		}
	}
}

func TestPriorityRenderedScenarios(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {120, 35}, {160, 45}} {
		for _, scenario := range []string{"opening", "expanded", "overdue-only", "undated-only", "handled-only", "empty-course", "long-names"} {
			app := priorityApp(t)
			app.width, app.height = size[0], size[1]
			snapshot := app.service.Snapshot()
			switch scenario {
			case "expanded":
				app.expanded[courseKey(1)] = true
				app.expanded[taskOverdueKey] = true
				app.expanded[bucketKey(1, "Tracked")] = true
				app.expanded[bucketKey(1, "Closed")] = true
			case "overdue-only", "undated-only", "handled-only":
				items := []model.Assignment{}
				for _, a := range snapshot.Assignments {
					if scenario == "overdue-only" && a.Overdue(app.currentTime()) || scenario == "undated-only" && !a.HasDue && a.Triage == model.Undecided || scenario == "handled-only" && a.Triage != model.Undecided {
						items = append(items, a)
					}
				}
				app = workFirstApp(t, app.currentTime(), snapshot.Courses, items)
				app.width, app.height = size[0], size[1]
			case "empty-course":
				app.courseScoped = true
				app.courseID = 2
				app.expanded[courseKey(2)] = true
			case "long-names":
				for i := range snapshot.Assignments {
					snapshot.Assignments[i].Name = strings.Repeat("课程 👩🏽‍💻 project description ", 12)
				}
				for i := range snapshot.Courses {
					if snapshot.Courses[i].ID == 1 {
						snapshot.Courses[i].Name = "Numerical Methods and Computational Science with Applied Mathematics 计算科学"
					}
				}
				app.Update(loadedMsg{snapshot})
			}
			app.clampCursor()
			view := stripANSI(app.View())
			lines := strings.Split(view, "\n")
			if len(lines) != size[1] {
				t.Fatalf("%v %s height %d", size, scenario, len(lines))
			}
			for _, line := range lines {
				if lipgloss.Width(line) != size[0] {
					t.Fatalf("%v %s wrong width %q", size, scenario, line)
				}
			}
			if strings.HasSuffix(scenario, "-only") && !strings.Contains(view, "No deadlines in the next 7 days") {
				t.Fatalf("missing empty priority state: %s", scenario)
			}
			if dir := os.Getenv("KAMPS_RENDER_DIR"); dir != "" {
				if err := os.MkdirAll(dir, 0700); err != nil {
					t.Fatal(err)
				}
				name := fmt.Sprintf("priority-%dx%d-%s.txt", size[0], size[1], scenario)
				if err := os.WriteFile(filepath.Join(dir, name), []byte(view+"\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}

func TestPriorityExpansionAndQuerySurviveOtherViewRefresh(t *testing.T) {
	app := priorityApp(t)
	app.expanded[courseKey(1)] = true
	app.taskCursor = taskAppearance(app, "undated", courseKey(1))
	key := app.taskNodes()[app.taskCursor].key
	app.switchTab(TabMaterials)
	app.search = "not a task"
	app.filter = "unknown"
	app.captureCursors()
	app.Update(app.Init()())
	app.switchTab(TabTasks)
	if !app.expanded[courseKey(1)] || app.taskNodes()[app.taskCursor].key != key {
		t.Fatal("expanded course focus lost across other-view refresh")
	}
	app.expanded[taskShortlistKey] = true
	app.taskCursor = findNodeByKey(app.taskNodes(), nodeTaskMore, taskShortlistKey)
	press(app, "enter")
	if app.taskNodes()[app.taskCursor].key != taskShortlistKey {
		t.Fatal("show fewer lost focus")
	}
}

func TestPriorityCourseDecisionDoesNotFollowIntoHandledGroup(t *testing.T) {
	app := priorityApp(t)
	app.expanded[courseKey(1)] = true
	app.expanded[bucketKey(1, "Tracked")] = true
	app.taskCursor = taskAppearance(app, "current-0", courseKey(1))
	press(app, "t")
	if app.cursorAssignmentID() != "current-1" || len(app.taskNodes()[app.taskCursor].parents) != 1 {
		t.Fatal("focus followed handled task instead of next undecided task")
	}
}
