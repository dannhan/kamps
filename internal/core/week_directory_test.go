package core

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/model"
)

func TestWeekDirectoriesUsePaddedNamesAndReuseExisting(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{LibraryDir: dir}
	courses := []model.Course{{ID: 1, Code: "CS", WorkspacePath: filepath.Join(dir, "course")}}
	for week := 0; week <= 16; week++ {
		material := model.Material{CourseID: 1, Section: fmt.Sprintf("week-%d", week), Name: "notes.pdf"}
		want := filepath.Join(dir, "course", fmt.Sprintf("week-%02d", week), "notes.pdf")
		if got := Destination(cfg, courses, material); got != want {
			t.Fatalf("week %d: got %q, want %q", week, got, want)
		}
	}
	legacy := filepath.Join(dir, "course", "week-1")
	if err := os.MkdirAll(legacy, 0755); err != nil {
		t.Fatal(err)
	}
	got := Destination(cfg, courses, model.Material{CourseID: 1, Section: "week-1", Name: "notes.pdf"})
	if want := filepath.Join(legacy, "notes.pdf"); got != want {
		t.Fatalf("existing directory: got %q, want %q", got, want)
	}
	fallback := Destination(cfg, []model.Course{{ID: 2, Code: "CS"}}, model.Material{CourseID: 2, Section: "week-2", Name: "notes.pdf"})
	if want := filepath.Join(dir, "CS", "week-02", "notes.pdf"); fallback != want {
		t.Fatalf("fallback: got %q, want %q", fallback, want)
	}
}
