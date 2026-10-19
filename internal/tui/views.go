package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/ruyam/kamps/internal/model"
)

// shortCode maps a course id to its short label ("ppl", "metnum", …).
func (m *App) shortCode(courseID int) string {
	for _, c := range m.courses {
		if c.ID == courseID {
			if c.Short != "" {
				return c.Short
			}
			return c.Code
		}
	}
	return "—"
}

// rowCols picks the column widths for one table row, dropping the id column and
// then shrinking the course column as space runs out — never overflowing. w is
// the pane's interior: the margin, the cursor gutter, the selection marker and
// the triage flag all live in the 4-column left gutter. Tree rows carry one
// more lead column and use treeRowCols (tree.go).
func rowCols(w int, withID bool) (shortW, idW, nameW int) {
	return colsFor(w, withID, 4)
}

// colsFor is the shared column solve; lead is the width of the row's left
// gutter (margin + cursor + marker + any extra columns).
func colsFor(w int, withID bool, lead int) (shortW, idW, nameW int) {
	shortW = 7
	if w >= lead+36 {
		shortW = 8 // room for the full short label ("strukdat")
	}
	if withID && w >= lead+30 {
		idW = 8
	}
	for {
		prefix := lead + 1 + shortW + 1 + 5 + 1 // gutter + course + due
		if idW > 0 {
			prefix += idW + 1
		}
		nameW = w - prefix
		if nameW >= 6 {
			return
		}
		switch {
		case idW > 0:
			idW = 0
		case shortW > 3:
			shortW--
		default:
			nameW = maxInt(1, nameW)
			return
		}
	}
}

// tint adds the cursor band's background to a style, so a whole row reads as one
// selected line rather than a coloured marker next to unhighlighted text.
func tint(s lipgloss.Style, band bool) lipgloss.Style {
	if band {
		return s.Background(themeTokens.Surface)
	}
	return s
}

// gap is one column of row separator, and blanks is n columns of end-of-row
// padding. Both have to carry the band on a highlighted row.
//
// This is the whole trick behind a solid highlight: joining styled cells with a
// plain " " punches unstyled holes through the band, so the selection comes out
// as stripes with the background only starting at whichever cell happens to be
// first. Every column between the pane's borders must be drawn under the band's
// background — the marker, the gaps and the padding alike.
func gap(band bool) string {
	if !band {
		return " "
	}
	return tint(stRow, true).Render(" ")
}

func blanks(band bool, n int) string {
	if n <= 0 {
		return ""
	}
	if !band {
		return strings.Repeat(" ", n)
	}
	return tint(stRow, true).Render(strings.Repeat(" ", n))
}

// assignRow renders one assignment line of exactly contentW columns, where
// contentW is its pane's interior:
//
//	▌▪t strukdat 08-21 K150001  Kuis Struktur Data awal
//	││└┴─ triage flag
//	││ └── space-selected
//	│└──── where the keyboard is
//	└───── the row's own margin, so the band reaches the border
//
// The marker is part of the same string as the row, and the row is padded out to
// the full interior, so the cursor line is one continuous bar with no unstyled
// column anywhere inside it.
func (m *App) assignRow(a model.Assignment, contentW int, cursor, selected, withID bool) string {
	h := rowHighlight{cursor: cursor, mark: selected}

	margin := h.gap()
	gutter := h.gutter()
	pick := h.markCell(selected)

	shortW, idW, nameW := rowCols(contentW, withID)

	nameStyle := stRow
	if a.Triage != model.Undecided && !selected {
		nameStyle = stIgnored
	}

	parts := []string{
		margin + gutter + pick + h.style(triageStyle(a.Triage)).Render(flagChar(a.Triage)),
		h.style(stCode).Render(pad(m.shortCode(a.CourseID), shortW, false)),
		h.style(stRowDim).Render(listDate(a)),
	}
	if idW > 0 {
		parts = append(parts, h.style(stRowDim).Render(pad(a.ID, idW, false)))
	}
	parts = append(parts, h.style(nameStyle).Render(truncate(a.Name, nameW)))
	row := strings.Join(parts, h.gap())

	return row + h.blanks(contentW-lipgloss.Width(row))
}

// ---- Tasks: list layout (the tree) ----

