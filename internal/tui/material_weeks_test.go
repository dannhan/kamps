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

func TestAllMaterialsStartsWithNewestCollapsedWeeks(t *testing.T) {
	app := libraryWorkspaceApp(t)
	app.switchTab(TabMaterials)
	nodes := app.matNodes()
	if len(nodes) != 9 {
		t.Fatalf("overview has %d rows, want 9 weeks", len(nodes))
	}
	for index, n := range nodes {
		if n.kind != nodeMaterialWeek || n.section != fmt.Sprintf("week-%d", 9-index) {
			t.Fatalf("wrong overview order: %+v", n)
		}
		if app.isOpen(n.key) || len(n.parents) > 0 {
			t.Fatal("overview week starts expanded or nested")
		}
		if len(app.materialNodeIDs(n)) != 15 {
			t.Fatal("week failed to aggregate all five courses")
		}
	}
	row := stripANSI(strings.Join(app.materialBlock(nodes[0], 0, 80), "\n"))
	if !strings.Contains(row, "15 files · 5 courses") {
		t.Fatalf("missing week counts: %s", row)
	}
	press(app, "enter")
	foundCourse, foundFile := false, false
	for _, n := range app.matNodes() {
		if n.kind == nodeCourse {
			foundCourse = true
			if n.section != "week-9" {
				t.Fatal("course outside focused week")
			}
			if len(app.materialNodeIDs(n)) != 3 {
				t.Fatal("course group targets files outside its week")
			}
		}
		if n.kind == nodeFile {
			foundFile = true
			if n.mat.Section != "week-9" {
				t.Fatal("expanded week leaks another week")
			}
		}
	}
	if !foundCourse || !foundFile {
		t.Fatal("Enter failed to reveal week materials")
	}
}

func TestMaterialsWeekSortPreservesFocusMarksAndExpansion(t *testing.T) {
	app := libraryWorkspaceApp(t)
	app.switchTab(TabMaterials)
	press(app, "enter")
	for i, n := range app.matNodes() {
		if n.kind == nodeFile {
			app.matCursor = i
			break
		}
	}
	focused := app.matNodes()[app.matCursor]
	press(app, " ")
	press(app, "S")
	if !app.materialsOldestFirst || app.matNodes()[app.matCursor].key != focused.key {
		t.Fatal("sort lost focus")
	}
	if !app.selectedM[focused.mat.ID] || !app.isOpen(materialWeekKey("week-9")) {
		t.Fatal("sort lost marks or expansion")
	}
	if app.matNodes()[0].section != "week-1" {
		t.Fatal("oldest-first not applied")
	}
	app.switchTab(TabTasks)
	app.switchTab(TabMaterials)
	if !app.materialsOldestFirst {
		t.Fatal("tab switch reset sort")
	}
	press(app, "S")
	if app.matNodes()[0].section != "week-9" || app.matNodes()[app.matCursor].key != focused.key {
		t.Fatal("newest-first not restored")
	}
	app.paneFocus = 1
	for i, c := range app.sidebarCourses() {
		if c.ID == 1 {
			app.courseCursor = i + 1
			break
		}
	}
	press(app, "enter")
	if !app.courseScoped || app.courseID != 1 {
		t.Fatal("sidebar did not choose course")
	}
	for _, n := range app.matNodes() {
		if n.kind == nodeCourse || n.kind == nodeMaterialWeek {
			t.Fatal("course library has redundant grouping")
		}
		if n.kind == nodeFile && n.mat.CourseID != 1 {
			t.Fatal("sidebar selection leaks other courses")
		}
	}
	if app.matNodes()[0].section != "week-9" {
		t.Fatal("selected course does not honor sort")
	}
	if !app.selectedM[focused.mat.ID] {
		t.Fatal("sidebar selection cleared batch mark")
	}
}

