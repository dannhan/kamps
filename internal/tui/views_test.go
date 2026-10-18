package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/source"
)

func newTestApp(t *testing.T, width, height int) *App {
	t.Helper()
	src := source.NewDemo()
	m := New(src, config.Default(), "test")
	m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	ctx := context.Background()
	cs, _ := src.Courses(ctx)
	as, _ := src.Assignments(ctx)
	ms, _ := src.Materials(ctx)
	m.Update(loadedMsg{cs, as, ms})
	return m
}

func press(m *App, msg tea.KeyMsg) { m.Update(msg) }

func runeKey(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

// TestViewAlignment is the guard for all the hand-drawn box maths: every
// rendered line must be exactly boxWidth wide, at every size and on every
// screen. A stray character shows up as a broken border otherwise.
func TestViewAlignment(t *testing.T) {
	for _, size := range [][2]int{{100, 30}, {80, 24}, {72, 18}} {
		m := newTestApp(t, size[0], size[1])
		want := m.boxWidth()
		wantH := m.boxHeight()

		for _, tab := range []Tab{TabTasks, TabMaterials, TabSync} {
			for _, lay := range []Layout{LayList, LayDashboard, LayBoard, LayWizard} {
				m.tab, m.layout = tab, lay
				lines := strings.Split(m.View(), "\n")
				if len(lines) != wantH {
					t.Errorf("size %v tab %v layout %v: %d lines, want %d", size, tab, lay, len(lines), wantH)
				}
				for i, ln := range lines {
					if got := lipgloss.Width(ln); got != want {
						t.Errorf("size %v tab %v layout %v: line %d width %d, want %d\n  %q",
							size, tab, lay, i, got, want, stripANSI(ln))
					}
				}
			}
		}
	}
}

func TestWizardRendersEveryStep(t *testing.T) {
	m := newTestApp(t, 96, 30)
	m.tab, m.layout = TabTasks, LayWizard
	for step := 0; step <= 3; step++ {
		m.wizStep = step
		if s := m.View(); !strings.Contains(s, "kamps") {
			t.Fatalf("step %d rendered nothing", step)
		}
	}
}

func TestTriageQueuesForCommit(t *testing.T) {
	m := newTestApp(t, 96, 30)
	m.tab = TabTasks

	id := m.cursorAssignmentID()
	if id == "" {
		t.Fatal("no cursor assignment")
	}
	press(m, runeKey('t'))

	a := m.byID(id)
	if a == nil || a.Triage != model.Tracked {
		t.Fatalf("assignment %s was not tracked", id)
	}
	if len(m.pending) != 1 {
		t.Fatalf("pending = %d, want 1", len(m.pending))
	}
}

func TestWizardCommitClearsQueue(t *testing.T) {
	m := newTestApp(t, 96, 30)
	m.tab, m.layout = TabTasks, LayList
	press(m, runeKey('t')) // queue one change
	if len(m.pending) != 1 {
		t.Fatalf("pending = %d, want 1", len(m.pending))
	}

	press(m, runeKey('c')) // open commit
	if m.layout != LayWizard || m.wizStep != 2 {
		t.Fatalf("commit should open the wizard at review, got layout=%v step=%d", m.layout, m.wizStep)
	}
	press(m, tea.KeyMsg{Type: tea.KeyEnter}) // apply
	if len(m.pending) != 0 {
		t.Fatalf("pending = %d after commit, want 0", len(m.pending))
	}
	if m.wizStep != 3 {
		t.Fatalf("step = %d after commit, want 3", m.wizStep)
	}
}

func TestDownloadMarksMaterials(t *testing.T) {
	m := newTestApp(t, 96, 30)
	m.tab = TabMaterials

	idx := -1
	for i, mt := range m.materials {
		if !mt.Downloaded {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Skip("demo has nothing left to download")
	}
	m.matCursor = idx
	press(m, runeKey('d'))

	if !m.materials[idx].Downloaded {
		t.Fatalf("material %s was not marked downloaded", m.materials[idx].ID)
	}
}

func TestDashboardCursorIsActionable(t *testing.T) {
	m := newTestApp(t, 96, 30)
	m.tab, m.layout = TabTasks, LayDashboard

	list := m.focusedList()
	if len(list) == 0 {
		t.Fatal("dashboard today panel is empty")
	}
	for _, a := range list {
		if a.Triage != model.Undecided {
			t.Fatalf("dashboard shows a decided item: %s is %s", a.ID, a.Triage)
		}
	}
	id := m.cursorAssignmentID()
	if id == "" {
		t.Fatal("dashboard has no cursor item")
	}

	press(m, runeKey('t'))

	if a := m.byID(id); a == nil || a.Triage != model.Tracked {
		t.Fatalf("dashboard triage key did not track %s", id)
	}
	for _, a := range m.dashShown() {
		if a.ID == id {
			t.Fatal("tracked item is still in the dashboard today panel")
		}
	}
	if len(m.pending) != 1 {
		t.Fatalf("pending = %d, want 1", len(m.pending))
	}
}

func TestViewCycleSkipsBoardAndWizard(t *testing.T) {
	m := newTestApp(t, 96, 30)
	m.tab, m.layout = TabTasks, LayDashboard

	seen := []Layout{}
	for i := 0; i < 4; i++ {
		seen = append(seen, m.layout)
		press(m, runeKey('v'))
	}
	if seen[0] != LayDashboard || seen[1] != LayList || seen[2] != LayDashboard || seen[3] != LayList {
		t.Fatalf("v cycle = %v, want dashboard/list alternating", seen)
	}
	for _, l := range seen {
		if l == LayBoard || l == LayWizard {
			t.Fatalf("v reached %v, which should not be in the cycle", l)
		}
	}
}

func TestCommitReturnsToPreviousView(t *testing.T) {
	m := newTestApp(t, 96, 30)
	m.tab, m.layout = TabTasks, LayDashboard
	press(m, runeKey('t')) // queue a change

	press(m, runeKey('c'))
	if m.layout != LayWizard {
		t.Fatalf("c did not open the wizard, layout=%v", m.layout)
	}
	for i := 0; i < 5 && m.layout == LayWizard; i++ {
		press(m, tea.KeyMsg{Type: tea.KeyEsc})
	}
	if m.layout != LayDashboard {
		t.Fatalf("esc returned to %v, want dashboard", m.layout)
	}
}

func TestNoAnimSyncFinishesImmediately(t *testing.T) {
	src := source.NewDemo()
	cfg := config.Default()
	cfg.NoAnim = true
	m := New(src, cfg, "test")
	m.tab = TabSync
	m.startSync()
	if !m.syncing {
		t.Fatal("sync did not start")
	}
	m.advance() // with --no-anim a single tick completes the sync
	if m.syncing {
		t.Fatal("sync still running after one tick with NoAnim")
	}
	if m.toast == "" {
		t.Fatal("no completion toast")
	}
}

func TestFilterCycles(t *testing.T) {
	m := newTestApp(t, 96, 30)
	start := m.filter
	press(m, runeKey('/'))
	if m.filter == start {
		t.Fatal("filter did not change")
	}
	if got, want := len(m.visible()), len(m.assignments); m.filter == "all" && got != want {
		t.Fatalf("all filter = %d items, want %d", got, want)
	}
}
