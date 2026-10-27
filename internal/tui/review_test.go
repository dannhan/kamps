package tui

import (
	"errors"
	"testing"

	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/model"
)

func TestHelpViewportMatchesBodyHeight(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {56, 16}, {79, 30}, {80, 24}, {100, 23}, {120, 40}} {
		app := demoApp(t)
		app.width, app.height = size[0], size[1]
		_, got := app.helpViewport()
		want := bodyHeightFor(size[0], size[1])
		if got != want {
			t.Errorf("%dx%d help viewport = %d, want %d", size[0], size[1], got, want)
		}
	}
}

func TestPaletteLinesArePure(t *testing.T) {
	app := demoApp(t)
	app.paletteOn = true
	app.paletteCursor = 999
	_ = app.paletteLines(80)
	if app.paletteCursor != 999 {
		t.Fatalf("View mutated paletteCursor to %d", app.paletteCursor)
	}
}

func TestErrorWithoutCacheIsDurable(t *testing.T) {
	app := demoApp(t)
	_, cmd := app.Update(errMsg{err: errors.New("network down")})
	if cmd != nil {
		t.Fatal("fallback must finish in loadCmd before publishing an error")
	}
	if app.loadError != "network down" || app.loading || app.syncing {
		t.Fatalf("failure not settled: error=%q loading=%v syncing=%v", app.loadError, app.loading, app.syncing)
	}
}

func TestCopyPreviewReturnsACommand(t *testing.T) {
	app := demoApp(t)
	app.tab = TabCommit
	setPendingForTest(app, map[string]model.Triage{"K901000": model.Tracked})
	if cmd := app.copyCommitPreview(); cmd == nil {
		t.Fatal("copy preview should return a command, not do I/O in Update")
	}
}

func TestPersistenceQueueKeepsLatestDecision(t *testing.T) {
	cfg := config.Config{StateDir: t.TempDir()}
	app := newDemoApp(t, cfg)
	app.tab = TabTasks
	id := app.assignments[0].ID
	focusTaskForTest(t, app, id)

	app.selected[id] = true
	first := app.applyTriage(model.Tracked)
	if first == nil {
		t.Fatal("first decision returned no persistence command")
	}
	app.selected[id] = true
	if armed := app.applyTriage(model.Missed); armed != nil {
		t.Fatal("a hidden decision should arm instead of writing")
	}
	if second := app.applyTriage(model.Missed); second != nil {
		t.Fatal("a second decision should wait for the in-flight write")
	}

	_, follow := app.Update(first())
	if follow == nil {
		t.Fatal("the newer decision was not queued after the first write")
	}
	app.Update(follow())
	if app.persistInFlight || len(app.pendingDecisions) != 0 {
		t.Fatalf("queue did not settle: inFlight=%v pending=%d", app.persistInFlight, len(app.pendingDecisions))
	}

	again := newDemoApp(t, cfg)
	if assignment := again.byID(id); assignment == nil || assignment.Triage != model.Missed {
		t.Fatalf("latest decision did not win: %#v", assignment)
	}
}
