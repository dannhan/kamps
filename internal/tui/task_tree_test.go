package tui

import (
	"strings"
	"testing"

	"github.com/ruyam/kamps/internal/model"
)

func TestTaskAttachmentsRemainInInspector(t *testing.T) {
	app := demoApp(t)
	task := model.Assignment{ID: "K-custom", Name: "Assignment with attachments"}
	app.materials = append(app.materials, model.Material{ID: "unrelated-id", AssignmentID: task.ID, Name: "Instructions.pdf"})
	materials := app.assignmentMaterials(task)
	if len(materials) != 1 || materials[0].Name != "Instructions.pdf" {
		t.Fatal("explicit attachment association lost")
	}
	detail := strings.Join(app.detailLines(node{kind: nodeTask, task: task}, 100), "\n")
	if !strings.Contains(detail, "Instructions.pdf") {
		t.Fatal("inspector lost attachment")
	}
}

func TestTaskFocusAndMarksAreIndependent(t *testing.T) {
	app := demoApp(t)
	nodes := app.taskNodes()
	app.taskCursor = 1
	focus := stripANSI(app.taskTreeRow(nodes, 1, 80))
	if !strings.Contains(focus, "▌") || strings.Contains(focus, "▪") {
		t.Fatal("focus glyph")
	}
	app.selected[nodes[1].task.ID] = true
	app.taskCursor = 2
	marked := stripANSI(app.taskTreeRow(nodes, 1, 80))
	if strings.Contains(marked, "▌") || !strings.Contains(marked, "▪") {
		t.Fatal("mark glyph")
	}
	app.taskCursor = 1
	both := stripANSI(app.taskTreeRow(nodes, 1, 80))
	if !strings.Contains(both, "▌") || !strings.Contains(both, "▪") {
		t.Fatal("combined glyphs")
	}
}
