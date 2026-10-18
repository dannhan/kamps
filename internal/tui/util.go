package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

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

func truncate(s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	r := []rune(s)
	if w <= 1 {
		if w <= 0 {
			return ""
		}
		return string(r[:w])
	}
	return string(r[:w-1]) + "…"
}

func spread(left, right string, w int) string {
	g := w - lipgloss.Width(left) - lipgloss.Width(right)
	if g < 1 {
		g = 1
	}
	return left + strings.Repeat(" ", g) + right
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

func flagGlyph(t model.Triage) string {
	switch t {
	case model.Tracked:
		return stFlagT.Render("t")
	case model.Missed:
		return stFlagM.Render("m")
	case model.Ignored:
		return stFlagI.Render("i")
	}
	return " "
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
	return stRowDim
}

func kv(k, v string, style lipgloss.Style) string {
	return stLabel.Render(pad(k, 6, false)) + " " + style.Render(v)
}

// bar renders a labelled count bar within width columns, e.g.
// "tracked   12  ██████░░░░░░░░".
func bar(label string, n, total, width int) string {
	return barW(label, n, total, width, 9)
}

// barW is bar with an explicit label column width.
func barW(label string, n, total, width, labelW int) string {
	barWidth := clampInt(width-(labelW+5), 4, 24)
	return stLabel.Render(pad(label, labelW, false)) + stValue.Render(fmt.Sprintf("%3d", n)) + "  " +
		barCells(n, total, barWidth, triageStyle(triageForLabel(label)))
}

// barCells renders just the filled/empty bar, width columns wide.
func barCells(n, total, width int, style lipgloss.Style) string {
	if width < 1 {
		width = 1
	}
	filled := clampInt(n*width/maxInt(total, 1), 0, width)
	return style.Render(strings.Repeat("█", filled)) + stRowDim.Render(strings.Repeat("░", width-filled))
}

func triageForLabel(label string) model.Triage {
	switch label {
	case "tracked":
		return model.Tracked
	case "missed":
		return model.Missed
	case "ignored":
		return model.Ignored
	}
	return model.Undecided
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
