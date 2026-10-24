package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

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

func TestMaterialRowsKeepRightPadding(t *testing.T) {
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
	if !strings.Contains(markedRow, markFull) || !strings.Contains(unmarkedRow, markEmpty) {
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
		if fileIndex < 0 && n.kind == nodeFile && app.materialHasChildren(n) {
			fileIndex = i
		}
	}
	if courseIndex < 0 || fileIndex < 0 {
		t.Fatal("demo materials tree lacks a course or expandable file")
	}

	row := stripANSI(app.matTreeRow(nodes, courseIndex, 98, app.materialNodeSelected(nodes[courseIndex])))
	if !strings.Contains(row, "▾ "+markEmpty) {
		t.Fatalf("open course should show a down caret and hollow mark: %q", row)
	}
	for _, id := range app.materialNodeIDs(nodes[courseIndex]) {
		app.selectedM[id] = true
	}
	row = stripANSI(app.matTreeRow(nodes, courseIndex, 98, app.materialNodeSelected(nodes[courseIndex])))
	if !strings.Contains(row, "▾ "+markFull) {
		t.Fatalf("fully marked course should show %q: %q", markFull, row)
	}

	fileKey := nodes[fileIndex].key
	app.expanded[fileKey] = true
	row = stripANSI(app.matTreeRow(nodes, fileIndex, 98, false))
	if !strings.Contains(row, markEmpty+" ▾ ") {
		t.Fatalf("expandable file should show a down caret before its name: %q", row)
	}
	app.expanded[fileKey] = false
	row = stripANSI(app.matTreeRow(nodes, fileIndex, 98, false))
	if !strings.Contains(row, markEmpty+" ▸ ") {
		t.Fatalf("folded file should show a right caret before its name: %q", row)
	}
}
