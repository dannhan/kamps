package plain

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/source"
	"github.com/ruyam/kamps/internal/store"
)

func TestPrintPlainAppliesSavedState(t *testing.T) {
	cfg := config.Config{StateDir: t.TempDir()}
	src := source.NewDemo()
	ctx := context.Background()
	snapshot, err := src.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assignments, materials := snapshot.Assignments, snapshot.Materials

	assignment := assignments[0]
	materialID := ""
	materialName := ""
	for _, mt := range materials {
		if !mt.Downloaded {
			materialID = mt.ID
			materialName = mt.Name
			break
		}
	}
	if materialID == "" {
		t.Fatal("demo has no undownloaded material")
	}

	state := store.New()
	state.SetDecision(assignment.ID, model.Tracked)
	state.SetDownloaded(materialID, true)
	if err := state.Save(cfg.StatePath()); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := PrintPlain(&buf, src, cfg); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	if line := lineContaining(out, assignment.Name); !strings.Contains(line, "tracked") {
		t.Fatalf("plain snapshot ignored the saved decision: %q", line)
	}
	if line := lineContaining(out, materialName); !strings.Contains(line, "local") {
		t.Fatalf("plain snapshot ignored the saved download: %q", line)
	}
}

func lineContaining(text, needle string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, needle) {
			return line
		}
	}
	return ""
}

type failingSource struct{}

func (failingSource) Label() string { return "failing" }

func (failingSource) Load(context.Context) (source.Snapshot, error) {
	return source.Snapshot{}, errors.New("network down")
}

func TestPrintPlainReturnsSourceError(t *testing.T) {
	var buf bytes.Buffer
	err := PrintPlain(&buf, failingSource{}, config.Config{})
	if err == nil {
		t.Fatal("expected a source error")
	}
	if !strings.Contains(err.Error(), "network down") {
		t.Fatalf("error lost the source cause: %v", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("printed a partial snapshot after a load failure: %q", buf.String())
	}
}