func TestWeekBatchTargetsRespectFiltersAndCollapse(t *testing.T) {
	app := libraryWorkspaceApp(t)
	app.switchTab(TabMaterials)
	week := app.matNodes()[0]
	press(app, " ")
	if len(app.selectedM) != 15 || len(app.materialTargets()) != 15 {
		t.Fatal("collapsed week failed to mark its files")
	}
	press(app, "enter")
	press(app, "enter")
	if len(app.selectedM) != 15 {
		t.Fatal("collapse lost batch marks")
	}
	press(app, " ")
	app.setPaletteFilter("downloaded")
	week = app.matNodes()[0]
	if len(app.materialNodeFiles(week)) != 5 {
		t.Fatal("filtered week includes undownloaded files")
	}
	press(app, "enter")
	for i, n := range app.matNodes() {
		if n.kind == nodeCourse {
			if len(app.materialNodeFiles(n)) != 1 {
				t.Fatal("filtered course targets other files")
			}
			app.matCursor = i
			if len(app.materialTargets()) != 1 {
				t.Fatal("download targets ignore week/filter")
			}
			break
		}
	}
	app.search = "Tugas Kelompok"
	if len(app.matNodes()) != 0 {
		t.Fatal("search should compose with downloaded filter")
	}
	app.setPaletteFilter("all")
	if len(app.matNodes()) == 0 {
		t.Fatal("search lost matching weeks")
	}
	for _, n := range app.matNodes() {
		if n.kind == nodeFile && !strings.Contains(n.mat.Name, "Tugas Kelompok") {
			t.Fatal("search leaks nonmatching file")
		}
	}
}

func TestMaterialsWeeksRendered(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	profile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(profile)
	for _, size := range [][2]int{{80, 24}, {120, 35}, {160, 45}} {
		for _, state := range []string{"overview", "expanded", "oldest", "course", "search", "empty"} {
			app := libraryWorkspaceApp(t)
			app.width, app.height = size[0], size[1]
			app.switchTab(TabMaterials)
			switch state {
			case "expanded":
				press(app, "enter")
			case "oldest":
				press(app, "S")
			case "course":
				app.courseScoped, app.courseID = true, 1
				app.matCursor = 0
			case "search":
				app.search = "Tugas Kelompok"
			case "empty":
				app.search = "zzzz-nonexistent"
			}
			view := app.View()
			lines := strings.Split(stripANSI(view), "\n")
			if len(lines) != size[1] {
				t.Fatal("wrong rendered height")
			}
			for _, line := range lines {
				if lipgloss.Width(line) != size[0] {
					t.Fatalf("wrong rendered width: %q", line)
				}
			}
			if !strings.Contains(stripANSI(view), app.materialSortLabel()) {
				t.Fatal("sort not visible")
			}
			if dir := os.Getenv("KAMPS_RENDER_DIR"); dir != "" {
				if err := os.MkdirAll(dir, 0700); err != nil {
					t.Fatal(err)
				}
				base := fmt.Sprintf("weeks-%dx%d-%s", size[0], size[1], state)
				for ext, content := range map[string]string{".txt": stripANSI(view), ".ansi": view} {
					if err := os.WriteFile(filepath.Join(dir, base+ext), []byte(content+"\n"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
		}
	}
}

func TestMaterialWeeksUseNumericOrderAndKeepOtherSections(t *testing.T) {
	snapshot := source.Snapshot{Courses: []model.Course{{ID: 1, Name: "Course"}}}
	for index, section := range []string{"week-2", "week-10", "week-1", "", "Resources"} {
		snapshot.Materials = append(snapshot.Materials, model.Material{ID: fmt.Sprintf("M%d", index), CourseID: 1, Name: "File", Section: section})
	}
	app := New(treeFixedSource{snapshot: snapshot}, config.Config{StateDir: t.TempDir()}, "test")
	app.Update(app.Init()())
	app.switchTab(TabMaterials)
	for _, expected := range [][]string{{"week-10", "week-2", "week-1", "week-0", "Resources"}, {"week-0", "week-1", "week-2", "week-10", "Resources"}} {
		nodes := app.matNodes()
		if len(nodes) != len(expected) {
			t.Fatal("sections lost")
		}
		for index, want := range expected {
			if nodes[index].section != want {
				t.Fatalf("section %d=%s, want %s", index, nodes[index].section, want)
			}
		}
		press(app, "S")
	}
}
