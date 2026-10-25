package tui

import (
	"strings"
	"testing"

	"github.com/ruyam/kamps/internal/model"
)

func TestCommitPreviewAndDrop(t *testing.T) {
	app := demoApp(t)
	app.tab = TabCommit
	setPendingForTest(app, map[string]model.Triage{"K901000": model.Tracked, "K901001": model.Missed})
	app.byID("K901000").Triage = model.Tracked

	closed := strings.Replace(app.taskLine(*app.byID("K901001")), "- [ ]", "- [x]", 1) + " #missed"
	want := app.taskLine(*app.byID("K901000")) + "\n" + closed
	if got := app.commitPreviewText(); got != want {
		t.Fatalf("preview = %q, want %q", got, want)
	}

	// Items sort by action, so the tracked item is first.
	app.commitCursor = 0
	app.dropCommitItem()

	if ok := app.queue.Has("K901000"); ok {
		t.Fatal("drop should remove the focused decision from the queue")
	}
	if app.queue.Len() != 1 {
		t.Fatalf("queue has %d items after drop, want 1", app.queue.Len())
	}
	if got := app.byID("K901000"); got == nil || got.Triage != model.Tracked {
		t.Fatal("drop should not revert the decision itself")
	}
}

func TestCommitEmptyQueueExplainsAndSkipsCopy(t *testing.T) {
	app := demoApp(t)
	app.tab = TabCommit
	setPendingForTest(app, map[string]model.Triage{})

	if got := app.commitPreviewText(); got != "" {
		t.Fatalf("empty queue preview = %q", got)
	}
	if cmd := app.copyCommitPreview(); cmd != nil {
		t.Fatal("copying an empty queue should not emit a command")
	}
	if !strings.Contains(app.toast, "nothing to copy") {
		t.Fatalf("empty copy toast = %q", app.toast)
	}
	body := stripANSI(strings.Join(app.commitBody(80, 10), "\n"))
	if !strings.Contains(body, "no decisions queued") || strings.Contains(body, "mark items in Tasks first") {
		t.Fatalf("empty commit body = %q", body)
	}
}

func TestCommitRowsClipWithEllipsis(t *testing.T) {
	app := demoApp(t)
	app.tab = TabCommit
	id := app.assignments[0].ID
	app.byID(id).Name = strings.Repeat("a very long assignment name ", 10)
	setPendingForTest(app, map[string]model.Triage{id: model.Tracked})

	body := stripANSI(strings.Join(app.commitBody(60, 20), "\n"))
	if !strings.Contains(body, "…") {
		t.Fatalf("clipped commit row has no ellipsis: %q", body)
	}
}

func TestCommitPreviewContainsGroupedExactLinesAndMarkers(t *testing.T) {
	app := demoApp(t)
	app.width, app.height = 100, 30
	app.tab = TabCommit
	setPendingForTest(app, map[string]model.Triage{
		"K901000": model.Tracked,
		"K901001": model.Missed,
		"K901002": model.Ignored,
		"K163827": model.Tracked,
	})
	view := stripANSI(app.View())
	for _, want := range []string{"done (track)", "missed", "ignore", "K901000 · ppl · Progress Report 1", "- [ ] [ppl] Progress Report 1 (K901000)", "will show as OVERDUE", `similar manual task: "Submit ppl Report 5"`} {
		if !strings.Contains(view, want) {
			t.Errorf("commit preview missing %q:\n%s", want, view)
		}
	}
}
