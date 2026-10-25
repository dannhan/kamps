package cache

import (
	"path/filepath"
	"testing"

	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/source"
)

func TestSaveLoadRoundTripAndSiteCheck(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.json")
	snapshot := source.Snapshot{Assignments: []model.Assignment{{ID: "K1"}}}
	if err := Save(path, "https://lms.test", snapshot); err != nil {
		t.Fatal(err)
	}
	loaded, savedAt, err := Load(path, "https://lms.test")
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Assignments) != 1 || savedAt.IsZero() {
		t.Fatalf("loaded = %+v savedAt = %v", loaded, savedAt)
	}
	if _, _, err := Load(path, "https://other.test"); err == nil {
		t.Fatal("cache from another site should be rejected")
	}
}