func (m *App) listLines(w, budget int) []string {
	contentH := maxInt(1, budget-4)
	nodes := m.taskNodes()
	start, end := windowRange(len(nodes), m.taskCursor, contentH)
	rows := make([]string, 0, contentH)
	if len(nodes) == 0 {
		rows = append(rows, " "+stEmpty.Render("nothing matches this filter"))
	}
	for i := start; i < end; i++ {
		rows = append(rows, m.taskTreeRow(nodes, i, w-2))
	}
	title := fmt.Sprintf("courses · %s · %d assignments", m.filter, len(m.visible()))
	return panelWithTitle(title, func(inner int) string {
		return taskTitleLine(title, inner)
	}, w, padRows(rows, contentH), true)
}

func taskTitleLine(title string, width int) string {
	states := []struct {
		glyph string
		name  string
		style lipgloss.Style
	}{{"○", "new", stFlagNew}, {"◐", "tracked", stFlagT}, {"●", "done", stDone}, {"◆", "ignored", stFlagI}, {"×", "missed", stFlagM}}
	var full, glyphs []string
	for _, state := range states {
		full = append(full, state.style.Render(state.glyph)+" "+stDim.Render(state.name))
		glyphs = append(glyphs, state.style.Render(state.glyph))
	}
	for _, legend := range []string{strings.Join(full, "  "), strings.Join(glyphs, "  ")} {
		leftWidth := maxInt(1, width-lipgloss.Width(legend)-2)
		left := " " + stPaneTitle.Render(truncate(title, leftWidth-1))
		remaining := width - lipgloss.Width(left)
		if lipgloss.Width(legend)+2 <= remaining {
			return left + strings.Repeat(" ", remaining-lipgloss.Width(legend)) + legend
		}
	}
	return " " + stPaneTitle.Render(truncate(title, width-1))
}

// treeDetail fills the right pane for whatever row the cursor is on: an
// assignment, or the course its children roll up into.
func (m *App) treeDetail(nodes []node, width int) ([]string, string) {
	if m.taskCursor < 0 || m.taskCursor >= len(nodes) {
		return nil, "detail"
	}
	n := nodes[m.taskCursor]
	if n.kind == nodeTask || n.kind == nodeTaskMaterial {
		return m.detailRows(n.task, width), n.task.ID + " · " + m.shortCode(n.courseID)
	}
	title := n.term
	if n.kind != nodeTerm {
		title = m.shortCode(n.courseID)
		if course, ok := m.courseByID(n.courseID); ok && course.Name != "" {
			title = course.Name
		}
	}
	return m.courseDetail(n.courseID, width), title
}

// detailRows renders the right-hand pane. width is the pane's interior; every
// line carries its own one-column margin, and text is wrapped to what is left.
func (m *App) detailRows(a model.Assignment, width int) []string {
	now := time.Now()
	due, dueStyle := dueInfo(a, now)
	textW := maxInt(8, width-1)

	rows := []string{
		" " + stRow.Bold(true).Render(truncate(a.Name, textW)),
		"",
		" " + kv("due", due, dueStyle),
		" " + kv("LMS", "submit: unknown", stRowDim),
		" " + kv("flag", a.Triage.String(), triageStyle(a.Triage)),
	}
	if a.EstMinutes > 0 {
		est := fmt.Sprintf("%d min", a.EstMinutes)
		if a.Goal != "" {
			est += " · " + a.Goal
		}
		rows = append(rows, " "+kv("est", est, stRowDim))
	}
	rows = append(rows, "", " "+stLabel.Render("would write to tasks.md:"))
	for _, ln := range wrapCell(m.taskLine(a), textW) {
		rows = append(rows, " "+stRowDim.Render(ln))
	}
	if a.Overdue(now) {
		rows = append(rows, " "+stOverdue.Render("! OVERDUE"))
	}
	if dup := m.similarManualTask(a); dup != "" {
		rows = append(rows, " "+stLabel.Render("similar manual task:"))
		rows = append(rows, " "+stRowDim.Render(truncate(`"`+dup+`" (duplicate?)`, textW)))
	}
	return rows
}

// similarManualTask is a demo stand-in for the future working/ cross-check.
func (m *App) similarManualTask(a model.Assignment) string {
	if !strings.Contains(strings.ToLower(a.Name), "progress report 5") {
		return ""
	}
	return "Submit " + m.shortCode(a.CourseID) + " Report 5"
}

func (m *App) taskLine(a model.Assignment) string {
	due := ""
	if a.HasDue {
		due = " due:" + a.Due.Format("2006-01-02")
	}
	return fmt.Sprintf("- [ ] [%s] %s (%s)%s", m.shortCode(a.CourseID), a.Name, a.ID, due)
}

// ---- Tasks: dashboard layout ----

