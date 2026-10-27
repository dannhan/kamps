package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/source"
)

// A busy library patterned after real coursework, rather than a handful of demo rows.
func libraryWorkspaceApp(t *testing.T) *App {
	t.Helper()
	names := []string{"Struktur Data B", "Basis Data C", "Interaksi Manusia dan Komputer A", "Metode Numerik A", "Proyek Perangkat Lunak A"}
	snapshot := source.Snapshot{}
	for i, name := range names {
		id := i + 1
		snapshot.Courses = append(snapshot.Courses, model.Course{ID: id, Name: name, Code: fmt.Sprintf("MIK162430%d", id)})
		for week := 1; week <= 9; week++ {
			for file := 0; file < 3; file++ {
				material := model.Material{ID: fmt.Sprintf("M%d-%d-%d", id, week, file), CourseID: id, Name: fmt.Sprintf("%d. %s.pdf", week, []string{"ADT Titik", "Tugas Kelompok Aplikasi dan Perancangan Antarmuka 课程", "Latihan Mandiri"}[file]), Section: fmt.Sprintf("week-%d", week), Kind: "file", Downloaded: file == 0}
				if file == 2 {
					material.Folder = "Latihan"
				}
				snapshot.Materials = append(snapshot.Materials, material)
			}
		}
		snapshot.Assignments = append(snapshot.Assignments, model.Assignment{ID: fmt.Sprintf("K%d", id), CourseID: id, Name: []string{"Implementasi ADT Stack", "Normalisasi Basis Data", "Tugas UI/UX berdasarkan Gemastik", "Tugas Individu VIII", "Progress Report 6"}[i], HasDue: true, Due: dueOn(workFirstNow(), id), Triage: model.Undecided})
	}
	snapshot.Materials[0].AssignmentID = "K1"
	app := New(treeFixedSource{snapshot: snapshot}, config.Config{StateDir: t.TempDir(), Theme: "dark"}, "test")
	app.now = workFirstNow
	app.Update(app.Init()())
	app.clampCursor()
	return app
}

func TestLeavingLocalLibraryShowsCourseMaterials(t *testing.T) {
	app := libraryWorkspaceApp(t)
	app.switchTab(TabMaterials)
	app.courseID = localCourseID
	app.setPaletteFilter("unknown")
	app.courseCursor = 1
	app.chooseCourse()
	if len(app.visibleMaterials()) == 0 {
		t.Fatal("course materials remain hidden after leaving local library")
	}
}

func TestWorkspaceHierarchyAndContext(t *testing.T) {
	app := libraryWorkspaceApp(t)
	app.expanded[courseKey(1)] = true
	for i, n := range app.taskNodes() {
		if n.kind != nodeTask {
			continue
		}
		block := stripANSI(strings.Join(app.taskBlock(n, i, 100), "\n"))
		inCourse := n.parents[0] == courseKey(n.courseID)
		if strings.Contains(block, app.fullCourseName(n.courseID)) == inCourse {
			t.Fatalf("incorrect course context: %s", block)
		}
		if inCourse && strings.Index(block, n.task.Name) != 7 {
			t.Fatalf("child not indented under course: %q", block)
		}
		if strings.HasSuffix(block, "\n"+strings.Repeat(" ", 100)) {
			t.Fatal("task adds trailing blank row")
		}
	}
	app.switchTab(TabMaterials)
	expandAllMaterials(app)
	var lastCourse string
	lastWeek := ""
	for i, n := range app.matNodes() {
		block := stripANSI(strings.Join(app.materialBlock(n, i, 120), "\n"))
		if n.kind == nodeMaterialWeek {
			lastWeek = n.section
			lastCourse = ""
		}
		if n.kind == nodeCourse {
			if n.section != lastWeek {
				t.Fatal("course outside its week")
			}
			name := app.fullCourseName(n.courseID)
			if name < lastCourse {
				t.Fatal("courses not alphabetical")
			}
			lastCourse = name
		}
		if n.kind == nodeMaterialSection && strings.Contains(block, "MIK") {
			t.Fatal("week repeats course code")
		}
		if n.kind == nodeFile {
			if app.materialHasChildren(n) {
				t.Fatal("file expandable")
			}
			if n.mat.Folder == "" && strings.Index(block, cleanMaterialLabel(n.mat.Name)) != 9 {
				t.Fatalf("wrong file indentation: %q", block)
			}
			if n.mat.Downloaded && !strings.Contains(block, cleanMaterialLabel(n.mat.Name)+"  PDF · Local") {
				t.Fatalf("state detached from file: %q", block)
			}
		}
	}
}

