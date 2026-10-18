package tui

import (
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/ruyam/kamps/internal/model"
)

// TestSnapshot renders every screen so the layout can be checked by eye with
// `go test -run TestSnapshot -v ./internal/tui`. It fails only on a panic.
func TestSnapshot(t *testing.T) {
	m := newTestApp(t, 96, 30)

	m.tab, m.layout = TabTasks, LayList
	m.taskCursor = 2
	t.Logf("\n== Tasks · list ==\n%s", m.View())

	m.tab, m.layout = TabTasks, LayDashboard
	m.taskCursor = 1
	t.Logf("\n== Tasks · dashboard ==\n%s", m.View())

	m.tab, m.layout = TabTasks, LayBoard
	m.boardCol = 0
	t.Logf("\n== Tasks · board ==\n%s", m.View())

	// A wizard mid-flow: a few selected, an action chosen, review pending.
	m.tab, m.layout = TabTasks, LayWizard
	items := m.visible()
	for i := 0; i < 3 && i < len(items); i++ {
		m.selected[items[i].ID] = true
	}
	m.wizAction = model.Tracked
	m.wizStep = 1
	t.Logf("\n== Tasks · wizard (action) ==\n%s", m.View())

	m.wizStep = 2
	m.pending = map[string]model.Triage{}
	for i := 0; i < 3 && i < len(items); i++ {
		m.pending[items[i].ID] = model.Tracked
	}
	t.Logf("\n== Tasks · wizard (review) ==\n%s", m.View())

	m.tab, m.layout = TabMaterials, LayList
	m.matCursor = 1
	m.selectedM[m.materials[2].ID] = true
	t.Logf("\n== Materials ==\n%s", m.View())

	m.tab = TabSync
	t.Logf("\n== Sync ==\n%s", m.View())
}

// TestCursorBandIsContinuous is the guard for the selection highlight: on the row
// the keyboard is on, the whole pane interior is one continuous bar — the marker,
// the gaps between the columns and the end-of-row padding alike.
//
// The bug it exists for: joining styled cells with a plain " " punches unstyled
// holes through the band, so the selection comes out striped, with the marker at
// the far left and the background only starting at some later column.
func TestCursorBandIsContinuous(t *testing.T) {
	// The band is a background colour, and lipgloss degrades to no colour at all
	// when stdout is not a terminal — which is the case under `go test`. Pin the
	// profile, or there are no escapes to inspect.
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI256)
	defer lipgloss.SetColorProfile(prev)

	for _, size := range [][2]int{{140, 40}, {100, 30}, {80, 24}, {60, 20}} {
		for _, tc := range []struct {
			name   string
			tab    Tab
			layout Layout
			setup  func(*App)
		}{
			{"tasks/list", TabTasks, LayList, func(m *App) { m.taskCursor = 3 }},
			{"tasks/dashboard", TabTasks, LayDashboard, func(m *App) { m.taskCursor = 3 }},
			{"tasks/board", TabTasks, LayBoard, func(m *App) { m.boardCol = 1; m.boardCursor[1] = 1 }},
			{"tasks/wizard", TabTasks, LayWizard, func(m *App) { m.taskCursor = 2 }},
			{"materials", TabMaterials, LayList, func(m *App) { m.matCursor = 3 }},
		} {
			m := newTestApp(t, size[0], size[1])
			m.tab, m.layout = tc.tab, tc.layout
			tc.setup(m)

			line, ok := cursorLine(m.View())
			if !ok {
				t.Fatalf("%s at %v: no cursor marker in the view", tc.name, size)
			}
			b := scanCursorBand(line)
			switch {
			case !b.banded:
				t.Fatalf("%s at %v: the cursor row carries no background at all\n  %q",
					tc.name, size, b.text)
			case !b.spansInterior():
				t.Errorf("%s at %v: the band covers columns %d..%d but the pane's interior is %d..%d — pad the row to the full inner width\n  %q",
					tc.name, size, b.first, b.last, b.lo, b.hi, b.text)
			default:
				if gaps := b.gaps(); len(gaps) > 0 {
					t.Errorf("%s at %v: %d unstyled column(s) inside the cursor band, at %v\n  %q",
						tc.name, size, len(gaps), gaps, b.text)
				}
			}
		}
	}
}

