package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ruyam/kamps/internal/cache"
	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/source"
)

func TestWriteCommitPreviewWritesExactlyPreview(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.md")
	app := newDemoApp(t, config.Config{
		StateDir:     t.TempDir(),
		LibraryDir:   t.TempDir(),
		CommitFile:   path,
		ExportFormat: "markdown",
	})
	app.tab = TabCommit
	id := app.assignments[0].ID
	app.pending = map[string]model.Triage{id: model.Tracked}
	app.byID(id).Triage = model.Tracked

	preview := app.commitPreviewText()
	if preview == "" {
		t.Fatal("empty preview")
	}
	cmd := app.writeCommitPreview()
	if cmd == nil {
		t.Fatal("expected an export command")
	}
	app.Update(cmd())
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != preview {
		t.Fatalf("file = %q, preview = %q", data, preview)
	}
}

type expiringSource struct{ expiry time.Time }

func (s *expiringSource) Label() string { return "expiring" }
func (s *expiringSource) Load(context.Context) (source.Snapshot, error) {
	return source.Snapshot{}, nil
}
func (s *expiringSource) SessionStatus() source.SessionStatus {
	return source.SessionStatus{ExpiresAt: s.expiry, OK: true}
}

func TestSessionExpiryWarning(t *testing.T) {
	app := New(&expiringSource{expiry: time.Now().Add(10 * time.Minute)}, config.Config{StateDir: t.TempDir(), LibraryDir: t.TempDir()}, "test")
	app.Update(app.Init()())
	if !strings.Contains(app.toast, "session expires in") {
		t.Fatalf("toast = %q", app.toast)
	}
}

func TestInitShowsCacheBeforeRefresh(t *testing.T) {
	cfg := config.Config{StateDir: t.TempDir(), LibraryDir: t.TempDir(), SiteURL: "https://lms.test"}
	if err := cache.Save(cfg.CachePath(), cfg.SiteURL, source.Snapshot{Courses: []model.Course{{ID: 1, Name: "Cached"}}}); err != nil {
		t.Fatal(err)
	}
	app := New(&expiringSource{expiry: time.Now().Add(time.Hour)}, cfg, "test")
	if cmd := app.Init(); cmd == nil {
		t.Fatal("expected a refresh command")
	}
	if !app.loaded || len(app.courses) != 1 || app.courses[0].Name != "Cached" {
		t.Fatalf("cache was not shown: loaded=%v courses=%+v", app.loaded, app.courses)
	}
	if app.cacheNotice == "" {
		t.Fatal("cache notice is empty")
	}
}
