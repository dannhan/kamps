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

type stubSource struct{ snapshot source.Snapshot }

func TestCourseResolvesNumericID(t *testing.T) {
	s := New(stubSource{snapshot: source.Snapshot{Courses: []model.Course{{ID: 42, Code: "CS"}}}}, config.Config{StateDir: t.TempDir(), LibraryDir: t.TempDir()})
	if _, err := s.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	if course, ok := s.Course("42"); !ok || course.ID != 42 {
		t.Fatalf("numeric course lookup = %+v, %v", course, ok)
	}
}

func (s stubSource) Label() string { return "stub" }

func (s stubSource) Load(context.Context) (source.Snapshot, error) {
	return s.snapshot, nil
}

func TestLoadPipelineAppliesStateAutoTriageAndScan(t *testing.T) {
	stateDir := t.TempDir()
	libraryDir := t.TempDir()
	cfg := config.Config{
		StateDir:           stateDir,
		LibraryDir:         libraryDir,
		AutoTrackSubmitted: true,
	}
	snap := source.Snapshot{
		Courses: []model.Course{{ID: 1, Code: "C1", Short: "C1", Name: "Course One"}},
		Assignments: []model.Assignment{
			{ID: "K1", CourseID: 1, CourseCode: "C1", Name: "submitted", Submission: model.SubmissionSubmitted},
			{ID: "K2", CourseID: 1, CourseCode: "C1", Name: "user decision", Submission: model.SubmissionSubmitted},
		},
		Materials: []model.Material{
			{ID: "M10", CourseID: 1, CourseCode: "C1", Name: "ten.pdf", Section: "week-10"},
			{ID: "M2", CourseID: 1, CourseCode: "C1", Name: "two.pdf", Section: "week-2"},
			{ID: "M1", CourseID: 1, CourseCode: "C1", Name: "one.pdf", Section: "week-1"},
		},
	}
	state := store.New()
	state.SetDecision("K2", model.Missed)
	if err := state.Save(cfg.StatePath()); err != nil {
		t.Fatal(err)
	}
	dest := Destination(cfg, snap.Courses, snap.Materials[2])
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(libraryDir, "stray.pdf"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	service := New(stubSource{snapshot: snap}, cfg)
	got, err := service.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Assignments[0].Triage != model.Tracked {
		t.Fatalf("auto triage = %v", got.Assignments[0].Triage)
	}
	if got.Assignments[1].Triage != model.Missed {
		t.Fatalf("user decision = %v", got.Assignments[1].Triage)
	}
	if !got.Materials[0].Downloaded {
		t.Fatal("scanned material should be downloaded")
	}
	if got.Materials[0].Section != "week-1" || got.Materials[1].Section != "week-2" || got.Materials[2].Section != "week-10" {
		t.Fatalf("material order = %q, %q, %q", got.Materials[0].Section, got.Materials[1].Section, got.Materials[2].Section)
	}
	if got.Scan.Found != 1 || got.Scan.Unknown != 1 {
		t.Fatalf("scan = %+v", got.Scan)
	}
	if !service.State().Downloaded("M1") {
		t.Fatal("download record was not persisted")
	}

	cached, ok := service.LoadCached()
	if !ok {
		t.Fatal("cache was not saved")
	}
	if !cached.FromCache || cached.Assignments[0].Triage != model.Tracked || !cached.Materials[0].Downloaded {
		t.Fatalf("cached pipeline = %+v", cached)
	}
}

func TestSectionLess(t *testing.T) {
	if !SectionLess("week-1", "week-2") || !SectionLess("week-2", "week-10") {
		t.Fatal("week sections are not in numeric order")
	}
	if SectionLabel("week-01") != "week-1" || SectionLabel("") != "week-0" {
		t.Fatal("week labels are not normalized")
	}
}
