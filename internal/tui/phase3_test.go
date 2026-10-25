package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/model"
)

func TestAutoTriageTracksSubmittedAndUserOverrideWins(t *testing.T) {
	app := demoApp(t)
	app.cfg.AutoTrackSubmitted = true
	id := app.assignments[0].ID
	app.assignments[0].Submission = model.SubmissionSubmitted
	app.assignments[0].Triage = model.Undecided
	app.applyAutoTriage()
	if app.byID(id).Triage != model.Tracked {
		t.Fatalf("auto triage = %v, want tracked", app.byID(id).Triage)
	}
	if app.state.AutoDecisions[id] != model.Tracked {
		t.Fatalf("auto decision = %v", app.state.AutoDecisions[id])
	}

	app.state.SetDecision(id, model.Missed)
	app.applyState()
	app.applyAutoTriage()
	if app.byID(id).Triage != model.Missed {
		t.Fatalf("user override lost: %v", app.byID(id).Triage)
	}
	if _, ok := app.state.AutoDecisions[id]; ok {
		t.Fatal("user decision should clear auto triage")
	}
}

func TestWriteCommitPreviewWritesExactlyPreview(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.md")
	app := newDemoApp(t, config.Config{
		StateDir:     t.TempDir(),
		LibraryDir:   t.TempDir(),
		CommitFile:   path,
		ExportFormat: "markdown",
	})
	app.tab = TabCommit
	id := app.assignments[0].ID
	app.pending = map[string]model.Triage{id: model.Tracked}
	app.byID(id).Triage = model.Tracked

	preview := app.commitPreviewText()
	if preview == "" {
		t.Fatal("empty preview")
	}
	cmd := app.writeCommitPreview()
	if cmd == nil {
		t.Fatal("expected an export command")
	}
	app.Update(cmd())
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != preview {
		t.Fatalf("file = %q, preview = %q", data, preview)
	}
}
