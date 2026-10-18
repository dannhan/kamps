package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// mockW is the width of every mockup frame. Narrow enough for an 88-col
// terminal, wide enough for two panels side by side.
const mockW = 86

// Mockup is one candidate screen.
type Mockup struct {
	Key  string
	Desc string
	Body func() string
}

// Mockups returns the candidate screens, in presentation order.
func Mockups() []Mockup {
	return []Mockup{
		{"A", "Form / wizard triage (the flow you sketched)", mockA},
		{"B", "Two-pane list + detail (bordered box)", mockB},
		{"C", "Dashboard overview (at-a-glance panels)", mockC},
		{"D", "Board — six buckets as a 2x2 grid", mockD},
	}
}

// ---- the four candidates ----

func mockA() string {
	cw := mockW - 4
	step := func(s string) string { return stStep.Render("◇") + " " + stLabel.Bold(true).Render(s) }
	active := func(s string) string { return stSection.Render("◆") + " " + stSection.Render(s) }

	on := stCheckOn.Render("◼")
	off := stCheckOff.Render("◻")

	body := []string{
		"",
		step("Preflight"),
		"   " + stRowDim.Render("backup 0d · sync 1/2 today · cookie ok ·") + " " + stSoon.Render("24 undecided past-due"),
		"",
		active("Select assignments to decide"),
		"   " + on + " " + row("ppl", "10-15", "K163827", "Progress Report 5 (UTS)", 44),
		"   " + off + " " + row("metnum", "09-02", "K152705", "Tugas Individu I. Galat", 44),
		"   " + on + " " + row("metnum", "09-16", "K152710", "Tugas Kelompok II. Solusi Numerik", 44),
		"   " + stKey.Render("↑↓") + stRowDim.Render(" move · ") + stKey.Render("space") + stRowDim.Render(" toggle · ") + stKey.Render("a") + stRowDim.Render(" all · ") + stKey.Render("/") + stRowDim.Render(" filter · ") + stKey.Render("enter") + stRowDim.Render(" confirm"),
		"",
		step("Action for 2 selected"),
		"   " + stValue.Render("track") + stRowDim.Render("   (est 60 · goal G-college)"),
		"",
		step("Review: lines to be written"),
		"   " + stRowDim.Render("- [ ] [ppl] Progress Report 5 (UTS) (K163827) due:2026-10-15"),
		"     " + stOverdue.Render("! will show as OVERDUE"),
		"     " + stRowDim.Render(`similar manual task: "Submit ppl Report 5" (duplicate?)`),
		"",
		step("Apply?") + "  " + stSection.Render("Yes"),
		"",
		step("Writing:") + " " + stDone.Render("today.py before ✓") + stRowDim.Render(" · 2 lines appended · ") + stDone.Render("today.py after ✓"),
		"",
		stDone.Render("└") + " " + stValue.Render("Done. 2 tracked.") + "  " + stRowDim.Render("Undo: git -C ~/docs/working restore tasks.md"),
	}
	return page(titleTabs("Tasks"), stLabel.Render("backup 0d · sync 1/2 · cookie ok"), body, mockW, cw)
}

func mockB() string {
	cw := mockW - 4
	lw, rw := 46, cw-47 // 35

	items := []struct{ flag, course, date, id, name string }{
		{"t", "ppl", "10-15", "K163827", "Progress Report 5 (UTS)"},
		{"m", "metnum", "09-02", "K152705", "Tugas Individu I. Galat"},
		{"i", "metnum", "09-16", "K152710", "Tugas Kelompok II. Solusi Numerik"},
		{"", "ppl", "10-19", "K172331", "Progress report 9"},
		{"", "metnum", "10-24", "K167686", "Proyek Metode Numerik"},
		{"", "metnum", "10-30", "K152742", "Tugas Individu VIII"},
		{"", "metnum", "11-02", "K152759", "QUIZ SPNL"},
		{"", "basdat", "11-07", "K160843", "Quiz Basis Data 1"},
		{"", "ppl", "11-12", "K172340", "Progress Report 7"},
		{"", "basdat", "11-20", "K160844", "Latihan Normalisasi"},
	}
	nameW := lw - 4 - 25
	rows := make([]string, 0, len(items))
	for i, it := range items {
		mark := " "
		if i == 0 {
			mark = stCursorGutter.Render("▌")
		}
		rows = append(rows, mark+flagGlyph(it.flag)+" "+
			pad(it.course, 6, false)+" "+pad(it.date, 5, false)+" "+
			pad(it.id, 8, false)+" "+stRow.Render(truncate(it.name, nameW)))
	}
	left := panel("list", lw, rows)

	detail := []string{
		stRow.Bold(true).Render("Progress Report 5 (UTS)"),
		"",
		kv("due", "2026-10-15 (2d overdue)", stOverdue),
		kv("LMS", "submit: unknown", stRowDim),
		kv("flag", "track", stFlagT),
		"",
		stLabel.Render("would write to tasks.md:"),
		stRowDim.Render("- [ ] [ppl] Progress Report 5"),
		stRowDim.Render("  (K163827) due:2026-10-15"),
		stOverdue.Render("  ! OVERDUE"),
	}
	right := panel("K163827 · ppl", rw, detail)

	body := []string{
		chips("undecided", []string{"undecided", "overdue", "tracked", "missed", "ignored"}, "course: all ▾", cw),
		"",
	}
	body = append(body, sideBySide(left, lw, right, rw)...)
	body = append(body,
		"",
		spread(stRowDim.Render("queue: ")+stFlagT.Render("1 track")+stRowDim.Render(" · ")+stFlagM.Render("1 missed")+stRowDim.Render(" · ")+stFlagI.Render("1 ignore"),
			stKey.Render("[c]")+" "+stRowDim.Render("review & commit"), cw),
		hints([]string{"t track", "i ignore", "m missed", "d done", "space select", "/ filter", "? help"}),
	)
	return page(titleTabs("Tasks"), stLabel.Render("backup 0d · sync 1/2 · cookie ok"), body, mockW, cw)
}