// shownCount is how many items of a section actually get a row: a section that
// has to be truncated spends its last row on the "… N more" line instead.
func shownCount(cap, total int) int {
	if cap <= 0 || total <= 0 {
		return 0
	}
	if total > cap {
		return cap - 1
	}
	return total
}

// courseStat is one line of the dashboard's per-course breakdown.
type courseStat struct {
	code      string
	undecided int
	total     int
}

// courseStats groups the backlog per course, biggest undecided pile first.
func (m *App) courseStats() []courseStat {
	byID := map[int]*courseStat{}
	var order []int
	for _, a := range m.assignments {
		st, ok := byID[a.CourseID]
		if !ok {
			st = &courseStat{code: m.shortCode(a.CourseID)}
			byID[a.CourseID] = st
			order = append(order, a.CourseID)
		}
		st.total++
		if a.Triage == model.Undecided {
			st.undecided++
		}
	}
	out := make([]courseStat, 0, len(order))
	for _, id := range order {
		out = append(out, *byID[id])
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].undecided != out[j].undecided {
			return out[i].undecided > out[j].undecided
		}
		return out[i].total > out[j].total
	})
	return out
}

// dashboardLines is the overview: what is overdue, due today and coming up on
// the left; how the whole backlog splits on the right. Every item in the today
// pane is live — ↑↓ walks it and t/m/i/u decide it.
func (m *App) dashboardLines(w, budget int) []string {
	leftW := w * 54 / 100
	rightW := w - leftW - 1
	leftInner := leftW - 2
	contentH := maxInt(3, budget-4)

	overdue, dueToday, soon := m.todayBuckets()
	alloc := m.dashAlloc(overdue, dueToday, soon)

	// The cursor index counts only *items*, in the same order dashShown()
	// returns them, so the highlight always matches the keyboard.
	todayRows := []string{}
	n := 0
	section := func(label string, items []model.Assignment, cap int, style lipgloss.Style) {
		if n > 0 {
			todayRows = append(todayRows, "")
		}
		// Group headers obey the same width rule as the rows, so every line of
		// the pane ends on the same column.
		head := stFaint.Render(" ▾ ") + style.Render(label) + stFaint.Render(fmt.Sprintf("  %d", len(items)))
		todayRows = append(todayRows, head+blanks(false, maxInt(0, leftInner-lipgloss.Width(head))))
		shown := shownCount(cap, len(items))
		for i := 0; i < shown; i++ {
			a := items[i]
			todayRows = append(todayRows, m.assignRow(a, leftInner, n == m.taskCursor, m.selected[a.ID], false))
			n++
		}
		if shown < len(items) {
			more := stFaint.Render(fmt.Sprintf("    … %d more", len(items)-shown))
			todayRows = append(todayRows, more+blanks(false, maxInt(0, leftInner-lipgloss.Width(more))))
		}
	}
	section("overdue", overdue, alloc[0], stOverdue)
	section("due today", dueToday, alloc[1], stSoon)
	section("next 3 days", soon, alloc[2], stFaint)

	und, tr, mi, ig := m.triageCounts()
	total := maxInt(1, len(m.assignments))
	summary := []string{
		bar("tracked", tr, total, rightW-2),
		bar("missed", mi, total, rightW-2),
		bar("undecided", und, total, rightW-2),
		bar("ignored", ig, total, rightW-2),
	}
	if contentH >= 7 {
		summary = append(summary, "", " "+stSoon.Render(fmt.Sprintf("%d past-due undecided", m.overdueCount())))
	}
	// Whatever room is left goes to a per-course breakdown, so the pane says
	// where the backlog actually lives.
	if contentH-len(summary) >= 3 {
		summary = append(summary, "", " "+stFaint.Render("by course"))
		for _, s := range m.courseStats() {
			if len(summary) >= contentH {
				break
			}
			bw := maxInt(3, (rightW-2)-16)
			summary = append(summary, " "+stCode.Render(pad(s.code, 8, false))+" "+
				stValue.Render(fmt.Sprintf("%2d", s.undecided))+stFaint.Render(fmt.Sprintf("/%-2d", s.total))+" "+
				barCells(s.undecided, s.total, bw, stSoon))
		}
	}

	return sideBySide(
		panel("today", leftW, padRows(todayRows, contentH), true), leftW, true,
		panel("backlog · "+m.termLabel(), rightW, padRows(summary, contentH), false), rightW, false,
	)
}

