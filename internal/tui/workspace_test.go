package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ruyam/kamps/internal/model"
)

func press(app *App, key string) {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	switch key {
	case "tab":
		msg = tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		msg = tea.KeyMsg{Type: tea.KeyShiftTab}
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	case " ":
		msg = tea.KeyMsg{Type: tea.KeySpace}
	}
	app.handleKey(msg)
}
func TestWorkspaceTaskFilters(t *testing.T) {
	app := demoApp(t)
	for _, filter := range []string{"attention", "upcoming", "closed"} {
		app.filter = filter
		for _, a := range app.visible() {
			if filter == "attention" && a.Triage != model.Undecided {
				t.Fatal("attention includes handled work")
			}
			if filter == "upcoming" && (a.Triage != model.Undecided || !a.HasDue || dueDays(a, app.currentTime()) < 0 || dueDays(a, app.currentTime()) > 7) {
				t.Fatal("upcoming semantics")
			}
			if filter == "closed" && a.Triage != model.Missed && a.Triage != model.Ignored {
				t.Fatal("closed includes tracked")
			}
		}
	}
	app.filter = "all"
	press(app, "f")
	if !app.filterOn {
		t.Fatal("filter menu did not open")
	}
	press(app, "k")
	press(app, "k")
	press(app, "enter")
	if app.filter != "attention" {
		t.Fatal("filter not applied")
	}
}
func TestWorkspaceFocusSelectionAndCourses(t *testing.T) {
	app := demoApp(t)
	app.width = 160
	id := app.taskNodes()[app.taskCursor].task.ID
	press(app, " ")
	press(app, "j")
	if !app.selected[id] || app.taskCursor != 2 {
		t.Fatal("mark and cursor coupled")
	}
	press(app, "tab")
	if app.paneFocus != 1 || app.tab != TabTasks {
		t.Fatal("tab should focus courses")
	}
	press(app, "t")
	if app.queue.Len() != 0 {
		t.Fatal("sidebar triaged hidden task")
	}
	for i, c := range app.sidebarCourses() {
		if c.ID == 0 {
			app.courseCursor = i + 1
			break
		}
	}
	press(app, "enter")
	if !app.courseScoped || app.courseID != 0 {
		t.Fatal("manual course zero cannot be scoped")
	}
	for _, a := range app.visible() {
		if a.CourseID != 0 {
			t.Fatal("manual course includes other courses")
		}
	}

	for _, a := range app.visible() {
		if a.Triage == model.Undecided {
			focusTaskForTest(t, app, a.ID)
			break
		}
	}
	press(app, "enter")
	if !app.detailVisible() || app.paneFocus != 2 {
		t.Fatal("inspector not focused")
	}
	press(app, "j")
	if app.detailScroll != minInt(1, app.detailMaxScroll()) {
		t.Fatal("inspector does not scroll")
	}
	press(app, "esc")
	if app.detailVisible() || app.paneFocus != 0 {
		t.Fatal("escape does not close inspector")
	}
	if !app.selected[id] {
		t.Fatal("course change lost marked item")
	}
	app.filter = "all"
	app.search = "quiz"
	press(app, "2")
	press(app, "1")
	if app.search != "quiz" {
		t.Fatal("view switch lost query")
	}
}
func TestWorkspaceHiddenSelectionStillConfirms(t *testing.T) {
	app := demoApp(t)
	id := app.taskNodes()[app.taskCursor].task.ID
	app.selected[id] = true
	app.search = "no-match-zzzz"
	if app.applyTriage(model.Ignored) != nil || !app.triageArmed {
		t.Fatal("hidden selection bypassed confirmation")
	}
	press(app, "esc")
	if app.triageArmed {
		t.Fatal("escape did not cancel")
	}
}
func TestWorkspaceDeadlineAndUnicode(t *testing.T) {
	app := demoApp(t)
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	app.now = func() time.Time { return now }
	for days, want := range map[int]string{0: "Today", 4: "In 4d", -21: "21d overdue"} {
		a := model.Assignment{HasDue: true, Due: now.AddDate(0, 0, days)}
		if got := readableDeadline(a, app); got != want {
			t.Fatalf("%s != %s", got, want)
		}
	}
	for _, line := range wrapCell(strings.Repeat("课程 👩🏽‍💻 café é ", 30), 20) {
		if !utf8.ValidString(line) || lipgloss.Width(line) > 20 {
			t.Fatalf("invalid wrapping: %q", line)
		}
	}
}
func TestWorkspaceMaterialsAndDisplayLabels(t *testing.T) {
	app := demoApp(t)
	app.switchTab(TabMaterials)
	for i, n := range app.matNodes() {
		if n.kind == nodeMaterialSection || n.kind == nodeMaterialWeek {
			app.matCursor = i
			key := n.key
			press(app, "enter")
			if app.isOpen(key) {
				t.Fatal("enter did not collapse week")
			}
			press(app, "enter")
			if !app.isOpen(key) {
				t.Fatal("enter did not open week")
			}
			break
		}
	}
	for i, n := range app.matNodes() {
		if n.kind == nodeFile {
			app.matCursor = i
			press(app, " ")
			rows := stripANSI(strings.Join(app.quietMaterialRows(app.matNodes(), 100, 1000), "\n"))
			if !strings.Contains(rows, "▪") || !strings.Contains(rows, "▌") {
				t.Fatal("focus and mark missing")
			}
			break
		}
	}
	original := "Lecture_01   Notes.pdf"
	if cleanMaterialLabel(original) != "Lecture 01 Notes.pdf" || original != "Lecture_01   Notes.pdf" {
		t.Fatal("display cleaning")
	}
}
func TestWorkspaceRenderedStates(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {120, 35}, {160, 45}} {
		for _, tab := range []Tab{TabTasks, TabMaterials} {
			for _, state := range []string{"normal", "inspector", "courses", "empty", "offline", "loading", "error", "long"} {
				app := demoApp(t)
				app.width, app.height = size[0], size[1]
				app.switchTab(tab)
				switch state {
				case "inspector":
					if tab == TabMaterials {
						for i, n := range app.matNodes() {
							if n.kind == nodeFile {
								app.matCursor = i
								break
							}
						}
					}
					press(app, "enter")
				case "courses":
					press(app, "tab")
				case "empty":
					app.search = "nonexistent-zzzz"
				case "offline":
					app.offline = true
					app.cacheNotice = "cached coursework · s retry"
				case "loading":
					app.loaded = false
				case "error":
					app.statusErr = "Download failed: session expired · run kamps login"
				case "long":
					snapshot := app.service.Snapshot()
					for i := range snapshot.Assignments {
						snapshot.Assignments[i].Name = strings.Repeat("课程 Long title 👩🏽‍💻 ", 10)
					}
					for i := range snapshot.Materials {
						snapshot.Materials[i].Name = strings.Repeat("课程_Long_title_", 10) + ".pdf"
					}
					app.Update(loadedMsg{snapshot})
				}
				view := stripANSI(app.View())
				lines := strings.Split(view, "\n")
				if len(lines) != size[1] {
					t.Fatalf("%v %s %s height %d", size, tab, state, len(lines))
				}
				for _, line := range lines {
					if !utf8.ValidString(line) || lipgloss.Width(line) != size[0] {
						t.Fatalf("%v %s %s invalid row %q", size, tab, state, line)
					}
				}
				if dir := os.Getenv("KAMPS_RENDER_DIR"); dir != "" {
					if err := os.MkdirAll(dir, 0700); err != nil {
						t.Fatal(err)
					}
					name := fmt.Sprintf("%s-%dx%d-%s.txt", tab, size[0], size[1], state)
					if err := os.WriteFile(filepath.Join(dir, name), []byte(view+"\n"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
		}
	}
}

func TestWorkspaceProgressAndFailures(t *testing.T) {
	app := demoApp(t)
	app.switchTab(TabMaterials)
	var id string
	for i, n := range app.matNodes() {
		if n.kind == nodeFile && !n.mat.Downloaded {
			app.matCursor = i
			id = n.mat.ID
			break
		}
	}
	app.downloads[id] = true
	app.Update(downloadProgressMsg{id: id, done: 25, total: 100})
	if app.downloadProgress[id] != "25%" {
		t.Fatal("progress not retained")
	}
	rows := strings.Join(app.quietMaterialRows(app.matNodes(), 120, 1000), "\n")
	if !strings.Contains(rows, "Downloading 25%") {
		t.Fatal("progress absent from row")
	}
	app.Update(downloadDoneMsg{id: id, err: fmt.Errorf("network unavailable")})
	if app.downloadErrors[id] == "" || app.downloads[id] {
		t.Fatal("failure state lost")
	}
	rows = strings.Join(app.quietMaterialRows(app.matNodes(), 120, 1000), "\n")
	if !strings.Contains(rows, "Failed · d retry") {
		t.Fatal("retry hint absent")
	}
}

func TestWorkspaceUpcomingBoundaries(t *testing.T) {
	now := workFirstNow()
	app := workFirstApp(t, now, []model.Course{{ID: 1, Name: "Course"}}, []model.Assignment{
		{ID: "past", HasDue: true, Due: dueOn(now, -1)}, {ID: "today", HasDue: true, Due: dueOn(now, 0)},
		{ID: "week", HasDue: true, Due: dueOn(now, 7)}, {ID: "later", HasDue: true, Due: dueOn(now, 8)},
		{ID: "undated"}, {ID: "handled", Triage: model.Tracked, HasDue: true, Due: dueOn(now, 1)},
	})
	app.filter = "upcoming"
	items := app.visible()
	if len(items) != 2 || items[0].ID != "today" || items[1].ID != "week" {
		t.Fatalf("upcoming: %v", items)
	}
	app.filter = "attention"
	if len(app.visible()) != 5 {
		t.Fatal("attention must include undated and stale work")
	}
}

func TestWorkspaceIgnoreShortcutAndInspector(t *testing.T) {
	app := demoApp(t)
	id := app.cursorAssignmentID()
	before := app.byID(id).Triage
	press(app, "enter")
	if app.byID(id).Triage != before || !app.detailVisible() {
		t.Fatal("Enter should inspect without deciding")
	}
	press(app, "i")
	if app.byID(id).Triage != model.Ignored {
		t.Fatal("i did not ignore")
	}
}