func mockC() string {
	cw := mockW - 4
	lw, rw := 46, cw-47

	today := []string{
		stOverdue.Render("OVERDUE  2"),
		"  " + stRowDim.Render("10-15") + " " + stCode.Render(pad("ppl", 7, false)) + " " + stRow.Render("Progress Report 5 (UTS)"),
		"  " + stRowDim.Render("09-02") + " " + stCode.Render(pad("metnum", 7, false)) + " " + stRow.Render("Tugas Individu I. Galat"),
		"",
		stSoon.Render("DUE TODAY  1"),
		"  " + stSoon.Render("10-18") + " " + stCode.Render(pad("metnum", 7, false)) + " " + stRow.Render("Kuis Numerik"),
		"",
		stRowDim.Render("NEXT 3 DAYS  2"),
		"  " + stRowDim.Render("10-19") + " " + stCode.Render(pad("ppl", 7, false)) + " " + stRow.Render("Progress report 9"),
		"  " + stRowDim.Render("10-21") + " " + stCode.Render(pad("metnum", 7, false)) + " " + stRow.Render("Latihan Bab 4"),
	}
	triage := []string{
		bar("track", 12, 24),
		bar("missed", 3, 24),
		bar("ignored", 5, 24),
		bar("undecided", 24, 24),
		"",
		stSoon.Render("24 past-due undecided"),
		stRowDim.Render("→ ") + stKey.Render("[c]") + stRowDim.Render(" review & commit"),
	}
	sync := []string{
		stRowDim.Render("backup ") + stDone.Render("0d") + stRowDim.Render(" · sync ") + stSoon.Render("1/2 today") + stRowDim.Render(" · cookie ") + stDone.Render("ok"),
		stRowDim.Render("last sync 2026-10-18 07:12 · 17638 Struktur Data B (+2 materials)"),
		stKey.Render("[s]") + stRowDim.Render(" sync now   ") + stKey.Render("[b]") + stRowDim.Render(" backup   ") + stKey.Render("[c]") + stRowDim.Render(" review & commit"),
	}

	body := []string{chips("today", []string{"today", "courses", "materials", "sync"}, "smt11 ▾", cw), ""}
	body = append(body, sideBySide(panel("today", lw, today), lw, panel("triage", rw, triage), rw)...)
	body = append(body, "")
	body = append(body, panel("sync", cw, sync)...)
	return page(titleTabs("Today"), stLabel.Render("backup 0d · sync 1/2 · cookie ok"), body, mockW, cw)
}

func mockD() string {
	cw := mockW - 4
	lw, rw := 40, cw-41

	col := func(title string, items [][3]string) []string {
		out := []string{}
		for _, it := range items {
			out = append(out, stRowDim.Render(it[0])+" "+stCode.Render(pad(it[1], 8, false))+" "+stRow.Render(truncate(it[2], lw-4-15)))
		}
		if len(items) == 0 {
			out = append(out, stEmpty.Render("—"))
		}
		return out
	}

	undecided := col("undecided", [][3]string{
		{"10-15", "ppl", "Progress Report 5 (UTS)"},
		{"10-19", "ppl", "Progress report 9"},
		{"10-24", "metnum", "Proyek Metode Numerik"},
		{"10-30", "metnum", "Tugas Individu VIII"},
		{"11-07", "basdat", "Quiz Basis Data 1"},
	})
	tracked := col("tracked", [][3]string{
		{"09-16", "metnum", "Tugas Kelompok II. Solusi Numerik"},
		{"10-22", "strukdat", "Submission Praktikum 6"},
	})
	missed := col("missed", [][3]string{
		{"09-02", "metnum", "Tugas Individu I. Galat"},
		{"09-09", "basdat", "Kuis 1 (lewat tenggat)"},
	})
	ignored := col("ignored", [][3]string{
		{"09-16", "metnum", "Tugas Kelompok II (template)"},
		{"09-23", "ppl", "Contoh laporan tahun lalu"},
	})

	row1 := sideBySide(panel("undecided 24", lw, undecided), lw, panel("tracked 12", rw, tracked), rw)
	row2 := sideBySide(panel("missed 3", lw, missed), lw, panel("ignored 5", rw, ignored), rw)

	body := []string{chips("board", []string{"board", "list", "today"}, "course: all ▾", cw), ""}
	body = append(body, row1...)
	body = append(body, "")
	body = append(body, row2...)
	body = append(body, "")
	body = append(body, spread(stRowDim.Render("space select · t track · i ignore · m missed"), stKey.Render("[c]")+" "+stRowDim.Render("review & commit"), cw))
	return page(titleTabs("Tasks"), stLabel.Render("backup 0d · sync 1/2 · cookie ok"), body, mockW, cw)
}