// lastSyncLine summarises the last sync: when, what, and what came in.
func (m *App) lastSyncLine() string {
	name := "—"
	if len(m.courses) > 0 {
		name = fmt.Sprintf("%d %s", m.courses[0].ID, m.courses[0].Name)
	}
	fresh := 0
	for _, mt := range m.materials {
		if !mt.Downloaded {
			fresh++
		}
	}
	stamp := time.Now().Add(-20 * time.Hour).Format("2006-01-02 15:04")
	return fmt.Sprintf("%s · %s (+%d materials)", stamp, name, fresh)
}

// termLabel is the term shown at the right of the dashboard's chip row.
func (m *App) termLabel() string {
	for _, c := range m.courses {
		if c.Term != "" {
			return c.Term
		}
	}
	return "this term"
}

// ---- Tasks: board layout (2x2 buckets) ----

func (m *App) boardLines(w, budget int) []string {
	colW := (w - 1) / 2
	colW2 := w - 1 - colW

	rowH := maxInt(6, (budget-1)/2)
	listRows := maxInt(1, rowH-4)

	col := func(idx, width int) []string {
		items := m.boardItems(idx)
		cursor := m.boardCursor[idx]
		start, end := windowRange(len(items), cursor, listRows)
		out := make([]string, 0, listRows)
		if len(items) == 0 {
			out = append(out, " "+stEmpty.Render("—"))
		}
		for i := start; i < end; i++ {
			out = append(out, m.assignRow(items[i], width-2, i == cursor && idx == m.boardCol, m.selected[items[i].ID], false))
		}
		return padRows(out, listRows)
	}
	title := func(idx int) string {
		return fmt.Sprintf("▾ %s (%d)", boardOrder[idx].String(), len(m.boardItems(idx)))
	}

	row1 := sideBySide(
		panel(title(0), colW, col(0, colW), m.boardCol == 0), colW, m.boardCol == 0,
		panel(title(1), colW2, col(1, colW2), m.boardCol == 1), colW2, m.boardCol == 1)
	row2 := sideBySide(
		panel(title(2), colW, col(2, colW), m.boardCol == 2), colW, m.boardCol == 2,
		panel(title(3), colW2, col(3, colW2), m.boardCol == 3), colW2, m.boardCol == 3)

	body := append([]string{}, row1...)
	body = append(body, "")
	return append(body, row2...)
}

// ---- Tasks: wizard layout ----

func (m *App) wizardLines(w, budget int) []string {
	inner := w - 2 // the pane's interior
	rows := []string{"", m.stepRail(inner), ""}

	switch m.wizStep {
	case 0:
		rows = append(rows, m.wizStepAttempt(inner)...)
	case 1:
		rows = append(rows, m.wizStepAction(inner)...)
	case 2:
		rows = append(rows, m.wizStepReview(inner)...)
	default:
		rows = append(rows, m.wizStepDone(inner)...)
	}

	rows = append(rows, "", " "+m.hintsLine(inner-1, "wizard"))
	return panel("review & commit", w, rows, true)
}

func (m *App) stepRail(step int) string {
	labels := []string{"Preflight", "Select", "Action", "Review", "Done"}
	parts := make([]string, len(labels))
	for i, l := range labels {
		switch {
		case i == 0, i-1 < step:
			parts[i] = stDone.Render("└ ") + stRowDim.Render(l)
		case i-1 == step:
			parts[i] = stSection.Render("◆ ") + stSection.Render(l)
		default:
			parts[i] = stStep.Render("◇ ") + stLabel.Render(l)
		}
	}
	return "  " + strings.Join(parts, stRowDim.Render("   "))
}

func (m *App) wizStepAttempt(w int) []string {
	out := []string{" " + stStep.Render("◇ ") + stLabel.Bold(true).Render("Select assignments to decide"), ""}
	items := m.visible()
	start, end := windowRange(len(items), m.taskCursor, 6)
	for i := start; i < end; i++ {
		// w is the pane's interior width, so the row fills it exactly and the
		// cursor band has no hole at the right edge.
		out = append(out, m.assignRow(items[i], w, i == m.taskCursor, m.selected[items[i].ID], true))
	}
	return out
}

func (m *App) wizStepAction(w int) []string {
	out := []string{" " + stSection.Render("◆ ") + stSection.Render(fmt.Sprintf("Action for %d selected", len(m.selected))), ""}
	for _, t := range []model.Triage{model.Tracked, model.Missed, model.Ignored} {
		mark := "  "
		if t == m.wizAction {
			mark = stSection.Render("▸ ")
		}
		extra := ""
		if t == model.Tracked {
			extra = stRowDim.Render("   (est 60 · goal G-college)")
		}
		out = append(out, "    "+mark+triageStyle(t).Render(t.String())+extra)
	}
	return out
}

