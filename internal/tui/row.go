package tui

import "github.com/charmbracelet/lipgloss"

// rowHighlight holds the two independent decorations a list row can carry: the
// cursor (where the keyboard is) and the mark (what space has selected).
//
// They are deliberately different. The cursor uses the strong Surface band and
// an accent gutter; marks use the softer Marked band and a filled square. That
// way a heavily marked tree still shows where the cursor is, and an unmarked
// row never looks selected.
type rowHighlight struct {
	cursor bool
	mark   bool
}

func (h rowHighlight) band() bool { return h.cursor || h.mark }

// style paints the band's background onto a cell style. Every cell of a row
// has to go through this, otherwise the gaps stay uncoloured and the highlight
// comes out striped.
func (h rowHighlight) style(s lipgloss.Style) lipgloss.Style {
	switch {
	case h.cursor:
		return s.Background(themeTokens.Surface)
	case h.mark:
		return s.Background(themeTokens.Marked)
	default:
		return s
	}
}

// gutter is the row's left-edge cell: an accent bar on the cursor row, a blank
// (banded, when marked) cell otherwise, so every row starts in the same column.
func (h rowHighlight) gutter() string {
	if h.cursor {
		return h.style(stCursor).Render("▌")
	}
	return h.style(stRow).Render(" ")
}

const (
	markEmpty = "▫"
	markFull  = "▪"
)

// markCell renders the mark column: a hollow square for an unmarked row and a
// filled accent square for a marked one.
func (h rowHighlight) markCell(marked bool) string {
	if marked {
		return h.style(stSelMark).Render(markFull)
	}
	return h.style(stFaint).Render(markEmpty)
}
