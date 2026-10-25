package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSearchStableAfterDownloadCompletion(t *testing.T) {
	app := demoApp(t)
	app.tab = TabMaterials
	app.search = "a"
	app.matCursor = 0
	material := app.materials[0]
	app.Update(downloadDoneMsg{id: material.ID, path: "/tmp/file"})
	if app.search != "a" {
		t.Fatalf("search = %q", app.search)
	}
	if nodes := app.matNodes(); app.matCursor < 0 || app.matCursor >= len(nodes) {
		t.Fatalf("cursor %d out of range (%d nodes)", app.matCursor, len(nodes))
	}
	_ = app.View()
}

func TestResizeDuringDownloadDoesNotPanic(t *testing.T) {
	app := demoApp(t)
	app.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	app.Update(downloadProgressMsg{id: app.materials[0].ID, done: 1, total: 10})
	if view := app.View(); view == "" {
		t.Fatal("empty view")
	}
}
