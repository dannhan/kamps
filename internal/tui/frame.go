package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Panes, gh-dash style: a plain rounded border, with the pane's title as the
// first line of content underlined by a rule. Titles embedded in borders
// ("╭─ list ────╮") read as 1990s BBS and cost a row; the chrome stays out of
// the way instead.
//
// Every helper returns a string of exactly w visible columns, which is what the
// alignment tests check.

// rule is a horizontal line of exactly w columns.
func rule(w int) string {
	if w < 1 {
		return ""
	}
	return stRule.Render(strings.Repeat("─", w))
}

func paneBorder(focused bool) lipgloss.Style {
	if focused {
		return stBorderHi
	}
	return stBorder
}

func paneTop(w int, focused bool) string {
	return paneBorder(focused).Render("╭" + strings.Repeat("─", maxInt(0, w-2)) + "╮")
}

func paneBottom(w int, focused bool) string {
	return paneBorder(focused).Render("╰" + strings.Repeat("─", maxInt(0, w-2)) + "╯")
}

func paneLine(content string, w int, focused bool) string {
	inner := maxInt(0, w-4)
	content = ansiTruncate(content, inner)
	gap := inner - lipgloss.Width(content)
	if gap < 0 {
		gap = 0
	}
	return paneBorder(focused).Render("│") + " " + content +
		strings.Repeat(" ", gap) + " " + paneBorder(focused).Render("│")
}

// panel draws a bordered pane of width w. An empty title skips the title row.
// The height follows the content, so pair panels with sideBySide, which pads
// the shorter one.
func panel(title string, w int, rows []string, focused bool) []string {
	inner := maxInt(1, w-4)
	content := make([]string, 0, len(rows)+2)
	if title != "" {
		content = append(content, stPaneTitle.Render(truncate(title, inner)), rule(inner))
	}
	content = append(content, rows...)

	out := make([]string, 0, len(content)+2)
	out = append(out, paneTop(w, focused))
	for _, c := range content {
		out = append(out, paneLine(c, w, focused))
	}
	return append(out, paneBottom(w, focused))
}

// sideBySide joins two panes, padding the shorter one with blank pane lines so
// both borders line up.
func sideBySide(left []string, lw int, lFocused bool, right []string, rw int, rFocused bool) []string {
	n := maxInt(len(left), len(right))
	out := make([]string, n)
	for i := 0; i < n; i++ {
		l, r := paneLine("", lw, lFocused), paneLine("", rw, rFocused)
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		out[i] = l + " " + r
	}
	return out
}

// hintsFit drops items from the end until they fit in w columns.
func hintsFit(items []string, w int) []string {
	out := make([]string, 0, len(items))
	total := 0
	for _, it := range items {
		add := lipgloss.Width(it)
		if len(out) > 0 {
			add += 3 // " · "
		}
		if len(out) > 0 && total+add > w {
			break
		}
		total += add
		out = append(out, it)
	}
	return out
}

// hints renders "key label · key label …" for the help bar.
func hints(items []string) string {
	parts := make([]string, len(items))
	for i, it := range items {
		fields := strings.SplitN(it, " ", 2)
		if len(fields) == 2 {
			parts[i] = stHelpKey.Render(fields[0]) + " " + stHelpTxt.Render(fields[1])
		} else {
			parts[i] = stHelpKey.Render(fields[0])
		}
	}
	return strings.Join(parts, stHelpSep.Render(" · "))
}
