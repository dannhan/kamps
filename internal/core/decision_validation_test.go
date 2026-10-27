package core

import (
	"context"
	"strings"
	"testing"

	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/source"
	"github.com/ruyam/kamps/internal/store"
)

func TestDecideRejectsUnknownIDsWithoutChangingDecisions(t *testing.T) {
	cfg := config.Config{StateDir: t.TempDir(), LibraryDir: t.TempDir()}
	service := New(stubSource{snapshot: source.Snapshot{Assignments: []model.Assignment{{ID: "known"}}}}, cfg)
	if _, err := service.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Decide([]string{"known", "missing"}, model.Tracked); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("expected error identifying missing ID, got %v", err)
	}
	if got := service.Snapshot().Assignments[0].Triage; got == model.Tracked {
		t.Fatal("valid assignment changed despite unknown ID")
	}
	state, err := store.Load(cfg.StatePath())
	if err != nil {
		t.Fatal(err)
	}
	if state.Decision("known") == model.Tracked {
		t.Fatal("decision persisted despite unknown ID")
	}
}
