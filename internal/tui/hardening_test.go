package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ruyam/kamps/internal/config"
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

func TestUnknownFilesFilterShowsLocalFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "stray.pdf"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	app := newDemoApp(t, config.Config{StateDir: t.TempDir(), LibraryDir: dir})
	app.tab = TabMaterials
	app.filter = "unknown"
	app.reconcileLibrary()
	if got := len(app.visibleMaterials()); got != 1 {
		t.Fatalf("unknown materials = %d, want 1", got)
	}
	nodes := app.matNodes()
	found := false
	for i, n := range nodes {
		if n.kind == nodeFile && n.mat.Kind == "local" {
			found = true
			app.matCursor = i
		}
	}
	if !found {
		t.Fatalf("local file not in tree: %+v", nodes)
	}
	if cmd := app.download(); cmd != nil {
		t.Fatal("local files should not start a download")
	}
	if !strings.Contains(app.toast, "local file") {
		t.Fatalf("toast = %q", app.toast)
	}
}
