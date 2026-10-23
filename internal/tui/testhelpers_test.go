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
