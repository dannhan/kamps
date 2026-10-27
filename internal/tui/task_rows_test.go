package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ruyam/kamps/internal/model"
)

func TestFooterShowsToasts(t *testing.T) {
	app := demoApp(t)
	app.width, app.height = 100, 24
	app.toast = "copied 2 lines"
	app.toastTicks = 20
	for _, tab := range []Tab{TabTasks, TabMaterials} {
		app.tab = tab
		view := stripANSI(app.View())
		if !strings.Contains(view, "copied 2 lines") {
			t.Fatalf("%s footer hid the toast: %q", tab, view)
		}
	}
}

func TestTaskQueueLabels(t *testing.T) {
	app := demoApp(t)
	if got := stripANSI(app.taskQueueLabel()); got != "" {
		t.Fatalf("empty queue label = %q", got)
	}
	app.queue.Set("a", model.Tracked)
	app.queue.Set("b", model.Tracked)
	app.queue.Set("c", model.Missed)
	if got := stripANSI(app.taskQueueLabel()); got != "queue: 2 track · 1 missed" {
		t.Fatalf("populated queue label = %q", got)
	}
}

func TestTaskTotalsShowEveryStateCount(t *testing.T) {
	app := demoApp(t)
	totals := tallyTriage(app.assignments)
	legend := stripANSI(app.taskTotals(200))
	for _, want := range []string{
		fmt.Sprintf("○ new %d", totals[0]),
		fmt.Sprintf("◐ tracked %d", totals[1]),
		fmt.Sprintf("◆ ignored %d", totals[3]),
		fmt.Sprintf("× missed %d", totals[4]),
	} {
		if !strings.Contains(legend, want) {
			t.Fatalf("legend %q missing %q", legend, want)
		}
	}
	if narrow := stripANSI(app.taskTotals(30)); strings.Contains(narrow, "new") || !strings.Contains(narrow, "○ ") {
		t.Fatalf("narrow legend should fall back to glyph and count: %q", narrow)
	}
}

func TestDueAgeUsesCalendarDays(t *testing.T) {
	now := time.Date(2026, 10, 18, 23, 30, 0, 0, time.UTC)
	if got := dueAge(model.Assignment{HasDue: true, Due: time.Date(2026, 10, 17, 23, 55, 0, 0, time.UTC)}, now); got != "−1d" {
		t.Fatalf("past due age = %q, want −1d", got)
	}
	if got := dueAge(model.Assignment{HasDue: true, Due: time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)}, now); got != "+2d" {
		t.Fatalf("upcoming age = %q, want +2d", got)
	}
}
