package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/source"
	"github.com/ruyam/kamps/internal/store"
)

func TestReconcileUsesRecordedFilename(t *testing.T) {
	dir := t.TempDir()
	actual := filepath.Join(dir, "slides.pptx")
	if err := os.WriteFile(actual, []byte("slides"), 0600); err != nil {
		t.Fatal(err)
	}
	s := &Service{cfg: config.Config{LibraryDir: dir, StateDir: t.TempDir()}, state: store.New()}
	s.state.SetDownloadRecord("M1", store.DownloadRecord{Path: actual})
	s.snapshot.Materials = []model.Material{{ID: "M1", Name: "0. Slide File", Downloaded: true}}
	if err := s.reconcile(); err != nil {
		t.Fatal(err)
	}
	if !s.snapshot.Materials[0].Downloaded || s.LocalPath(s.snapshot.Materials[0]) != actual {
		t.Fatalf("material %+v, path %q", s.snapshot.Materials[0], s.LocalPath(s.snapshot.Materials[0]))
	}
}

func TestRecordedServerFilenameSurvivesRestart(t *testing.T) {
	cfg := config.Config{LibraryDir: t.TempDir(), StateDir: t.TempDir()}
	actual := filepath.Join(cfg.LibraryDir, "actual.pdf")
	if err := os.WriteFile(actual, []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}
	state := store.New()
	state.SetDownloadRecord("M1", store.DownloadRecord{Path: actual})
	if err := state.Save(cfg.StatePath()); err != nil {
		t.Fatal(err)
	}
	src := stubSource{snapshot: source.Snapshot{Materials: []model.Material{{ID: "M1", Name: "LMS title File", Kind: "file"}}}}
	svc := New(src, cfg)
	snap, err := svc.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !snap.Materials[0].Downloaded || svc.LocalPath(snap.Materials[0]) != actual {
		t.Fatalf("material = %+v, path = %q", snap.Materials[0], svc.LocalPath(snap.Materials[0]))
	}
}
