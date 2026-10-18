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

// assignRow renders one assignment line, at most contentW columns, dropping the
// id column and shrinking the course column when space is tight.
//
// The cursor row is drawn as a solid bar — the strongest "you are here" we have
// — and a space-selected row carries a filled ▪ marker in the gutter.
func (m *App) assignRow(a model.Assignment, contentW int, cursor, selected, withID bool) string {
	mark := " "
	if cursor {
		mark = stCursorGutter.Render("▌")
	}
	pick := " "
	if selected {
		pick = stSelMark.Render("▪")
	}

	shortW, idW := 7, 0
	if contentW >= 40 {
		shortW = 8 // room for the full short label ("strukdat")
	}
	if withID {
		idW = 8
	}
	nameW := contentW - 13 - shortW - idW
	if nameW < 6 && withID {
		idW = 0
		nameW = contentW - 11 - shortW
	}
	if nameW < 6 {
		shortW = clampInt(contentW-17, 3, 7)
		nameW = contentW - 11 - shortW
	}
	if nameW < 1 {
		nameW = 1
	}

	nameStyle := stRow
	switch {
	case selected:
		nameStyle = stSelName
	case a.Triage != model.Undecided:
		nameStyle = stRowDim
	}

	parts := []string{
		mark + pick + flagGlyph(a.Triage),
		stCode.Render(pad(m.shortCode(a.CourseID), shortW, false)),
		stRowDim.Render(listDate(a)),
	}
	if idW > 0 {
		parts = append(parts, stRowDim.Render(pad(a.ID, idW, false)))
	}
	parts = append(parts, nameStyle.Render(truncate(a.Name, nameW)))
	row := strings.Join(parts, " ")

	if cursor {
		return stRowCursor.Render(pad(stripANSI(row), contentW, false))
	}
	return row
}

// ---- Tasks: list layout (two panes) ----

func (m *App) listLines(w, budget int) []string {
	cw := w - 4
	leftW := cw * 54 / 100
	rightW := cw - leftW - 1

	listRows := maxInt(1, budget-7)
	items := m.visible()
	start, end := windowRange(len(items), m.taskCursor, listRows)

	rows := make([]string, 0, end-start+1)
	if len(items) == 0 {
		rows = append(rows, stEmpty.Render("nothing matches this filter"))
	}
	for i := start; i < end; i++ {
		rows = append(rows, m.assignRow(items[i], leftW-4, i == m.taskCursor, m.selected[items[i].ID], true))
	}

	var detail []string
	if len(items) > 0 && m.taskCursor < len(items) {
		detail = m.detailRows(items[m.taskCursor], rightW-4)
	}

	left := panel("list", leftW, rows)
	right := panel(m.detailTitle(items), rightW, detail)

	body := []string{
		chips(m.filter, filterNames, m.layout.String()+" · course: all ▾", cw),
		"",
	}
	body = append(body, sideBySide(left, leftW, right, rightW)...)
	body = append(body, "", m.statusLine(cw))
	body = append(body, m.hintsLine(cw, "tasks"))
	return body
}

func (m *App) detailTitle(items []model.Assignment) string {
	if len(items) > 0 && m.taskCursor < len(items) {
		a := items[m.taskCursor]
		return a.ID + " · " + m.shortCode(a.CourseID)
	}
	return "detail"
}

func (m *App) detailRows(a model.Assignment, width int) []string {
	now := time.Now()
	due, dueStyle := dueInfo(a, now)

	rows := []string{
		stRow.Bold(true).Render(truncate(a.Name, width)),
		"",
		kv("due", due, dueStyle),
		kv("LMS", "submit: unknown", stRowDim),
		kv("flag", a.Triage.String(), triageStyle(a.Triage)),
	}
	if a.EstMinutes > 0 {
		est := fmt.Sprintf("%d min", a.EstMinutes)
		if a.Goal != "" {
			est += " · " + a.Goal
		}
		rows = append(rows, kv("est", est, stRowDim))
	}
	rows = append(rows, "", stLabel.Render("would write to tasks.md:"))
	for _, ln := range wrapCell(m.taskLine(a), width) {
		rows = append(rows, stRowDim.Render(ln))
	}
	if a.Overdue(now) {
		rows = append(rows, stOverdue.Render("  ! OVERDUE"))
	}
	if dup := m.similarManualTask(a); dup != "" {
		rows = append(rows, stLabel.Render("similar manual task:"))
		rows = append(rows, stRowDim.Render(truncate(`"`+dup+`" (duplicate?)`, width)))
	}
	return rows
}

