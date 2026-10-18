package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Hand-drawn boxes. lipgloss.Border cannot embed a title in the border, and we
// want titles like "╭─ list ────╮", so we build the lines ourselves. Every
// helper returns a string of *exactly* w visible columns (see the alignment
// tests) so nested boxes line up.
//
// The outer frame is drawn in Charm purple (stFrame); nested panels use the
// muted grey stBorder, so the shell reads at a glance.

func frameTop(left, right string, w int) string {
	f := w - lipgloss.Width(left) - lipgloss.Width(right) - 8
	if f < 1 {
		f = 1
	}
	return stFrame.Render("╭─ ") + left + stFrame.Render(" "+strings.Repeat("─", f)+" ") + right + stFrame.Render(" ─╮")
}

func frameLine(content string, w int) string {
	gap := w - 4 - lipgloss.Width(content)
	if gap < 0 {
		gap = 0
	}
	return stFrame.Render("│") + " " + content + strings.Repeat(" ", gap) + " " + stFrame.Render("│")
}

func frameBottom(w int) string {
	return stFrame.Render("╰" + strings.Repeat("─", w-2) + "╯")
}

func panelTop(title string, w int) string {
	f := w - lipgloss.Width(title) - 5
	if f < 1 {
		f = 1
	}
	return stBorder.Render("╭─ ") + stBoxTtl.Render(title) + stBorder.Render(" "+strings.Repeat("─", f)+"╮")
}

func panelLine(content string, w int) string {
	gap := w - 4 - lipgloss.Width(content)
	if gap < 0 {
		gap = 0
	}
	return stBorder.Render("│") + " " + content + strings.Repeat(" ", gap) + " " + stBorder.Render("│")
}

func panelBottom(w int) string {
	return stBorder.Render("╰" + strings.Repeat("─", w-2) + "╯")
}

// panel wraps rows in a titled box of width w.
func panel(title string, w int, rows []string) []string {
	out := []string{panelTop(title, w)}
	for _, r := range rows {
		out = append(out, panelLine(r, w))
	}
	return append(out, panelBottom(w))
}

// sideBySide joins two stacked panels, padding the shorter with blank lines.
func sideBySide(left []string, lw int, right []string, rw int) []string {
	n := maxInt(len(left), len(right))
	out := make([]string, n)
	for i := 0; i < n; i++ {
		l, r := panelLine("", lw), panelLine("", rw)
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

// chips renders the filter row: the active chip highlighted, the rest muted,
// with a right-aligned value. Chips are dropped (never the active one) when the
// row would not fit, and the right value is dropped first.
func chips(active string, names []string, right string, w int) string {
	seg := func(n string) string {
		if n == active {
			return stChipOn.Render(n)
		}
		return stChipOff.Render(n)
	}
	render := func(list []string) string {
		parts := make([]string, len(list))
		for i, n := range list {
			parts[i] = seg(n)
		}
		return strings.Join(parts, "  ")
	}

	list := append([]string(nil), names...)
	for {
		left := render(list)
		if lipgloss.Width(left) <= w {
			if right != "" && lipgloss.Width(left)+1+lipgloss.Width(right) <= w {
				return spread(left, right, w)
			}
			return left
		}
		if len(list) <= 1 {
			return left
		}
		dropped := false
		for i := len(list) - 1; i >= 0; i-- {
			if list[i] != active {
				list = append(list[:i], list[i+1:]...)
				dropped = true
				break
			}
		}
		if !dropped {
			return left
		}
	}
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

// hints renders "key label · key label …".
func hints(items []string) string {
	parts := make([]string, len(items))
	for i, it := range items {
		fields := strings.SplitN(it, " ", 2)
		if len(fields) == 2 {
			parts[i] = stKey.Render(fields[0]) + " " + stRowDim.Render(fields[1])
		} else {
			parts[i] = stKey.Render(fields[0])
		}
	}
	return strings.Join(parts, stRowDim.Render(" · "))
}
