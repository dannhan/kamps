package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ruyam/kamps/internal/model"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	state := New()
	state.SetDecision("K901000", model.Tracked)
	state.SetDecision("K901001", model.Missed)
	state.SetDownloaded("M165401", true)
	if err := state.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := loaded.Decision("K901000"); got != model.Tracked {
		t.Fatalf("decision = %v, want tracked", got)
	}
	if got := loaded.Decision("K901001"); got != model.Missed {
		t.Fatalf("decision = %v, want missed", got)
	}
	if _, ok := loaded.Downloads["M165401"]; !ok {
		t.Fatal("download flag did not survive the round trip")
	}
	if loaded.Version != version {
		t.Fatalf("version = %d, want %d", loaded.Version, version)
	}
}

func TestUndecidedDeletesTheEntry(t *testing.T) {
	state := New()
	state.SetDecision("K1", model.Tracked)
	state.SetDecision("K1", model.Undecided)
	if _, ok := state.Decisions["K1"]; ok {
		t.Fatal("undecided should not be stored")
	}
	if got := state.Decision("K1"); got != model.Undecided {
		t.Fatalf("decision = %v, want undecided", got)
	}
}

func TestLoadMissingFileStartsEmpty(t *testing.T) {
	state, err := Load(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	if len(state.Decisions) != 0 || len(state.Downloads) != 0 {
		t.Fatalf("missing file produced state: %+v", state)
	}
}

func TestLoadCorruptFileReportsAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	state, err := Load(path)
	if err == nil {
		t.Fatal("corrupt state should report an error")
	}
	if len(state.Decisions) != 0 {
		t.Fatalf("corrupt state should fall back empty, got %+v", state)
	}
}

func TestLoadDropsUnknownDecisions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	body := `{"version":1,"decisions":{"K1":1,"K2":99,"K3":0}}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	state, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := state.Decision("K1"); got != model.Tracked {
		t.Fatalf("valid decision = %v, want tracked", got)
	}
	for _, id := range []string{"K2", "K3"} {
		if _, ok := state.Decisions[id]; ok {
			t.Fatalf("invalid decision %s survived the load", id)
		}
	}
}

func TestSaveCreatesMissingDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deep", "state", "state.json")
	state := New()
	state.SetDecision("K1", model.Ignored)
	if err := state.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := loaded.Decision("K1"); got != model.Ignored {
		t.Fatalf("decision = %v, want ignored", got)
	}
}
