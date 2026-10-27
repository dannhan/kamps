package tui

import (
	"testing"

	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/core"
	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/source"
)

func demoApp(t *testing.T) *App {
	t.Helper()
	return newDemoApp(t, config.Config{StateDir: t.TempDir()})
}

func newDemoApp(t *testing.T, cfg config.Config) *App {
	t.Helper()
	app := New(source.NewDemo(), cfg, "test")
	app.Update(app.Init()())
	expandAllMaterials(app)
	return app
}

// expandAllMaterials opens the week and folder rows so rendering tests do not
// have to repeat the default-collapsed navigation.
func expandAllMaterials(app *App) {
	app.tabExpanded[TabMaterials] = app.expanded
	for pass := 0; pass < 3; pass++ {
		for _, n := range app.matNodes() {
			if n.kind == nodeMaterialWeek || n.kind == nodeMaterialSection || n.kind == nodeMaterialFolder {
				app.expanded[n.key] = true
			}
		}
	}
}

func findTaskNode(nodes []node, kind nodeKind, value string) int {
	for i, n := range nodes {
		if n.kind != kind {
			continue
		}
		switch kind {
		case nodeCourse:
			if n.courseID == 18515 {
				return i
			}
		case nodeBucket:
			if n.bucket == value && n.courseID == 18515 {
				return i
			}
		case nodeTask:
			if n.task.ID == value {
				return i
			}
		case nodeTaskMaterial:
			if n.mat.ID == value {
				return i
			}
		}
	}
	return -1
}

func setPendingForTest(app *App, pending map[string]model.Triage) {
	app.queue = core.NewQueue()
	for id, action := range pending {
		app.queue.Set(id, action)
	}
}

func taskTreeRowForTest(t *testing.T) string {
	t.Helper()
	app := demoApp(t)
	app.showAll = true
	app.taskCursor = findTaskNode(app.taskNodes(), nodeTask, "K901013")
	return app.taskTreeRow(app.taskNodes(), app.taskCursor, 98)
}

// focusTaskForTest explicitly opens the course and decision subgroup needed by
// action tests; opening-screen tests continue to use the actual collapsed defaults.
func focusTaskForTest(t *testing.T, app *App, id string) {
	t.Helper()
	assignment := app.byID(id)
	if assignment == nil {
		t.Fatalf("unknown task %s", id)
	}
	app.expanded[courseKey(assignment.CourseID)] = true
	if assignment.Triage == model.Tracked {
		app.expanded[bucketKey(assignment.CourseID, "Tracked")] = true
	}
	if assignment.Triage == model.Missed || assignment.Triage == model.Ignored {
		app.expanded[bucketKey(assignment.CourseID, "Closed")] = true
	}
	for i, n := range app.taskNodes() {
		if n.kind == nodeTask && n.task.ID == id {
			app.taskCursor = i
			return
		}
	}
	t.Fatalf("task %s absent after opening course", id)
}
