package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/ruyam/kamps/internal/model"
)

// View renders the whole screen.
func (m *App) View() string {
	header := m.headerView()
	tabs := m.tabBar()
	rule := stRule.Render(strings.Repeat("─", m.width))

	body := m.bodyView()
	lines := []string{header, "", tabs, rule, body, rule, m.footerView()}
	return strings.Join(lines, "\n")
}

// ---- chrome ----

func (m *App) headerView() string {
	left := stHeader.Render("◆ kamps") + "  " + stHeaderDim.Render(m.src.Label())
	right := m.syncIndicator()
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func (m *App) syncIndicator() string {
	term := m.term()
	termBit := ""
	if term != "" {
		termBit = stHeaderDim.Render(term) + "  "
	}
	if m.syncing {
		return termBit + m.spinner.View() + stSoon.Render(" syncing")
	}
	return termBit + stDone.Render("● ready")
}

func (m *App) term() string {
	for _, c := range m.courses {
		if c.Term != "" {
			return c.Term
		}
	}
	return ""
}

func (m *App) tabBar() string {
	parts := make([]string, 0, secCount)
	for s := Section(0); s < secCount; s++ {
		label := fmt.Sprintf("%s %d", s, m.len(s))
		if s == m.section {
			parts = append(parts, stTabActive.Render(label))
		} else {
			parts = append(parts, stTabIdle.Render(label))
		}
	}
	return " " + strings.Join(parts, " ")
}

func (m *App) footerView() string {
	if m.toast != "" {
		return " " + stToast.Render(m.toast)
	}
	return " " + m.help.View(m.keys)
}

// ---- body ----

func (m *App) bodyView() string {
	if m.helpOn {
		return m.helpBody()
	}
	if len(m.courses) == 0 && len(m.assignments) == 0 && len(m.materials) == 0 {
		return m.padBody(stEmpty.Render("  Loading…"))
	}
	switch m.section {
	case SecCourses:
		return m.padBody(m.coursesView())
	case SecAssignments:
		return m.padBody(m.assignmentsView())
	case SecMaterials:
		return m.padBody(m.materialsView())
	case SecDownloads:
		return m.padBody(m.downloadsView())
	}
	return ""
}

func (m *App) helpBody() string {
	title := stTitle.Render("kamps — keys")
	full := m.help.FullHelpView(m.keys.FullHelp())
	hint := stHelpBox.Render(title + "\n\n" + full + "\n\n" + stFooter.Render("press ? or esc to close"))
	return m.padBody(hint)
}

func (m *App) padBody(s string) string {
	have := lipgloss.Height(s)
	want := m.bodyHeight()
	if have < want {
		s += strings.Repeat("\n", want-have)
	}
	return s
}

// ---- tables ----

type cell struct {
	text  string
	width int
	style lipgloss.Style
	right bool
}

func (m *App) coursesView() string {
	headers := []cell{
		t("NAME", 28), t("CODE", 11), t("MATERI", 6), t("DUE", 4), t("NOTE", m.flex(28, 11, 6, 4)),
	}
	rows := make([][]cell, 0, len(m.courses))
	for _, c := range m.courses {
		mats := 0
		for _, mt := range m.materials {
			if mt.CourseID == c.ID {
				mats++
			}
		}
		dues := 0
		for _, a := range m.assignments {
			if a.CourseID == c.ID && a.Status != model.Done && a.HasDue {
				dues++
			}
		}
		note := ""
		style := stRow
		if c.Source == "whatsapp" || c.Source == "manual" {
			note = c.Source
			style = stRowDim
		} else if c.HasPraktikum {
			note = "praktikum"
			style = stRow
		}
		rows = append(rows, []cell{
			{c.Name, 28, style, false},
			{c.Code, 11, stCode, false},
			{fmt.Sprintf("%d", mats), 6, stRowDim, true},
			{fmt.Sprintf("%d", dues), 4, dueCountStyle(dues), true},
			{note, m.flex(28, 11, 6, 4), stTag, false},
		})
	}
	return m.tableView(headers, rows, m.cursors[SecCourses])
}

func (m *App) assignmentsView() string {
	flexW := m.flex(10, 11, 9)
	headers := []cell{
		t("DUE", 10), t("COURSE", 11), t("STATUS", 9), t("ASSIGNMENT", flexW),
	}
	now := time.Now()
	rows := make([][]cell, 0, len(m.assignments))
	for _, a := range m.assignments {
		dueStyle, dueText := dueCell(a, now)
		stStyle := statusStyle(a.Status)
		nameStyle := stRow
		if a.Status == model.Done {
			nameStyle = stRowDim
		}
		rows = append(rows, []cell{
			{dueText, 10, dueStyle, false},
			{a.CourseCode, 11, stCode, false},
			{a.Status.Glyph() + " " + a.Status.String(), 9, stStyle, false},
			{a.Name, flexW, nameStyle, false},
		})
	}
	return m.tableView(headers, rows, m.cursors[SecAssignments])
}

func (m *App) materialsView() string {
	flexW := m.flex(3, 8, 9, 8)
	headers := []cell{
		t("", 3), t("WEEK", 8), t("MATERIAL", flexW), t("SIZE", 9), t("STATE", 8),
	}
	rows := make([][]cell, 0, len(m.materials))
	for _, mt := range m.materials {
		sel, selStyle := "[ ]", stRowDim
		if m.selected[mt.ID] {
			sel, selStyle = "[×]", stDone
		}
		week := mt.Section
		if week == "" {
			week = "—"
		}
		state, stateStyle := materialState(mt, m.progress[mt.ID])
		rows = append(rows, []cell{
			{sel, 3, selStyle, false},
			{week, 8, stRowDim, false},
			{mt.Name, flexW, stRow, false},
			{humanSize(mt.Size), 9, stRowDim, true},
			{state, 8, stateStyle, false},
		})
	}
	return m.tableView(headers, rows, m.cursors[SecMaterials])
}

func (m *App) downloadsView() string {
	if len(m.queue) == 0 {
		done := 0
		for _, mt := range m.materials {
			if mt.Downloaded {
				done++
			}
		}
		msg := fmt.Sprintf("  No active downloads.\n\n  %d file(s) already local.\n  Press %s on the Materials tab to queue one.",
			done, stCode.Render("d"))
		return "\n" + msg + "\n"
	}
	var b strings.Builder
	b.WriteString("\n")
	for _, id := range m.queue {
		name := id
		for _, mt := range m.materials {
			if mt.ID == id {
				name = mt.Name
				break
			}
		}
		frac := m.progress[id]
		b.WriteString(fmt.Sprintf("  %s\n", name))
		b.WriteString("    " + renderBar(frac, m.flex(0)) + " " + stRowDim.Render(fmt.Sprintf("%3.0f%%", frac*100)) + "\n\n")
	}
	return b.String()
}

func (m *App) tableView(headers []cell, rows [][]cell, cursor int) string {
	var b strings.Builder
	b.WriteString("  " + renderCells(headers, true, true) + "\n")
	start, end := windowRange(len(rows), cursor, m.bodyHeight()-1)
	if len(rows) == 0 {
		b.WriteString(stEmpty.Render("  nothing here") + "\n")
		return b.String()
	}
	for i := start; i < end; i++ {
		if i == cursor {
			plain := renderCells(rows[i], true, false)
			line := stCursorGutter.Render("▌") + " " + stRowCursor.Render(pad(plain, m.contentWidth(), false))
			b.WriteString(line + "\n")
		} else {
			b.WriteString("  " + renderCells(rows[i], false, false) + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// ---- helpers ----

func (m *App) contentWidth() int {
	if m.width-2 < 20 {
		return 20
	}
	return m.width - 2
}

func (m *App) bodyHeight() int {
	h := m.height - 5 // header, tabs, rule, rule, footer
	if h < 3 {
		return 3
	}
	return h
}

// flex computes the width of the final, flexible column.
func (m *App) flex(fixed ...int) int {
	sum := 0
	for _, w := range fixed {
		sum += w
	}
	rest := m.contentWidth() - sum - len(fixed)
	if rest < 8 {
		rest = 8
	}
	return rest
}

func t(s string, w int) cell {
	return cell{text: s, width: w, style: stHeaderDim.Bold(true)}
}

func renderCells(cells []cell, plain bool, head bool) string {
	parts := make([]string, len(cells))
	for i, c := range cells {
		p := pad(c.text, c.width, c.right)
		switch {
		case plain && head:
			parts[i] = stHeaderDim.Bold(true).Render(p)
		case plain:
			parts[i] = p
		default:
			parts[i] = c.style.Render(p)
		}
	}
	return strings.Join(parts, " ")
}

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

func truncate(s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	r := []rune(s)
	if w <= 1 {
		return string(r[:w])
	}
	return string(r[:w-1]) + "…"
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

func renderBar(frac float64, width int) string {
	if width < 4 {
		width = 20
	}
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	filled := int(frac * float64(width))
	return stDone.Render(strings.Repeat("█", filled)) +
		strings.Repeat("░", width-filled)
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

func dueCell(a model.Assignment, now time.Time) (lipgloss.Style, string) {
	if !a.HasDue {
		return stIgnored, "no due"
	}
	text := a.Due.Format("2006-01-02")
	switch {
	case a.Status == model.Done:
		return stDone, text
	case a.Due.Before(now):
		return stOverdue, text
	case a.Due.Sub(now) < 72*time.Hour:
		return stSoon, text
	default:
		return stRowDim, text
	}
}

func statusStyle(s model.Status) lipgloss.Style {
	switch s {
	case model.Done:
		return stDone
	case model.Doing:
		return stDoing
	case model.Ignored:
		return stIgnored
	default:
		return stTodo
	}
}

func materialState(mt model.Material, prog float64) (string, lipgloss.Style) {
	if prog > 0 && prog < 1 {
		return "↓ " + fmt.Sprintf("%d%%", int(prog*100)), stSoon
	}
	if mt.Downloaded {
		return "local", stDone
	}
	if mt.Kind == "url" {
		return "link", stCode
	}
	return "new", stRowDim
}

func dueCountStyle(n int) lipgloss.Style {
	if n == 0 {
		return stRowDim
	}
	return stSoon
}
