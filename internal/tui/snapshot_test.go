package tui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/source"
)

// TestSnapshot renders each section at a fixed size so the layout can be
// eyeballed with `go test -run TestSnapshot -v ./internal/tui`.
// It is a smoke test: it fails only if rendering panics.
func TestSnapshot(t *testing.T) {
	src := source.NewDemo()
	m := New(src, config.Default(), "test")
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 28})

	ctx := context.Background()
	cs, _ := src.Courses(ctx)
	as, _ := src.Assignments(ctx)
	ms, _ := src.Materials(ctx)
	m.Update(loadedMsg{cs, as, ms})

	m.cursors[SecMaterials] = 4 // make the cursor visible in a non-first row

	for _, s := range []Section{SecCourses, SecAssignments, SecMaterials, SecDownloads} {
		m.section = s
		t.Logf("\n%s\n%s", s, m.View())
	}

	m.helpOn = true
	t.Logf("\nHELP\n%s", m.View())
}

// TestQueueAndProgress exercises the fake download pipeline.
func TestQueueAndProgress(t *testing.T) {
	src := source.NewDemo()
	m := New(src, config.Default(), "test")
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 28})
	ctx := context.Background()
	cs, _ := src.Courses(ctx)
	as, _ := src.Assignments(ctx)
	ms, _ := src.Materials(ctx)
	m.Update(loadedMsg{cs, as, ms})

	m.section = SecMaterials
	m.cursors[SecMaterials] = 1 // Modul Praktikum 6.pdf, not downloaded
	if cmd := m.enqueue(); cmd == nil {
		t.Fatal("expected a download to be queued")
	}
	if len(m.queue) != 1 {
		t.Fatalf("queue = %v, want 1 item", m.queue)
	}
	if m.section != SecDownloads {
		t.Fatalf("expected to jump to Downloads, got %v", m.section)
	}
	for i := 0; i < 40 && len(m.queue) > 0; i++ {
		m.advance()
	}
	if len(m.queue) != 0 {
		t.Fatalf("download did not finish: %v", m.queue)
	}
	if !m.materials[1].Downloaded {
		t.Fatal("material was not marked downloaded")
	}
}