func TestMaterialInteractionAndBlockScrolling(t *testing.T) {
	app := libraryWorkspaceApp(t)
	app.switchTab(TabMaterials)
	expandAllMaterials(app)
	app.width, app.height = 80, 24
	for i, n := range app.matNodes() {
		if n.kind != nodeFile {
			continue
		}
		app.matCursor = i
		rows, start, end := app.materialListWindow(app.matNodes(), 35, 8)
		if start > 0 && (app.matNodes()[start].kind == nodeFile || app.matNodes()[start].kind == nodeMaterialFolder) && !strings.HasPrefix(strings.TrimSpace(stripANSI(rows[0])), strings.TrimSuffix(truncate(app.fullCourseName(app.matNodes()[start].courseID), 20), "…")) {
			t.Fatal("scrolled files lost their course context")
		}
		if i < start || i >= end || len(rows) > 8 {
			t.Fatal("focused file outside window")
		}
	}
	for i, n := range app.matNodes() {
		if n.kind != nodeFile || n.mat.AssignmentID == "" {
			continue
		}
		app.matCursor = i
		press(app, " ")
		if !app.selectedM[n.mat.ID] {
			t.Fatal("file not marked")
		}
		press(app, "enter")
		if app.paneFocus != 2 {
			t.Fatal("Enter failed to open file inspector")
		}
		detail := stripANSI(strings.Join(app.materialDetailLines(n, 80), "\n"))
		if !strings.Contains(detail, "Related task") || !strings.Contains(detail, "Implementasi ADT Stack") {
			t.Fatal("related task missing")
		}
		press(app, "esc")
		if app.paneFocus != 0 || !app.selectedM[n.mat.ID] {
			t.Fatal("closing inspector lost focus or mark")
		}
		break
	}
	for i, n := range app.matNodes() {
		if n.kind != nodeMaterialWeek {
			continue
		}
		before := strings.Join(app.materialNodeIDs(n), ",")
		app.matCursor = i
		press(app, "enter")
		app.switchTab(TabTasks)
		app.switchTab(TabMaterials)
		if app.isOpen(n.key) {
			t.Fatal("collapse lost on tab switch")
		}
		if strings.Join(app.materialNodeIDs(n), ",") != before {
			t.Fatal("collapse changed batch targets")
		}
		break
	}
	app.offline = true
	app.cacheNotice = "cached coursework"
	footer := stripANSI(strings.Join(app.footerLines(80, sizeStandard), ""))
	for _, hint := range []string{"enter", "d download", "space mark", "? all keys"} {
		if !strings.Contains(footer, hint) {
			t.Fatalf("offline footer lost %s: %s", hint, footer)
		}
	}
}

func TestLibraryWorkspaceRendered(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	previousProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(previousProfile)
	for _, size := range [][2]int{{80, 24}, {120, 35}, {160, 45}} {
		for _, tab := range []Tab{TabTasks, TabMaterials} {
			app := libraryWorkspaceApp(t)
			app.width, app.height = size[0], size[1]
			app.switchTab(tab)
			if tab == TabTasks {
				for _, c := range app.courses {
					app.expanded[courseKey(c.ID)] = true
				}
			}
			app.downloads["M2-1-1"] = true
			app.downloadProgress["M2-1-1"] = "42%"
			app.downloadErrors["M2-2-1"] = "session expired"
			view := app.View()
			lines := strings.Split(stripANSI(view), "\n")
			if len(lines) != size[1] {
				t.Fatal("height mismatch")
			}
			for _, line := range lines {
				if lipgloss.Width(line) != size[0] {
					t.Fatalf("width mismatch: %q", line)
				}
			}
			if dir := os.Getenv("KAMPS_RENDER_DIR"); dir != "" {
				if err := os.MkdirAll(dir, 0700); err != nil {
					t.Fatal(err)
				}
				name := fmt.Sprintf("library-%s-%dx%d", tab, size[0], size[1])
				for ext, content := range map[string]string{".txt": stripANSI(view), ".ansi": view} {
					if err := os.WriteFile(filepath.Join(dir, name+ext), []byte(content+"\n"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
		}
	}
}

func TestCourseTaskDeadlinesStayInlineOnlyWithinCourseGroups(t *testing.T) {
	app := libraryWorkspaceApp(t)
	assignment := app.assignments[0]
	for _, parents := range [][]string{
		{courseKey(assignment.CourseID)},
		{courseKey(assignment.CourseID), bucketKey(assignment.CourseID, "Tracked")},
		{"tasks:attention"},
		{taskOverdueKey},
	} {
		n := node{kind: nodeTask, courseID: assignment.CourseID, task: assignment, parents: parents}
		inCourse := parents[0] == courseKey(assignment.CourseID)
		for _, width := range []int{80, 120, 160} {
			rows := app.taskBlock(n, 0, width)
			first := stripANSI(rows[0])
			inline := strings.Contains(first, assignment.Name+" · "+readableDeadline(assignment, app))
			if inline != inCourse {
				t.Fatalf("parents %v: wrong deadline placement: %q", parents, rows)
			}
			if inCourse && len(rows) != 1 {
				t.Fatalf("short course task occupies %d lines", len(rows))
			}
		}
	}
	n := node{kind: nodeTask, courseID: assignment.CourseID, task: assignment, parents: []string{courseKey(assignment.CourseID)}}
	n.task.Name = strings.Repeat("课程 👩🏽‍💻 Long title ", 20)
	for _, width := range []int{24, 80, 120, 160} {
		rows := app.taskBlock(n, 0, width)
		for _, row := range rows {
			if lipgloss.Width(row) != width {
				t.Fatalf("long course task exceeds %d columns: %q", width, row)
			}
		}
		last := stripANSI(rows[len(rows)-1])
		if !strings.Contains(last, readableDeadline(n.task, app)) {
			t.Fatalf("deadline lost from long title: %q", rows)
		}
		if width >= 80 && !strings.Contains(last, " · ") {
			t.Fatalf("wrapped course task lost inline separator: %q", last)
		}
	}
}