func (m *App) wizStepReview(w int) []string {
	out := []string{" " + stStep.Render("◇ ") + stLabel.Bold(true).Render("Review: lines to be written"), ""}
	if len(m.pending) == 0 {
		return append(out, "    "+stRowDim.Render("nothing queued"))
	}
	n := 0
	for id := range m.pending {
		a := m.byID(id)
		if a == nil {
			continue
		}
		out = append(out, "    "+stRowDim.Render(truncate(m.taskLine(*a), maxInt(8, w-5))))
		if a.Overdue(time.Now()) {
			out = append(out, "      "+stOverdue.Render("! will show as OVERDUE"))
		}
		n++
		if n >= 6 {
			out = append(out, "    "+stRowDim.Render(fmt.Sprintf("… and %d more", len(m.pending)-n)))
			break
		}
	}
	return out
}

func (m *App) wizStepDone(w int) []string {
	mode := "nothing written yet (dry-run)"
	if !m.dryRun {
		mode = "written"
	}
	return []string{
		" " + stStep.Render("◇ ") + stLabel.Bold(true).Render("Writing"),
		"    " + stDone.Render("today.py before ✓") + stRowDim.Render(" · "+mode+" · ") + stDone.Render("today.py after ✓"),
		"",
		" " + stDone.Render("└ ") + stValue.Render("Done.") + "  " + stRowDim.Render("Undo: git -C ~/docs/working restore tasks.md"),
	}
}

// ---- Materials ----

// materialsLines is the Materials tree: courses as parents, weeks (or the
// "general" section) as children, the files under each. matNodes() is what
// the pane draws and what the keyboard walks.
func (m *App) materialsLines(w, budget int) []string {
	leftW := w * 58 / 100
	rightW := w - leftW - 1
	contentH := maxInt(3, budget-4)

	nodes := m.matNodes()
	leftInner := leftW - 2
	start, end := windowRange(len(nodes), m.matCursor, contentH)

	rows := []string{}
	if len(nodes) == 0 {
		rows = append(rows, " "+stEmpty.Render("nothing here"))
	}
	for i := start; i < end; i++ {
		n := nodes[i]
		rows = append(rows, m.matRow(n, leftInner, i == m.matCursor, m.materialNodeSelected(n)))
	}

	var detail []string
	title := "detail"
	if m.matCursor >= 0 && m.matCursor < len(nodes) {
		n := nodes[m.matCursor]
		switch n.kind {
		case nodeFile:
			detail = m.materialDetail(n.mat, rightW-2)
		case nodeWeek:
			files := m.sectionFiles(n.courseID, n.mat.Section)
			detail = m.matGroupDetail(sectionLabel(n.mat.Section), files, rightW-2)
			title = m.shortCode(n.courseID) + " · " + sectionLabel(n.mat.Section)
		case nodeCourse:
			files := m.courseFiles(n.courseID)
			detail = m.matGroupDetail(m.shortCode(n.courseID), files, rightW-2)
			title = m.shortCode(n.courseID)
		}
	}

	local := 0
	for _, mt := range m.materials {
		if mt.Downloaded {
			local++
		}
	}
	pane := fmt.Sprintf("▾ materials (%d) · %d local", len(m.materials), local)

	return sideBySide(
		panel(pane, leftW, padRows(rows, contentH), true), leftW, true,
		panel(title, rightW, padRows(detail, contentH), false), rightW, false,
	)
}

func (m *App) materialDetail(mt model.Material, width int) []string {
	state, stateStyle := materialState(mt)
	section := mt.Section
	if section == "" {
		section = "—"
	}
	size := humanSize(mt.Size)
	if mt.Size == 0 {
		size = "unknown"
	}
	origin := "Kulon2 course page"
	if mt.Kind == "url" {
		origin = "WhatsApp hand-out"
	}
	textW := maxInt(8, width-1)

	rows := []string{
		" " + stRow.Bold(true).Render(truncate(mt.Name, textW)),
		"",
		" " + kv("course", m.shortCode(mt.CourseID), stCode),
		" " + kv("week", section, stRowDim),
		" " + kv("kind", mt.Kind, stRowDim),
		" " + kv("size", size, stRowDim),
		" " + kv("state", state, stateStyle),
		" " + kv("from", origin, stRowDim),
	}
	if mt.URL != "" {
		rows = append(rows, "", " "+stFaint.Render("source"))
		for _, ln := range wrapCell(mt.URL, textW) {
			rows = append(rows, " "+stCode.Render(ln))
		}
	}
	if !mt.Downloaded {
		rows = append(rows, "", " "+stFaint.Render("press ")+stKey.Render("d")+stFaint.Render(" to fetch into the library"))
	}
	return rows
}

