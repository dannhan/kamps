package tui

import (
	"testing"

	"github.com/ruyam/kamps/internal/config"
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
	return app
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

func taskTreeRowForTest(t *testing.T) string {
	t.Helper()
	app := demoApp(t)
	app.taskCursor = findTaskNode(app.taskNodes(), nodeTask, "K901013")
	return app.taskTreeRow(app.taskNodes(), app.taskCursor, 98)
}
