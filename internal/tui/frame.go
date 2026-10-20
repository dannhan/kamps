package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Pane lines and titles.
//
// A pane's interior is the whole span between its two border glyphs, w-2 columns.
// panel passes each row through untouched, so a row owns that space — margins
// included. That is what lets a highlighted row's background reach both borders
// instead of stopping one column short and leaving a visible gap; ordinary text
// rows simply start with a space of their own.
//
// Every helper returns a string of exactly w visible columns, which is what the
// alignment tests check.

// rule is a horizontal line of exactly w columns.
func titleLine(title, right string, width int) string {
	inner := maxInt(0, width-2)
	rightWidth := lipgloss.Width(right)
	leftWidth := maxInt(0, inner-rightWidth-1)
	left := stPaneTitle.Render(truncate(title, maxInt(0, leftWidth-1)))
	gap := maxInt(1, inner-lipgloss.Width(left)-rightWidth)
	return fitLine(" "+left+strings.Repeat(" ", gap)+right+" ", width)
}

func bodyFrame(rows []string, width, height int) []string {
	if height <= 0 {
		return nil
	}
	innerWidth := maxInt(0, width-2)
	contentHeight := maxInt(0, height-2)
	out := make([]string, 0, height)
	if width >= 2 {
		out = append(out, paneTop(width, true))
	} else {
		out = append(out, fitLine("", width))
	}
	for i := 0; i < contentHeight; i++ {
		content := ""
		if i < len(rows) {
			content = rows[i]
		}
		content = ansiTruncate(content, maxInt(0, innerWidth-2))
		line := " " + content + strings.Repeat(" ", maxInt(0, innerWidth-lipgloss.Width(content)-2)) + " "
		if width >= 2 {
			out = append(out, paneBorder(true).Render("│")+fitLine(line, innerWidth)+paneBorder(true).Render("│"))
		} else {
			out = append(out, fitLine(content, width))
		}
	}
	if width >= 2 {
		out = append(out, paneBottom(width, true))
	} else {
		out = append(out, fitLine("", width))
	}
	for len(out) < height {
		out = append(out, fitLine("", width))
	}
	return out[:height]
}

func fixedTabCell(label string, active bool, width int) string {
	text := " " + pad(truncate(label, maxInt(0, width-2)), maxInt(0, width-2), false) + " "
	cell := lipgloss.NewStyle().Foreground(themeTokens.Subtle).Background(themeTokens.Bg)
	if active {
		cell = lipgloss.NewStyle().Foreground(themeTokens.Bright).Background(themeTokens.Surface)
	}
	return cell.Render(text)
}

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
	inner := maxInt(0, w-2)
	content = ansiTruncate(content, inner)
	gap := inner - lipgloss.Width(content)
	if gap < 0 {
		gap = 0
	}
	return paneBorder(focused).Render("│") + content +
		strings.Repeat(" ", gap) + paneBorder(focused).Render("│")
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
func taskFooterHints(width int) string {
	items := []string{"↑↓ move", "h/l fold", "space select", "t/m/i decide", "c commit", "f filter", "? more"}
	kept := []string{"? more"}
	used := lipgloss.Width(kept[0])
	for _, item := range items[:len(items)-1] {
		candidateWidth := lipgloss.Width(item) + 3 + used
		if candidateWidth > width {
			break
		}
		kept = append(kept[:len(kept)-1], append([]string{item}, kept[len(kept)-1])...)
		used = candidateWidth
	}
	parts := make([]string, len(kept))
	for i, item := range kept {
		key, label, _ := strings.Cut(item, " ")
		parts[i] = stHelpKey.Render(key) + " " + stDim.Render(label)
	}
	return strings.Join(parts, stTreeLine.Render(" · "))
}

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
