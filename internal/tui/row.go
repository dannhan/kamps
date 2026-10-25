package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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

const markFull = "▪"

// caretGlyph is the expand/collapse indicator drawn before a parent row's
// name. Open containers show a down caret, folded ones a right caret.
func caretGlyph(open bool) string {
	if open {
		return "▾"
	}
	return "▸"
}

// containerLead renders the shared first columns of a parent row: the caret
// that says whether the container is open, then the mark column. It lives in
// the same slot as a task's triage glyph and mark, so child names line up.
func (h rowHighlight) containerLead(open, marked bool) string {
	return h.style(stFaint).Render(caretGlyph(open)) + h.style(stRow).Render(" ") +
		h.markCell(marked) + h.style(stRow).Render(" ")
}

// markCell renders the mark column: a hollow square for an unmarked row and a
// filled accent square for a marked one.
func (h rowHighlight) markCell(marked bool) string {
	if marked {
		return h.style(stSelMark).Render(markFull)
	}
	return h.style(stRow).Render(" ")
}

func (h rowHighlight) alignRight(content, meta string, width int) string {
	if meta == "" {
		return content
	}
	gap := width - lipgloss.Width(content) - lipgloss.Width(meta)
	if gap < 1 {
		return content
	}
	return content + h.style(stRow).Render(strings.Repeat(" ", gap)) + meta
}

func sharedFocusRow(content string, width int, focused bool) string {
	content = ansiTruncate(content, maxInt(0, width))
	padding := strings.Repeat(" ", maxInt(0, width-lipgloss.Width(content)))
	if focused {
		content = lipgloss.NewStyle().Background(themeTokens.Surface).Render(content)
		padding = stRow.Background(themeTokens.Surface).Render(padding)
	}
	return content + padding
}
