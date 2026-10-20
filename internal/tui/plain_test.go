package tui

import (
	"bytes"
	"context"
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
	assignments, _ := src.Assignments(ctx)
	materials, _ := src.Materials(ctx)

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
	PrintPlain(&buf, src, cfg)
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
