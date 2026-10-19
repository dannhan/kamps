package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/ruyam/kamps/internal/model"
)

// ---- plain text helpers (ANSI-unaware input) ----

func pad(s string, w int, right bool) string {
	s = truncate(s, w)
	n := w - lipgloss.Width(s)
	if n <= 0 {
		return s
	}
	if right {
		return strings.Repeat(" ", n) + s
	}
	return s + strings.Repeat(" ", n)
}

// padStyled pads already-styled text to width using its visible width.
func padStyled(s string, w int) string {
	n := w - lipgloss.Width(s)
	if n <= 0 {
		return s
	}
	return s + strings.Repeat(" ", n)
}

// ansiTruncate clamps styled text to w visible columns without cutting an escape
// sequence in half, which is what plain rune-slicing would do.
func ansiTruncate(s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	return lipgloss.NewStyle().MaxWidth(w).Render(s)
}

// fitLine clamps styled text to exactly w visible columns: it truncates overlong
// lines (ANSI-aware) and pads short ones. Every line the app emits goes through
// this, so nothing can ever spill past the terminal.
func fitLine(s string, w int) string {
	if lipgloss.Width(s) > w {
		s = lipgloss.NewStyle().MaxWidth(w).Render(s)
	}
	return padStyled(s, w)
}

func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	return ansi.Truncate(s, w, "…")
}

func windowRange(total, cursor, height int) (int, int) {
	if height < 1 {
		height = 1
	}
	if total <= height {
		return 0, total
	}
	start := cursor - height/2
	if start < 0 {
		start = 0
	}
	if start+height > total {
		start = total - height
	}
	return start, start + height
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func humanSize(n int64) string {
	if n <= 0 {
		return "—"
	}
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	units := []string{"KB", "MB", "GB"}
	f := float64(n)
	i := -1
	for f >= unit && i < len(units)-1 {
		f /= unit
		i++
	}
	return fmt.Sprintf("%.1f %s", f, units[i])
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func lessAssignment(a, b model.Assignment) bool {
	if a.HasDue != b.HasDue {
		return a.HasDue // dated items first
	}
	if a.HasDue && !a.Due.Equal(b.Due) {
		return a.Due.Before(b.Due)
	}
	return a.ID < b.ID
}

func sortSlice(s []model.Assignment) {
	sort.Slice(s, func(i, j int) bool { return lessAssignment(s[i], s[j]) })
}

// ---- styled atoms ----

func flagChar(t model.Triage) string {
	switch t {
	case model.Tracked:
		return "◐"
	case model.Missed:
		return "×"
	case model.Ignored:
		return "◆"
	}
	return "○"
}

func triageStyle(t model.Triage) lipgloss.Style {
	switch t {
	case model.Tracked:
		return stFlagT
	case model.Missed:
		return stFlagM
	case model.Ignored:
		return stFlagI
	}
	return stFlagNew
}

func kv(k, v string, style lipgloss.Style) string {
	return stLabel.Render(pad(k, 6, false)) + " " + style.Render(v)
}

// bar renders a labelled count bar within width columns, e.g.
// "tracked   12  ██████░░░░░░░░".
func bar(label string, n, total, width int) string {
	return barW(label, n, total, width, 9)
}

// barW is bar with an explicit label column width. The leading column is the
// pane's margin, so the bar lines up with the rest of the pane's content.
func barW(label string, n, total, width, labelW int) string {
	barWidth := clampInt(width-(labelW+5)-1, 4, 24)
	return " " + stLabel.Render(pad(label, labelW, false)) + stValue.Render(fmt.Sprintf("%3d", n)) + "  " +
		barCells(n, total, barWidth, barStyleFor(label))
}

// barStyleFor colour-codes the filled part of a bar by what it counts. The
// undecided pile gets the loudest colour: it is the one that needs you.
func barStyleFor(label string) lipgloss.Style {
	switch label {
	case "track", "tracked":
		return stFlagT
	case "missed":
		return stFlagM
	case "ignore", "ignored":
		return stFlagI
	case "done":
		return stDone
	case "new", "undecided":
		return stFlagNew
	}
	return stFlagNew
}

// barCells renders just the filled/empty bar, width columns wide. The empty
// track uses the border grey so a filled segment always stands out, even for
// the muted "ignore" bar.
func barCells(n, total, width int, style lipgloss.Style) string {
	if width < 1 {
		width = 1
	}
	filled := clampInt(n*width/maxInt(total, 1), 0, width)
	return style.Render(strings.Repeat("█", filled)) + stBarEmpty.Render(strings.Repeat("░", width-filled))
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
