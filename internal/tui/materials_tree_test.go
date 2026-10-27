package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestMaterialsTreeKeepsRelatedAssignmentsInInspector(t *testing.T) {
	app := demoApp(t)
	app.tab = TabMaterials
	nodes := app.matNodes()
	foundSection, foundAssignment := false, false
	for _, n := range nodes {
		foundSection = foundSection || (n.kind == nodeMaterialSection || n.kind == nodeMaterialWeek)
		if n.kind == nodeMaterialAssignment {
			t.Fatal("linked task leaked into file list")
		}
		if n.kind == nodeFile {
			if task := app.assignmentForMaterial(n.mat); task != nil && task.ID == "K165483" {
				details := stripANSI(strings.Join(app.materialDetailLines(n, 80), "\n"))
				foundAssignment = strings.Contains(details, task.Name)
			}
		}
	}
	if !foundSection || !foundAssignment {
		t.Fatalf("Materials tree lacks a week or attached assignment: section=%v assignment=%v", foundSection, foundAssignment)
	}
	app.filter = "downloaded"
	for _, n := range app.matNodes() {
		if n.kind == nodeFile && !n.mat.Downloaded {
			t.Fatalf("downloaded filter included new material %q", n.mat.Name)
		}
	}
}

func TestMaterialRowsFillTheAvailableWidth(t *testing.T) {
	app := demoApp(t)
	app.tab = TabMaterials
	nodes := app.matNodes()
	if len(nodes) == 0 {
		t.Fatal("demo data has no material rows")
	}
	for i := range nodes {
		row := stripANSI(app.matTreeRow(nodes, i, 60, false))
		if got := lipgloss.Width(row); got != 60 {
			t.Fatalf("material row %d width %d, want 60: %q", i, got, row)
		}
		if !strings.HasSuffix(row, "  ") {
			t.Fatalf("material row %d has no right padding: %q", i, row)
		}
	}
}

func TestMaterialRowsShowMarkState(t *testing.T) {
	app := demoApp(t)
	app.tab = TabMaterials

	nodes := app.matNodes()
	file := -1
	for i, n := range nodes {
		if n.kind == nodeFile {
			file = i
			break
		}
	}
	if file < 0 {
		t.Fatal("demo materials tree has no file rows")
	}
	markedRow := stripANSI(app.matTreeRow(nodes, file, 98, true))
	unmarkedRow := stripANSI(app.matTreeRow(nodes, file, 98, false))
	if !strings.Contains(markedRow, markFull) || strings.Contains(unmarkedRow, markFull) || strings.Contains(unmarkedRow, "▫") {
		t.Fatalf("material mark glyphs = %q / %q", markedRow, unmarkedRow)
	}
}

func TestMaterialParentRowsShowCaretAndMark(t *testing.T) {
	app := demoApp(t)
	app.tab = TabMaterials

	nodes := app.matNodes()
	if len(nodes) == 0 {
		t.Fatal("demo materials tree is empty")
	}
	courseIndex := -1
	fileIndex := -1
	for i, n := range nodes {
		if courseIndex < 0 && n.kind == nodeCourse {
			courseIndex = i
		}
		if fileIndex < 0 && n.kind == nodeFile {
			fileIndex = i
		}
	}
	if courseIndex < 0 || fileIndex < 0 {
		t.Fatal("demo materials tree lacks a course or file")
	}

	row := stripANSI(app.matTreeRow(nodes, courseIndex, 98, app.materialNodeSelected(nodes[courseIndex])))
	if !strings.Contains(row, "▾") || strings.Contains(row, "▫") {
		t.Fatalf("open course should show a down caret and no placeholder mark: %q", row)
	}
	for _, id := range app.materialNodeIDs(nodes[courseIndex]) {
		app.selectedM[id] = true
	}
	row = stripANSI(app.matTreeRow(nodes, courseIndex, 98, app.materialNodeSelected(nodes[courseIndex])))
	if !strings.Contains(row, "▾ "+markFull) {
		t.Fatalf("fully marked course should show %q: %q", markFull, row)
	}

	row = stripANSI(app.matTreeRow(nodes, fileIndex, 98, false))
	if strings.ContainsAny(row, "▾▸") {
		t.Fatalf("file should open details, not advertise children: %q", row)
	}

}

func TestMaterialRowsShowDownloadedMarkerOnly(t *testing.T) {
	app := demoApp(t)
	app.tab = TabMaterials
	nodes := app.matNodes()
	var downloaded, fresh = -1, -1
	for i, n := range nodes {
		if n.kind != nodeFile {
			continue
		}
		if n.mat.Downloaded && downloaded < 0 {
			downloaded = i
		}
		if !n.mat.Downloaded && fresh < 0 {
			fresh = i
		}
	}
	if downloaded < 0 || fresh < 0 {
		t.Fatal("demo materials need both downloaded and new files")
	}
	if row := stripANSI(app.matTreeRow(nodes, downloaded, 98, false)); !strings.Contains(row, "●") || strings.Contains(row, "downloaded") {
		t.Fatalf("downloaded file row = %q", row)
	}
	if row := stripANSI(app.matTreeRow(nodes, fresh, 98, false)); !strings.Contains(row, "○") {
		t.Fatalf("new file row = %q", row)
	}
}

func TestCourseMaterialsStartWithSectionsOpen(t *testing.T) {
	app := demoApp(t)
	app.expanded = map[string]bool{}
	app.tab = TabMaterials
	app.courseScoped, app.courseID = true, 18515
	nodes := app.matNodes()
	foundSection, foundFile, foundFolder := false, false, false
	folderFile := false
	for _, n := range nodes {
		foundSection = foundSection || (n.kind == nodeMaterialSection || n.kind == nodeMaterialWeek)
		foundFile = foundFile || n.kind == nodeFile
		foundFolder = foundFolder || n.kind == nodeMaterialFolder
		if n.kind == nodeFile && n.mat.Folder != "" {
			folderFile = true
		}
	}
	if !foundSection || !foundFile {
		t.Fatalf("materials should start with weeks open: section=%v file=%v", foundSection, foundFile)
	}
	if foundFolder && !folderFile {
		t.Fatal("folders should initially reveal their files")
	}
}