func materialState(mt model.Material) (string, lipgloss.Style) {
	if mt.Downloaded {
		return "local", stDone
	}
	if mt.Kind == "url" {
		return "link", stCode
	}
	return "new", stRowDim
}

// ---- Sync ----

func (m *App) syncLines(w, budget int) []string {
	leftW := w * 42 / 100
	rightW := w - leftW - 1
	contentH := maxInt(3, budget-4)

	und, tr, mi, ig := m.triageCounts()
	status := []string{
		" " + kv("state", "idle", stRowDim),
		" " + kv("today", "1/2 runs", stSoon),
		" " + kv("cookie", "ok (expires in ~6h)", stDone),
		" " + kv("backup", "0d old", stRowDim),
		"",
		" " + stFaint.Render("source"),
		" " + stRowDim.Render("  "+m.src.Label()),
		"",
		" " + stFaint.Render("last sync"),
		" " + stRowDim.Render("  "+truncate(m.lastSyncLine(), rightW-8)),
	}

	library := []string{
		" " + stFaint.Render("local library"),
		" " + stRowDim.Render("  ~/Documents/kamps/<course>/<week>/"),
		"",
		" " + stFaint.Render("triaged"),
		" " + stRowDim.Render(fmt.Sprintf("  tracked %d · missed %d · ignored %d · undecided %d", tr, mi, ig, und)),
		"",
		" " + stFaint.Render("keys"),
		" " + stRowDim.Render("  ") + stKey.Render("s") + stRowDim.Render(" run a sync"),
		" " + stRowDim.Render("  ") + stKey.Render("b") + stRowDim.Render(" back up state"),
	}

	return sideBySide(
		panel("status", leftW, padRows(status, contentH), true), leftW, true,
		panel("library", rightW, padRows(library, contentH), false), rightW, false,
	)
}

// ---- small helpers ----

func (m *App) hintsLine(w int, where string) string {
	_, _, items, _ := m.tabContent(w, maxInt(1, m.height-5))
	if where == "help" {
		items = []string{"↑↓ move", "tab switch", "? more"}
	}
	return hints(hintsFit(append(items, "? more"), w))
}

// padRows pads a pane's content out to n rows, so panes fill the screen even
// when there is little to show.
func padRows(rows []string, n int) []string {
	out := make([]string, 0, maxInt(n, len(rows)))
	out = append(out, rows...)
	for len(out) < n {
		out = append(out, "")
	}
	if len(out) > n {
		out = out[:n]
	}
	return out
}

func listDate(a model.Assignment) string {
	if !a.HasDue {
		return "  —  "
	}
	return a.Due.Format("01-02")
}

func dueInfo(a model.Assignment, now time.Time) (string, lipgloss.Style) {
	if !a.HasDue {
		return "no due date", stIgnored
	}
	date := a.Due.Format("2006-01-02")
	switch {
	case a.Overdue(now):
		return fmt.Sprintf("%s (%dd overdue)", date, -daysBetween(now, a.Due)), stOverdue
	case sameDay(a.Due, now):
		return date + " (today)", stSoon
	case a.Due.After(now) && a.Due.Before(now.AddDate(0, 0, 4)):
		return fmt.Sprintf("%s (%dd)", date, daysBetween(now, a.Due)), stSoon
	default:
		return date, stRowDim
	}
}

func daysBetween(from, to time.Time) int {
	return int(to.Sub(from).Hours() / 24)
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// wrapCell soft-wraps plain text to width, on spaces where possible.
func wrapCell(s string, w int) []string {
	if w < 8 {
		w = 8
	}
	var out []string
	for len(s) > w {
		cut := w
		for cut > 0 && s[cut] != ' ' {
			cut--
		}
		if cut <= 0 {
			cut = w
		}
		out = append(out, s[:cut])
		s = strings.TrimLeft(s[cut:], " ")
	}
	return append(out, s)
}

func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		switch {
		case r == 0x1b:
			inEsc = true
		case inEsc && r == 'm':
			inEsc = false
		case !inEsc:
			b.WriteRune(r)
		}
	}
	return b.String()
}
