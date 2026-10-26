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
	if cmd := app.applyTriage(model.Tracked); cmd != nil {
		app.Update(cmd())
	}

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
	if again.queue.Len() != 0 {
		t.Fatalf("restart should not rebuild the export queue, got %d items", again.queue.Len())
	}
}

func TestUndecidedRemovesTheStoredDecision(t *testing.T) {
	cfg := config.Config{StateDir: t.TempDir()}
	app := newDemoApp(t, cfg)
	id := app.assignments[0].ID

	app.selected[id] = true
	if cmd := app.applyTriage(model.Missed); cmd != nil {
		app.Update(cmd())
	}
	app.selected[id] = true
	if cmd := app.applyTriage(model.Undecided); cmd != nil {
		app.Update(cmd())
	}

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
	if cmd := app.download(); cmd != nil {
		app.Update(cmd())
	}

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

func TestTriageDefersPersistenceToCommand(t *testing.T) {
	cfg := config.Config{StateDir: t.TempDir()}
	app := newDemoApp(t, cfg)
	app.tab = TabTasks
	id := app.assignments[0].ID
	app.selected[id] = true

	cmd := app.applyTriage(model.Ignored)
	if cmd == nil {
		t.Fatal("applyTriage returned no persistence command")
	}
	if !app.persistInFlight || len(app.pendingDecisions) == 0 {
		t.Fatalf("decision was not queued: inFlight=%v pending=%d", app.persistInFlight, len(app.pendingDecisions))
	}

	app.Update(cmd())
	if app.persistInFlight || len(app.pendingDecisions) != 0 {
		t.Fatalf("save did not settle: inFlight=%v pending=%d", app.persistInFlight, len(app.pendingDecisions))
	}
	again := newDemoApp(t, cfg)
	if assignment := again.byID(id); assignment == nil || assignment.Triage != model.Ignored {
		t.Fatalf("decision did not persist: %#v", assignment)
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
	if !strings.Contains(app.statusErr, "state:") {
		t.Fatalf("corrupt state should leave a durable warning, got %q", app.statusErr)
	}
}