// ---- building blocks ----

func page(titleLeft, titleRight string, body []string, w, cw int) string {
	_ = cw
	lines := []string{frameTop(titleLeft, titleRight, w)}
	for _, b := range body {
		lines = append(lines, frameLine(b, w))
	}
	lines = append(lines, frameBottom(w))
	return strings.Join(lines, "\n")
}

func frameTop(left, right string, w int) string {
	f := w - lipgloss.Width(left) - lipgloss.Width(right) - 8
	if f < 1 {
		f = 1
	}
	return stBorder.Render("┌─ ") + left + stBorder.Render(" "+strings.Repeat("─", f)+" ") + right + stBorder.Render(" ─┐")
}

func frameLine(content string, w int) string {
	gap := w - 4 - lipgloss.Width(content)
	if gap < 0 {
		gap = 0
	}
	return stBorder.Render("│") + " " + content + strings.Repeat(" ", gap) + " " + stBorder.Render("│")
}

func frameBottom(w int) string {
	return stBorder.Render("└" + strings.Repeat("─", w-2) + "┘")
}

func panelTop(title string, w int) string {
	f := w - lipgloss.Width(title) - 5
	if f < 1 {
		f = 1
	}
	return stBorder.Render("┌─ ") + stBoxTtl.Render(title) + stBorder.Render(" "+strings.Repeat("─", f)+"┐")
}

func panelLine(content string, w int) string {
	gap := w - 4 - lipgloss.Width(content)
	if gap < 0 {
		gap = 0
	}
	return stBorder.Render("│") + " " + content + strings.Repeat(" ", gap) + " " + stBorder.Render("│")
}

func panelBottom(w int) string {
	return stBorder.Render("└" + strings.Repeat("─", w-2) + "┘")
}

func panel(title string, w int, rows []string) []string {
	out := []string{panelTop(title, w)}
	for _, r := range rows {
		out = append(out, panelLine(r, w))
	}
	return append(out, panelBottom(w))
}

// sideBySide joins two stacked panels, padding the shorter with blank lines.
func sideBySide(left []string, lw int, right []string, rw int) []string {
	n := max(len(left), len(right))
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

func titleTabs(active string) string {
	names := []string{"Tasks", "Materials", "Today", "Sync"}
	parts := make([]string, len(names))
	for i, n := range names {
		if n == active {
			parts[i] = stTabOn.Render(n)
		} else {
			parts[i] = stTabOff.Render(n)
		}
	}
	return stHeader.Render("kamps") + "  " + strings.Join(parts, "  ")
}

func chips(active string, names []string, right string, w int) string {
	parts := make([]string, len(names))
	for i, n := range names {
		if n == active {
			parts[i] = stChipOn.Render(n)
		} else {
			parts[i] = stChipOff.Render(n)
		}
	}
	return spread(strings.Join(parts, "  "), stLabel.Render(right), w)
}

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

func row(course, date, id, name string, nameW int) string {
	return stCode.Render(pad(course, 7, false)) + " " + stRowDim.Render(date) + " " +
		stRowDim.Render(pad(id, 8, false)) + " " + stRow.Render(truncate(name, nameW))
}

func kv(k, v string, style lipgloss.Style) string {
	return stLabel.Render(pad(k, 5, false)) + " " + style.Render(v)
}

func flagGlyph(flag string) string {
	switch flag {
	case "t":
		return stFlagT.Render("t")
	case "m":
		return stFlagM.Render("m")
	case "i":
		return stFlagI.Render("i")
	}
	return " "
}

// bar renders a labelled count bar, e.g. "track    12  ██████░░░░".
func bar(label string, n, total int) string {
	const barW = 14
	filled := n * barW / max(total, 1)
	style := stFlagT
	switch label {
	case "missed":
		style = stFlagM
	case "ignored":
		style = stFlagI
	case "undecided":
		style = stSoon
	}
	return stLabel.Render(pad(label, 9, false)) + stValue.Render(fmt.Sprintf("%3d", n)) + "  " +
		style.Render(strings.Repeat("█", filled)) + stRowDim.Render(strings.Repeat("░", barW-filled))
}

func spread(left, right string, w int) string {
	g := w - lipgloss.Width(left) - lipgloss.Width(right)
	if g < 1 {
		g = 1
	}
	return left + strings.Repeat(" ", g) + right
}
