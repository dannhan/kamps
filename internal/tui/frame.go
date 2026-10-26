package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Shell lines.
//
// The main area is not boxed: the top information and the footer are fenced
// off with full-width rules, which leaves every column for content and keeps a
// split pane from feeling squeezed between two borders. Every helper returns a
// string of exactly w visible columns, which is what the alignment tests check.

// titleLine is the filter/status row between the tab bar and the first rule.
func titleLine(title, right string, width int) string {
	inner := maxInt(0, width-2)
	rightWidth := lipgloss.Width(right)
	leftWidth := maxInt(0, inner-rightWidth-1)
	left := stPaneTitle.Render(truncate(title, maxInt(0, leftWidth-1)))
	gap := maxInt(1, inner-lipgloss.Width(left)-rightWidth)
	return fitLine(" "+left+strings.Repeat(" ", gap)+right+" ", width)
}

// bodyLines clamps the content area to height full-width rows, padding with
// blanks and never letting a row spill past the terminal edge.
func bodyLines(rows []string, width, height int) []string {
	if height <= 0 {
		return nil
	}
	out := make([]string, 0, height)
	for i := 0; i < height; i++ {
		content := ""
		if i < len(rows) {
			content = rows[i]
		}
		out = append(out, fitLine(content, width))
	}
	return out
}

func fixedTabCell(label string, active bool, width int) string {
	text := " " + pad(truncate(label, maxInt(0, width-2)), maxInt(0, width-2), false) + " "
	return tabCellStyle(active).Render(text)
}

func rule(w int) string {
	if w < 1 {
		return ""
	}
	return stRule.Render(strings.Repeat("─", w))
}

// hints renders "key label · key label …" for the footer.
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
