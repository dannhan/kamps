package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/ruyam/kamps/internal/model"
)

// Every tree row has the same structural columns:
//
//	gutter | guides | expander | mark | space | glyph | prefix | title | meta
//
// The expander is always two cells (caret + space, or two blanks), so a row
// with a container and a leaf title start in the same column. The glyph is a
// one-cell state/triage flag plus a space; reserveGlyph keeps that column even
// when a particular row has no glyph.
const (
	treeRowRightPad    = 2
	courseRuleMinWidth = 60
	dueDaysWidth       = 5
	dueColumnWidth     = 1 + dueDaysWidth // "!" slot plus right-aligned days
)

type treeRowSpec struct {
	nodes []node
	index int
	width int

	cursor   bool
	selected bool
	partial  bool

	expandable bool
	open       bool

	glyph        string
	glyphStyle   lipgloss.Style
	reserveGlyph bool

	prefix func(rowHighlight) string
	title  func(rowHighlight) string
	meta   func(rowHighlight) string

	// blank renders a full-width empty row for a non-selectable spacer.
	blank bool
	// rule fills the gap between the title and meta with a dim horizontal
	// rule. Course headers set it when the row is wide enough.
	rule bool
}

// treeRow is the single renderer for Tasks and Materials rows. It owns the
// connector, expander, mark, glyph, truncation, and right alignment so callers
// never concatenate structural columns themselves.
func (m *App) treeRow(spec treeRowSpec) string {
	if spec.blank {
		return strings.Repeat(" ", spec.width)
	}
	h := rowHighlight{cursor: spec.cursor, mark: spec.selected, partial: spec.partial}
	background := h.style

	head := h.gutter() + background(stGuide).Render(treePrefix(spec.nodes, spec.index))
	if spec.expandable {
		head += background(stFaint).Render(caretGlyph(spec.open)) + background(stRow).Render(" ")
	} else {
		head += background(stRow).Render("  ")
	}
	head += h.markCell() + background(stRow).Render(" ")

	if spec.reserveGlyph || spec.glyph != "" {
		if spec.glyph != "" {
			head += background(spec.glyphStyle).Render(spec.glyph) + background(stRow).Render(" ")
		} else {
			head += background(stRow).Render("  ")
		}
	}
	if spec.prefix != nil {
		head += spec.prefix(h)
	}

	meta := ""
	if spec.meta != nil {
		meta = spec.meta(h)
	}
	metaWidth := lipgloss.Width(meta)
	gap := 0
	if meta != "" {
		gap = 1
	}
	available := spec.width - treeRowRightPad - lipgloss.Width(head) - metaWidth - gap
	if available < 0 {
		available = 0
	}

	title := ""
	if spec.title != nil && available > 0 {
		title = ansi.Truncate(spec.title(h), available, "…")
	}
	row := head + title
	if meta != "" {
		fill := spec.width - treeRowRightPad - lipgloss.Width(row) - metaWidth
		if fill < 1 {
			fill = 1
		}
		if spec.rule {
			row += background(stGuide).Render(strings.Repeat("─", fill))
		} else {
			row += background(stRow).Render(strings.Repeat(" ", fill))
		}
		row += meta
	} else if spec.rule {
		fill := spec.width - treeRowRightPad - lipgloss.Width(row)
		if fill > 0 {
			row += background(stGuide).Render(strings.Repeat("─", fill))
		}
	}

	if width := lipgloss.Width(row); width > spec.width {
		row = ansiTruncate(row, spec.width)
	}
	if pad := spec.width - lipgloss.Width(row); pad > 0 {
		padding := strings.Repeat(" ", pad)
		if h.band() {
			padding = background(stRow).Render(padding)
		}
		row += padding
	}
	return row
}

// Due-date buckets. The last day in each bucket is a threshold; 0 and the
// negative range are both named so the mapping has no inline magic numbers.
const (
	dueOverdueMaxDays = -1 // days <= -1 is overdue
	dueSoonMaxDays    = 7  // 0..7 days
	dueNormalMaxDays  = 29 // 8..29 days; 30+ is far
)

type dueTone int

const (
	dueToneNone dueTone = iota
	dueToneOverdue
	dueToneSoon
	dueToneNormal
	dueToneFar
)

// dueToneForDays maps a relative day count to the semantic urgency slot.
func dueToneForDays(days int, hasDue bool) dueTone {
	if !hasDue {
		return dueToneNone
	}
	switch {
	case days <= dueOverdueMaxDays:
		return dueToneOverdue
	case days <= dueSoonMaxDays:
		return dueToneSoon
	case days <= dueNormalMaxDays:
		return dueToneNormal
	default:
		return dueToneFar
	}
}

func dueStyleFor(tone dueTone) lipgloss.Style {
	switch tone {
	case dueToneOverdue:
		return stDueOverdue
	case dueToneSoon:
		return stDueSoon
	case dueToneNormal:
		return stDueNormal
	case dueToneFar:
		return stDueFar
	default:
		return stDueNone
	}
}

func dueDays(a model.Assignment, now time.Time) int {
	if !a.HasDue {
		return 0
	}
	dueDay := dateOnly(a.Due)
	today := dateOnly(now)
	return int(dueDay.Sub(today).Hours() / 24)
}

func dueAge(a model.Assignment, now time.Time) string {
	if !a.HasDue {
		return "—"
	}
	days := dueDays(a, now)
	if days < 0 {
		return fmt.Sprintf("−%dd", -days)
	}
	return fmt.Sprintf("+%dd", days)
}

// dueCell is a fixed-width right-aligned due column. The first cell is the
// "!" flag slot; the remaining cells hold the relative day count, right
// aligned, so the column's right edge never moves.
func dueCell(a model.Assignment, now time.Time) string {
	flag := " "
	if a.HasDue && a.Due.Before(now) {
		flag = "!"
	}
	value := "—"
	if a.HasDue {
		value = dueAge(a, now)
	}
	value = ansi.Truncate(value, dueDaysWidth, "…")
	if width := lipgloss.Width(value); width < dueDaysWidth {
		value = strings.Repeat(" ", dueDaysWidth-width) + value
	}
	return flag + value
}

// taskDisplayName drops the LMS activity-type suffix that Moodle folds into the
// accessible name. The full name stays in the detail pane and the export.
func taskDisplayName(name string) string {
	trimmed := strings.TrimSpace(name)
	lower := strings.ToLower(trimmed)
	for _, suffix := range []string{" assignment", " quiz"} {
		if !strings.HasSuffix(lower, suffix) {
			continue
		}
		display := strings.TrimSpace(trimmed[:len(trimmed)-len(suffix)])
		if display != "" {
			return display
		}
	}
	return trimmed
}