// cursorLine returns the first rendered line carrying the cursor marker.
func cursorLine(view string) (string, bool) {
	for _, ln := range strings.Split(view, "\n") {
		if strings.Contains(ln, "▌") {
			return ln, true
		}
	}
	return "", false
}

// cursorBand is one cursor row measured against the pane it sits in.
type cursorBand struct {
	lo, hi int    // the pane's interior columns, inclusive
	first  int    // first interior column carrying the band, -1 if none
	last   int    // last interior column carrying the band
	banded bool   // the interior carries a background at all
	mask   []bool // mask[i] is true when column i was drawn with a background
	text   string // the line with its escapes stripped
}

// spansInterior reports whether the band reaches both of the pane's borders, i.e.
// the row was padded to the full inner width instead of stopping short.
func (b cursorBand) spansInterior() bool {
	return b.banded && b.first == b.lo && b.last == b.hi
}

// gaps lists the columns between the two ends of the band that are *not* drawn
// with its background: the stripes that are the bug.
func (b cursorBand) gaps() []int {
	if !b.banded {
		return nil
	}
	var out []int
	for i := b.first; i <= b.last; i++ {
		if !b.mask[i] {
			out = append(out, i)
		}
	}
	return out
}

// scanCursorBand measures a rendered line. The pane the cursor is in is the span
// between two border glyphs, which is exactly the span the row has to fill; the
// other pane on the line and the join between the two panes are somebody else's.
func scanCursorBand(line string) cursorBand {
	mask, plain := ansiBackgroundMask(line)
	b := cursorBand{mask: mask, text: string(plain), lo: -1, hi: -1, first: -1, last: -1}

	from := 0
	for i, r := range plain {
		if r != '│' {
			continue
		}
		if from > 0 && slices.Contains(plain[from:i], '▌') {
			b.lo, b.hi = from, i-1
			break
		}
		from = i + 1
	}
	if b.lo < 0 || b.hi < b.lo {
		return b
	}

	for i := b.lo; i <= b.hi && i < len(mask); i++ {
		if mask[i] {
			if b.first < 0 {
				b.first = i
			}
			b.last = i
		}
	}
	b.banded = b.first >= 0
	return b
}

// ansiBackgroundMask walks styled text and reports, per rune, whether that rune
// was drawn with a background colour set. plain is the text with the escapes
// removed, aligned 1:1 with the mask.
func ansiBackgroundMask(s string) (mask []bool, plain []rune) {
	bg := false
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			j := strings.IndexByte(s[i:], 'm')
			if j < 0 {
				break
			}
			bg = applySGR(s[i+2:i+j], bg)
			i += j + 1
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		mask = append(mask, bg)
		plain = append(plain, r)
		i += size
	}
	return mask, plain
}

// applySGR folds one SGR parameter list into the current background state. It has
// to understand the extended forms (48;5;n, 48;2;r;g;b) because a naive scan for
// "0" inside them misreads the colour index as a reset.
func applySGR(params string, bg bool) bool {
	ps := strings.Split(params, ";")
	for i := 0; i < len(ps); i++ {
		n, err := strconv.Atoi(ps[i])
		if err != nil {
			continue
		}
		switch {
		case n == 38 || n == 48:
			// Extended colour: skip the "5;n" or "2;r;g;b" tail.
			if i+1 < len(ps) && ps[i+1] == "5" {
				i += 2
			} else if i+1 < len(ps) && ps[i+1] == "2" {
				i += 4
			}
			if n == 48 {
				bg = true
			}
		case n == 0 || n == 49:
			bg = false
		case n >= 40 && n <= 47, n >= 100 && n <= 107:
			bg = true
		}
	}
	return bg
}
