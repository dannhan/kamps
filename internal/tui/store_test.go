package tui

import (
	"os"
	"strings"
	"testing"

	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/model"
)

func TestDecisionsSurviveRestart(t *testing.T) {
	cfg := config.Config{StateDir: t.TempDir()}
	app := newDemoApp(t, cfg)
	app.tab = TabTasks
	id := app.assignments[0].ID
	app.selected[id] = true
	app.applyTriage(model.Tracked)

	if _, err := os.Stat(cfg.StatePath()); err != nil {
		t.Fatalf("decision did not reach the state file: %v", err)
	}

	again := newDemoApp(t, cfg)
	assignment := again.byID(id)
	if assignment == nil {
		t.Fatalf("restarted app lost assignment %s", id)
	}
	if assignment.Triage != model.Tracked {
		t.Fatalf("triage = %v, want tracked", assignment.Triage)
	}
	if len(again.pending) != 0 {
		t.Fatalf("restart should not rebuild the export queue, got %d items", len(again.pending))
	}
}

func TestUndecidedRemovesTheStoredDecision(t *testing.T) {
	cfg := config.Config{StateDir: t.TempDir()}
	app := newDemoApp(t, cfg)
	id := app.assignments[0].ID

	app.selected[id] = true
	app.applyTriage(model.Missed)
	app.selected[id] = true
	app.applyTriage(model.Undecided)

	again := newDemoApp(t, cfg)
	assignment := again.byID(id)
	if assignment == nil {
		t.Fatalf("restarted app lost assignment %s", id)
	}
	if assignment.Triage != model.Undecided {
		t.Fatalf("triage = %v, want undecided", assignment.Triage)
	}
}

func TestDownloadsSurviveRestart(t *testing.T) {
	cfg := config.Config{StateDir: t.TempDir()}
	app := newDemoApp(t, cfg)
	app.tab = TabMaterials

	id := ""
	for _, material := range app.materials {
		if !material.Downloaded && material.Kind == "file" {
			id = material.ID
			break
		}
	}
	if id == "" {
		t.Fatal("demo has no undownloaded material")
	}
	app.selectedM[id] = true
	app.download()

	again := newDemoApp(t, cfg)
	downloaded := false
	for _, material := range again.materials {
		if material.ID == id {
			downloaded = material.Downloaded
			break
		}
	}
	if !downloaded {
		t.Fatalf("download flag for %s did not survive the restart", id)
	}
}

func TestCorruptStateFallsBackAndWarns(t *testing.T) {
	cfg := config.Config{StateDir: t.TempDir()}
	if err := os.WriteFile(cfg.StatePath(), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	app := newDemoApp(t, cfg)
	if !app.loaded {
		t.Fatal("app did not load despite a corrupt state file")
	}
	if len(app.assignments) == 0 {
		t.Fatal("corrupt state should not drop source data")
	}
	if !strings.Contains(app.toast, "state:") {
		t.Fatalf("corrupt state should leave a warning, got toast %q", app.toast)
	}
}
