package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestMockupAlignment guards the hand-drawn box maths: every line of every
// mockup must have exactly the same visible width, or the borders will not
// line up in a terminal.
func TestMockupAlignment(t *testing.T) {
	for _, m := range Mockups() {
		lines := strings.Split(m.Body(), "\n")
		if len(lines) == 0 {
			t.Fatalf("mockup %s rendered nothing", m.Key)
		}
		w := lipgloss.Width(lines[0])
		if w != mockW {
			t.Errorf("mockup %s: first line width = %d, want %d", m.Key, w, mockW)
		}
		for i, ln := range lines {
			if got := lipgloss.Width(ln); got != w {
				t.Errorf("mockup %s: line %d width = %d, want %d\n  %q", m.Key, i, got, w, stripANSI(ln))
			}
		}
	}
}

// stripANSI makes the failure output readable.
func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		switch {
		case r == 0x1b:
			inEsc = true
		case inEsc && r == 'm':
			inEsc = false
		case !inEsc:
			b.WriteRune(r)
		}
	}
	return b.String()
}