// similarManualTask is a demo stand-in for the future working/ cross-check.
func (m *App) similarManualTask(a model.Assignment) string {
	if a.Triage != model.Undecided || !strings.Contains(strings.ToLower(a.Name), "progress report 5") {
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

// dashboardLines is the overview screen: what is overdue, due today and coming
// up on the left; how the whole backlog splits on the right; sync state below.
// Every item in the today panel is live — ↑↓ walks it and t/m/i/u decide it.
func (m *App) dashboardLines(w, budget int) []string {
	cw := w - 4
	leftW := cw * 54 / 100
	rightW := cw - leftW - 1

	panelH := m.dashPanelHeight()
	contentH := maxInt(1, panelH-2)

	overdue, dueToday, soon := m.todayBuckets()
	alloc := m.dashAlloc(overdue, dueToday, soon)

	// The today panel. The cursor index counts only *items*, in the same order
	// dashShown() returns them, so the highlight always matches the keyboard.
	todayRows := []string{}
	n := 0
	section := func(label string, items []model.Assignment, cap int, style lipgloss.Style) {
		if n > 0 {
			todayRows = append(todayRows, "")
		}
		todayRows = append(todayRows, style.Bold(true).Render(fmt.Sprintf("%s  %d", label, len(items))))
		shown := shownCount(cap, len(items))
		for i := 0; i < shown; i++ {
			a := items[i]
			todayRows = append(todayRows, m.assignRow(a, leftW-4, n == m.taskCursor, m.selected[a.ID], false))
			n++
		}
		if shown < len(items) {
			todayRows = append(todayRows, stRowDim.Render(fmt.Sprintf("   … %d more", len(items)-shown)))
		}
	}
	section("OVERDUE", overdue, alloc[0], stOverdue)
	section("DUE TODAY", dueToday, alloc[1], stSoon)
	section("NEXT 3 DAYS", soon, alloc[2], stRowDim)
	if len(todayRows) > contentH {
		todayRows = todayRows[:contentH]
	}

	und, tr, mi, ig := m.triageCounts()
	total := maxInt(1, len(m.assignments))
	triageRows := []string{
		bar("track", tr, total, rightW-4),
		bar("missed", mi, total, rightW-4),
		bar("ignore", ig, total, rightW-4),
		bar("undecided", und, total, rightW-4),
	}
	if contentH >= 6 {
		triageRows = append(triageRows, "", stSoon.Render(fmt.Sprintf("%d past-due undecided", m.overdueCount())))
	}
	if contentH >= 7 {
		triageRows = append(triageRows, stRowDim.Render("→ ")+stKey.Render("[c]")+stRowDim.Render(" review & commit"))
	}
	// Whatever room is left goes to a compact per-course breakdown, so the
	// panel says where the backlog actually lives.
	if contentH-len(triageRows) >= 3 {
		triageRows = append(triageRows, "", stLabel.Render("by course"))
		for _, s := range m.courseStats() {
			if len(triageRows) >= contentH {
				break
			}
			barW := maxInt(3, (rightW-4)-15)
			triageRows = append(triageRows, stCode.Render(pad(s.code, 8, false))+" "+
				stValue.Render(fmt.Sprintf("%2d", s.undecided))+stRowDim.Render(fmt.Sprintf("/%-2d", s.total))+" "+
				barCells(s.undecided, s.total, barW, stSoon))
		}
	}
	if len(triageRows) > contentH {
		triageRows = triageRows[:contentH]
	}

	syncRows := []string{
		stRowDim.Render("backup ") + stDone.Render("0d") + stRowDim.Render(" · sync ") + stSoon.Render("1/2 today") + stRowDim.Render(" · cookie ") + stDone.Render("ok"),
		stRowDim.Render(m.lastSyncLine()),
		stKey.Render("[s]") + stRowDim.Render(" sync now   ") + stKey.Render("[b]") + stRowDim.Render(" backup   ") + stKey.Render("[c]") + stRowDim.Render(" review & commit"),
	}

	body := []string{chips("today", []string{"today", "courses", "materials", "sync"}, m.termLabel()+" ▾", cw), ""}
	body = append(body, sideBySide(panel("today", leftW, todayRows), leftW, panel("triage", rightW, triageRows), rightW)...)
	body = append(body, "")
	body = append(body, panel("sync", cw, syncRows)...)
	body = append(body, "", m.statusLine(cw), m.hintsLine(cw, "tasks"))
	return body
}

// lastSyncLine summarises the last sync for the dashboard's sync panel.
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
	return fmt.Sprintf("last sync %s · %s (+%d materials)", stamp, name, fresh)
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
	cw := w - 4
	colW := (cw - 1) / 2
	colW2 := cw - 1 - colW

	avail := maxInt(4, budget-4) // chips, blank, blank, hints
	panelH := (avail - 1) / 2
	listRows := maxInt(1, panelH-2)

	col := func(idx, width int) []string {
		items := m.boardItems(idx)
		cursor := m.boardCursor[idx]
		start, end := windowRange(len(items), cursor, listRows)
		out := make([]string, 0, end-start+1)
		if len(items) == 0 {
			out = append(out, stEmpty.Render("—"))
		}
		for i := start; i < end; i++ {
			out = append(out, m.assignRow(items[i], width-4, i == cursor && idx == m.boardCol, m.selected[items[i].ID], false))
		}
		return out
	}
	title := func(idx int) string {
		return fmt.Sprintf("%s %d", boardOrder[idx].String(), len(m.boardItems(idx)))
	}

	row1 := sideBySide(panel(title(0), colW, col(0, colW)), colW, panel(title(1), colW2, col(1, colW2)), colW2)
	row2 := sideBySide(panel(title(2), colW, col(2, colW)), colW, panel(title(3), colW2, col(3, colW2)), colW2)

	body := []string{chips(m.filter, filterNames, m.layout.String()+" · course: all ▾", cw), ""}
	body = append(body, row1...)
	body = append(body, "")
	body = append(body, row2...)
	body = append(body, m.hintsLine(cw, "tasks"))
	return body
}

// ---- Tasks: wizard layout ----

func (m *App) wizardLines(w, budget int) []string {
	cw := w - 4
	body := []string{"", m.stepRail(m.wizStep), ""}

	switch m.wizStep {
	case 0:
		body = append(body, m.wizStepAttempt(cw)...)
		body = append(body, "   "+stKey.Render("↑↓")+stRowDim.Render(" move · ")+stKey.Render("space")+stRowDim.Render(" toggle · ")+stKey.Render("a")+stRowDim.Render(" all · ")+stKey.Render("enter")+stRowDim.Render(" confirm"))
	case 1:
		body = append(body, m.wizStepAction(cw)...)
		body = append(body, "   "+stKey.Render("t")+stRowDim.Render(" track · ")+stKey.Render("m")+stRowDim.Render(" missed · ")+stKey.Render("i")+stRowDim.Render(" ignore · ")+stKey.Render("enter")+stRowDim.Render(" apply"))
	case 2:
		body = append(body, m.wizStepReview(cw)...)
		body = append(body, "   "+stKey.Render("enter")+stRowDim.Render(" apply · ")+stKey.Render("esc")+stRowDim.Render(" back"))
	default:
		body = append(body, m.wizStepDone(cw)...)
	}

	body = append(body, "", m.statusLine(cw), m.hintsLine(cw, "wizard"))
	return body
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
	out := []string{stStep.Render("◇ ") + stLabel.Bold(true).Render("Select assignments to decide"), ""}
	items := m.visible()
	start, end := windowRange(len(items), m.taskCursor, 6)
	for i := start; i < end; i++ {
		a := items[i]
		row := m.assignRow(a, w-4, i == m.taskCursor, m.selected[a.ID], true)
		if i != m.taskCursor {
			row = " " + row
		}
		out = append(out, row)
	}
	return out
}

func (m *App) wizStepAction(w int) []string {
	out := []string{stSection.Render("◆ ") + stSection.Render(fmt.Sprintf("Action for %d selected", len(m.selected))), ""}
	for _, t := range []model.Triage{model.Tracked, model.Missed, model.Ignored} {
		mark := "  "
		if t == m.wizAction {
			mark = stSection.Render("▸ ")
		}
		extra := ""
		if t == model.Tracked {
			extra = stRowDim.Render("   (est 60 · goal G-college)")
		}
		out = append(out, "   "+mark+triageStyle(t).Render(t.String())+extra)
	}
	return out
}

func (m *App) wizStepReview(w int) []string {
	out := []string{stStep.Render("◇ ") + stLabel.Bold(true).Render("Review: lines to be written"), ""}
	if len(m.pending) == 0 {
		return append(out, "   "+stRowDim.Render("nothing queued"))
	}
	n := 0
	for id := range m.pending {
		a := m.byID(id)
		if a == nil {
			continue
		}
		out = append(out, "   "+stRowDim.Render(truncate(m.taskLine(*a), w-3)))
		if a.Overdue(time.Now()) {
			out = append(out, "     "+stOverdue.Render("! will show as OVERDUE"))
		}
		n++
		if n >= 6 {
			out = append(out, "   "+stRowDim.Render(fmt.Sprintf("… and %d more", len(m.pending)-n)))
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
		stStep.Render("◇ ") + stLabel.Bold(true).Render("Writing"),
		"   " + stDone.Render("today.py before ✓") + stRowDim.Render(" · "+mode+" · ") + stDone.Render("today.py after ✓"),
		"",
		stDone.Render("└ ") + stValue.Render("Done.") + "  " + stRowDim.Render("Undo: git -C ~/docs/working restore tasks.md"),
	}
}

// ---- Materials ----

func (m *App) materialsLines(w, budget int) []string {
	cw := w - 4
	rowsBudget := maxInt(1, budget-6)

	// fixed cells: sel(3)+week(8)+size(9)+state(8) + 4 separators = 32
	flexW := maxInt(10, cw-2-32)
	head := t("", 3) + " " + t("WEEK", 8) + " " + t("MATERIAL", flexW) + " " + t("SIZE", 9) + " " + t("STATE", 8)

	start, end := windowRange(len(m.materials), m.matCursor, rowsBudget)
	lines := []string{"  " + head}
	if len(m.materials) == 0 {
		lines = append(lines, stEmpty.Render("  nothing here"))
	}
	for i := start; i < end; i++ {
		mt := m.materials[i]
		sel := "[ ]"
		if m.selectedM[mt.ID] {
			sel = "[×]"
		}
		week := mt.Section
		if week == "" {
			week = "—"
		}
		state, stateStyle := materialState(mt)

		plain := pad(sel, 3, false) + " " + pad(week, 8, false) + " " +
			pad(truncate(mt.Name, flexW), flexW, false) + " " +
			pad(humanSize(mt.Size), 9, true) + " " + pad(state, 8, false)
		styled := stRowDim.Render(pad(sel, 3, false)) + " " +
			stRowDim.Render(pad(week, 8, false)) + " " +
			stRow.Render(pad(truncate(mt.Name, flexW), flexW, false)) + " " +
			stRowDim.Render(pad(humanSize(mt.Size), 9, true)) + " " +
			stateStyle.Render(pad(state, 8, false))

		if i == m.matCursor {
			lines = append(lines, stCursorGutter.Render("▌")+" "+stRowCursor.Render(plain))
			continue
		}
		lines = append(lines, "  "+styled)
	}
	lines = append(lines, "", m.materialStatus(cw), m.hintsLine(cw, "materials"))
	return lines
}

func (m *App) materialStatus(w int) string {
	local := 0
	for _, mt := range m.materials {
		if mt.Downloaded {
			local++
		}
	}
	left := stRowDim.Render(fmt.Sprintf("%d local · %d missing", local, len(m.materials)-local))
	right := stKey.Render("[d]") + " " + stRowDim.Render("download")
	return spread(left, right, w)
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
	cw := w - 4
	und, tr, mi, ig := m.triageCounts()
	rows := []string{
		kv("state", "idle", stRowDim),
		kv("today", "1/2 runs", stSoon),
		kv("cookie", "ok (expires in ~6h)", stDone),
		kv("backup", "0d old · ~/.local/share/kamps/backups", stRowDim),
		"",
		stLabel.Render("local library"),
		stRowDim.Render("  ~/Documents/kamps/<course>/<week>/"),
		"",
		stLabel.Render("triaged"),
		stRowDim.Render(fmt.Sprintf("  tracked %d · missed %d · ignored %d · undecided %d", tr, mi, ig, und)),
		"",
		stKey.Render("[s]") + stRowDim.Render(" run a sync   ") + stKey.Render("[b]") + stRowDim.Render(" back up state"),
	}
	body := []string{chips("sync", []string{"status", "runs", "library"}, m.src.Label(), cw), ""}
	body = append(body, panel("sync", cw, rows)...)
	body = append(body, m.hintsLine(cw, "sync"))
	return body
}

// ---- small helpers ----

func (m *App) hintsLine(w int, where string) string {
	var items []string
	switch where {
	case "materials":
		items = []string{"↑↓ move", "space select", "d download", "/ filter", "tab", "? help", "q quit"}
	case "sync":
		items = []string{"s sync", "? help", "q quit"}
	case "wizard":
		items = []string{"↑↓ move", "space toggle", "enter confirm", "esc back", "? help"}
	default:
		items = []string{"↑↓ move", "space select", "t/m/i/u triage", "/ filter", "v view", "c commit", "? help"}
	}
	return hints(hintsFit(items, w))
}

func t(s string, w int) string {
	return stHeaderDim.Bold(true).Render(pad(s, w, false))
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
