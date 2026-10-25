package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ruyam/kamps/internal/model"
)

func TestLoadMigratesVersionOne(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	body := `{"version":1,"decisions":{"K1":1},"downloads":{"M1":true}}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	state, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if state.Version != version {
		t.Fatalf("version = %d, want %d", state.Version, version)
	}
	if _, ok := state.Downloads["M1"]; !ok {
		t.Fatal("v1 download flag was not migrated")
	}
	if got := state.Decision("K1"); got != model.Tracked {
		t.Fatalf("decision = %v", got)
	}
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatalf("backup was not written: %v", err)
	}
}

func TestAutoDecisionDoesNotOverrideUser(t *testing.T) {
	state := New()
	state.SetDecision("K1", model.Missed)
	state.SetAutoDecision("K1", model.Tracked)
	if got := state.AutoDecisions["K1"]; got != model.Undecided {
		t.Fatalf("auto decision = %v, want undecided", got)
	}
	assignments := []model.Assignment{{ID: "K1"}}
	state.Apply(assignments, nil)
	if assignments[0].Triage != model.Missed {
		t.Fatalf("triage = %v, want missed", assignments[0].Triage)
	}
}

func TestUpdateMergesAcrossWriters(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	lock := filepath.Join(dir, "state.lock")
	if err := Update(path, lock, func(s *State) { s.SetDecision("K1", model.Tracked) }); err != nil {
		t.Fatal(err)
	}
	if err := Update(path, lock, func(s *State) { s.SetDecision("K2", model.Ignored) }); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Decision("K1") != model.Tracked || loaded.Decision("K2") != model.Ignored {
		t.Fatalf("merged decisions = %+v", loaded.Decisions)
	}
}

func TestSaveUsesPrivateMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	state := New()
	if err := state.Save(path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}
}
