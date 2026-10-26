package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// rowHighlight holds the independent decorations a list row can carry: the
// cursor (where the keyboard is), a mark (what space has selected), and a
// partial mark (a container where only some children are selected).
//
// They are deliberately different. The cursor uses the Surface band and an
// accent gutter; marks use the Marked band and a square. Every band is paired
// with a glyph so focus and selection survive monochrome.
type rowHighlight struct {
	cursor  bool
	mark    bool
	partial bool
}

func (h rowHighlight) band() bool { return h.cursor || h.mark }

// style paints the band's background onto a cell style. Every cell of a row
// has to go through this, otherwise the gaps stay uncoloured and the highlight
// comes out striped.
func (h rowHighlight) style(s lipgloss.Style) lipgloss.Style {
	switch {
	case h.cursor:
		return bandStyle(s, themeTokens.Surface)
	case h.mark:
		return bandStyle(s, themeTokens.Marked)
	default:
		return s
	}
}

// primaryStyle is the row's main label: the band plus bold on the cursor row.
func (h rowHighlight) primaryStyle() lipgloss.Style {
	style := h.style(stRow)
	if h.cursor {
		style = style.Bold(true)
	}
	return style
}

// gutter is the row's left-edge cell: an accent bar on the cursor row, a blank
// (banded, when marked) cell otherwise, so every row starts in the same column.
func (h rowHighlight) gutter() string {
	if h.cursor {
		return h.style(stCursor).Render("▌")
	}
	return h.style(stRow).Render(" ")
}

// gap is one banded column between two cells, and blanks is n banded columns
// at the end of a row. Every separator has to carry the band, otherwise plain
// spaces punch uncoloured holes through the highlight.
func (h rowHighlight) gap() string { return h.style(stRow).Render(" ") }

func (h rowHighlight) blanks(n int) string {
	if n <= 0 {
		return ""
	}
	return h.style(stRow).Render(strings.Repeat(" ", n))
}

const (
	markFull    = "▪"
	markPartial = "▫"
)

// caretGlyph is the expand/collapse indicator drawn before a parent row's
// name. Open containers show a down caret, folded ones a right caret.
func caretGlyph(open bool) string {
	if open {
		return "▾"
	}
	return "▸"
}

// markCell renders the mark column: blank, a filled accent square when the row
// (or its whole group) is selected, and a hollow square when only part of a
// group is selected.
func (h rowHighlight) markCell() string {
	switch {
	case h.mark:
		return h.style(stSelMark).Render(markFull)
	case h.partial:
		return h.style(stSelMark).Render(markPartial)
	default:
		return h.style(stRow).Render(" ")
	}
}

// paneRow is the shared row renderer for lists that do not paint cell by cell:
// Commit groups and command-palette results. It uses the same cursor language
// as the trees.
func paneRow(content string, width int, state rowHighlight) string {
	prefix := state.style(stRow).Render(" ")
	if state.cursor {
		prefix = state.style(stCursor).Render("▌")
	}
	available := maxInt(0, width-lipgloss.Width(prefix))
	content = ansiTruncate(content, available)
	padding := strings.Repeat(" ", maxInt(0, available-lipgloss.Width(content)))
	if state.band() {
		content = state.style(lipgloss.NewStyle()).Render(content)
		padding = state.style(stRow).Render(padding)
	}
	return prefix + content + padding
}
