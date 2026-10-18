package tui

import (
	"testing"
	"time"

	"github.com/ruyam/kamps/internal/model"
)

func TestHumanSize(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "—"},
		{512, "512 B"},
		{2048, "2.0 KB"},
		{1572864, "1.5 MB"},
	}
	for _, c := range cases {
		if got := humanSize(c.in); got != c.want {
			t.Errorf("humanSize(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestWindowRange(t *testing.T) {
	// Fits entirely.
	if s, e := windowRange(5, 2, 10); s != 0 || e != 5 {
		t.Errorf("got %d,%d want 0,5", s, e)
	}
	// Cursor keeps itself visible near the end.
	if s, e := windowRange(100, 99, 10); e-s != 10 || e != 100 {
		t.Errorf("got %d,%d want a 10-wide window ending at 100", s, e)
	}
	// Cursor near the start.
	if s, _ := windowRange(100, 0, 10); s != 0 {
		t.Errorf("start = %d, want 0", s)
	}
}

func TestLessAssignment(t *testing.T) {
	now := time.Now()
	dated := model.Assignment{ID: "a", HasDue: true, Due: now}
	undated := model.Assignment{ID: "b", HasDue: false}
	if !lessAssignment(dated, undated) {
		t.Error("dated assignments should sort before undated ones")
	}
	soon := model.Assignment{ID: "c", HasDue: true, Due: now.Add(time.Hour)}
	if !lessAssignment(dated, soon) {
		t.Error("earlier due dates should sort first")
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("hello", 10); got != "hello" {
		t.Errorf("got %q", got)
	}
	if got := truncate("hello world", 6); got != "hello…" {
		t.Errorf("got %q", got)
	}
}
