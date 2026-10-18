package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/ruyam/kamps/internal/model"
)

// ---- the node contract ----
//
// The two tree panes (Tasks list and Materials) and the keyboard share one
// model: the node list is the exact set of rows the pane draws, in the order
// it draws them. Cursor, selection and triage all read that list, so what is
// highlighted is always what the keys act on — treeNodes() and matNodes() are
// the single source of truth for both.

type nodeKind uint8

const (
	nodeCourse nodeKind = iota
	nodeTask
	nodeWeek
	nodeFile
)

type node struct {
	kind     nodeKind
	courseID int
	task     model.Assignment // nodeTask
	mat      model.Material   // nodeWeek (its Section) and nodeFile
}

// depth of a row: courses at the root, their children one step in, material
// files two steps in (course ▸ week ▸ file).
func (n node) depth() int {
	switch n.kind {
	case nodeTask, nodeWeek:
		return 1
	case nodeFile:
		return 2
	}
	return 0
}

func courseKey(id int) string { return fmt.Sprintf("c:%d", id) }

func weekKey(id int, section string) string { return fmt.Sprintf("w:%d:%s", id, section) }

// isOpen defaults to open: a tree that arrives collapsed is a tree nobody
// asked for. Collapsing is the explicit act.
func (m *App) isOpen(key string) bool {
	v, ok := m.expanded[key]
	return !ok || v
}

func (m *App) setOpen(key string, open bool) {
	if open {
		delete(m.expanded, key)
		return
	}
	m.expanded[key] = false
}

func sectionLabel(section string) string {
	if section == "" {
		return "general"
	}
	return section
}

// idsOf is the ids a node's children would carry, for selection bookkeeping.
func idsOf(files []model.Material) []string {
	out := make([]string, 0, len(files))
	for _, f := range files {
		out = append(out, f.ID)
	}
	return out
}

// toggleSet is the one selection toggle for both panes: a group turns the
// whole group on, or — when everything in it was on — off.
func toggleSet(set map[string]bool, ids []string) {
	allOn := len(ids) > 0
	for _, id := range ids {
		if !set[id] {
			allOn = false
			break
		}
	}
	for _, id := range ids {
		if allOn {
			delete(set, id)
		} else {
			set[id] = true
		}
	}
}

// allSelected reports whether a group row shows a mark.
func allSelected(set map[string]bool, ids []string) bool {
	if len(ids) == 0 {
		return false
	}
	for _, id := range ids {
		if !set[id] {
			return false
		}
	}
	return true
}

// ---- Tasks tree ----

// taskNodes is the Tasks list pane: one course row per course that has work
// under the active filter, its visible assignments underneath when expanded.
func (m *App) taskNodes() []node {
	vis := m.visible()
	if len(vis) == 0 {
		return nil
	}
	out := []node{}
	claimed := map[int]bool{}
	for _, c := range m.courses {
		var kids []model.Assignment
		for _, a := range vis {
			if a.CourseID == c.ID {
				kids = append(kids, a)
			}
		}
		if len(kids) == 0 {
			continue
		}
		claimed[c.ID] = true
		out = append(out, node{kind: nodeCourse, courseID: c.ID})
		if m.isOpen(courseKey(c.ID)) {
			for _, a := range kids {
				out = append(out, node{kind: nodeTask, courseID: c.ID, task: a})
			}
		}
	}
	// Assignments whose course has no row (or no course at all) still belong
	// in the tree — drop them on the floor and they stop existing.
	var extra []int
	rest := map[int][]model.Assignment{}
	for _, a := range vis {
		if !claimed[a.CourseID] {
			if len(rest[a.CourseID]) == 0 {
				extra = append(extra, a.CourseID)
			}
			rest[a.CourseID] = append(rest[a.CourseID], a)
		}
	}
	sort.Ints(extra)
	for _, id := range extra {
		out = append(out, node{kind: nodeCourse, courseID: id})
		if m.isOpen(courseKey(id)) {
			for _, a := range rest[id] {
				out = append(out, node{kind: nodeTask, courseID: id, task: a})
			}
		}
	}
	return out
}

func (m *App) courseByID(id int) (model.Course, bool) {
	for _, c := range m.courses {
		if c.ID == id {
			return c, true
		}
	}
	return model.Course{}, false
}

// courseAll is every assignment of a course, filter aside — that is what the
// detail pane and the counts should sum over.
func (m *App) courseAll(courseID int) []model.Assignment {
	out := []model.Assignment{}
	for _, a := range m.assignments {
		if a.CourseID == courseID {
			out = append(out, a)
		}
	}
	return out
}

func (m *App) courseChildren(courseID int) []model.Assignment {
	out := []model.Assignment{}
	for _, a := range m.visible() {
		if a.CourseID == courseID {
			out = append(out, a)
		}
	}
	return out
}

// ---- Materials tree ----

// matNodes is the Materials pane: course ▸ week (or "general") ▸ file. Weeks
// keep the source's own label; an empty section shows as "general".
func (m *App) matNodes() []node {
	out := []node{}
	seenCourse := map[int]bool{}
	for _, c := range m.courses {
		files := m.courseFiles(c.ID)
		if len(files) == 0 {
			continue
		}
		seenCourse[c.ID] = true
		out = append(out, node{kind: nodeCourse, courseID: c.ID})
		if m.isOpen(courseKey(c.ID)) {
			out = append(out, m.weekNodes(c.ID, files)...)
		}
	}
	var extra []int
	rest := map[int][]model.Material{}
	for _, mt := range m.materials {
		if !seenCourse[mt.CourseID] {
			if len(rest[mt.CourseID]) == 0 {
				extra = append(extra, mt.CourseID)
			}
			rest[mt.CourseID] = append(rest[mt.CourseID], mt)
		}
	}
	sort.Ints(extra)
	for _, id := range extra {
		files := rest[id]
		out = append(out, node{kind: nodeCourse, courseID: id})
		if m.isOpen(courseKey(id)) {
			out = append(out, m.weekNodes(id, files)...)
		}
	}
	return out
}

// weekNodes builds one week row per distinct section (in the order the
// materials arrive, section-sorted) and the files under it when open.
func (m *App) weekNodes(courseID int, files []model.Material) []node {
	out := []node{}
	seen := map[string]bool{}
	for _, mt := range files {
		if seen[mt.Section] {
			continue
		}
		seen[mt.Section] = true
		out = append(out, node{kind: nodeWeek, courseID: courseID, mat: model.Material{Section: mt.Section}})
		if m.isOpen(weekKey(courseID, mt.Section)) {
			for _, f := range files {
				if f.Section == mt.Section {
					out = append(out, node{kind: nodeFile, courseID: courseID, mat: f})
				}
			}
		}
	}
	return out
}

func (m *App) courseFiles(courseID int) []model.Material {
	out := []model.Material{}
	for _, mt := range m.materials {
		if mt.CourseID == courseID {
			out = append(out, mt)
		}
	}
	return out
}

func (m *App) sectionFiles(courseID int, section string) []model.Material {
	out := []model.Material{}
	for _, mt := range m.courseFiles(courseID) {
		if mt.Section == section {
			out = append(out, mt)
		}
	}
	return out
}

// ---- Tasks tree rows ----

// treeRowCols is the column solve for tree rows. Same solve as rowCols but
// with a five-column lead (margin, cursor, pick, chevron-or-flag, indent) and
// an attachment column, each of which degrades away as space runs out.
func treeRowCols(w int, withID bool) (shortW, idW, attW, nameW int) {
	shortW, idW, attW = 7, 0, 3
	if w >= 45 {
		shortW = 8 // room for the full short label ("strukdat")
	}
	if withID && w >= 39 {
		idW = 8
	}
	for {
		prefix := 5 + 1 + shortW + 1 + 5 + 1
		if idW > 0 {
			prefix += idW + 1
		}
		if attW > 0 {
			prefix += attW + 1
		}
		nameW = w - prefix
		if nameW >= 6 {
			return
		}
		switch {
		case attW > 0:
			attW = 0
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

// attCount is how many files hang off an assignment. The source names
// attachments "M<assignmentID>:<file>" — the demo convention, the contract the
// tree reads.
func (m *App) attCount(a model.Assignment) int {
	n := 0
	for _, mt := range m.materials {
		if mt.Kind == "attachment" && strings.HasPrefix(mt.ID, a.ID+":") {
			n++
		}
	}
	return n
}

// treeAssignHeader is the column header above the tree rows. It is padded to
// contentW by the same rule as the rows, so every line of the pane ends on
// the same column.
func treeAssignHeader(contentW int, withID bool) string {
	shortW, idW, attW, nameW := treeRowCols(contentW, withID)
	parts := []string{
		"     ",
		pad("course", shortW, false),
		pad("due", 5, false),
	}
	if idW > 0 {
		parts = append(parts, pad("id", idW, false))
	}
	if attW > 0 {
		parts = append(parts, pad("⌗", attW, false))
	}
	parts = append(parts, pad("title", nameW, false))
	line := strings.Join(parts, " ")
	return stFaint.Render(line) + blanks(false, contentW-lipgloss.Width(line))
}

// taskRow renders one assignment child of the tree. Same band discipline as
// assignRow: every cell through tint, joined with gap, padded with blanks, so
// the cursor line is one continuous bar.
func (m *App) taskRow(a model.Assignment, contentW int, cursor, selected, withID bool) string {
	band := cursor || selected

	margin := gap(band)
	gutter := gap(band)
	if cursor {
		gutter = tint(stCursor, band).Render("▌")
	}
	pick := gap(band)
	if selected {
		pick = tint(stSelMark, band).Render("▪")
	}

	shortW, idW, attW, nameW := treeRowCols(contentW, withID)

	nameStyle := stRow
	if a.Triage != model.Undecided && !selected {
		nameStyle = stIgnored
	}

	parts := []string{
		margin + gutter + pick + tint(triageStyle(a.Triage), band).Render(flagChar(a.Triage)) + gap(band),
		tint(stCode, band).Render(pad(m.shortCode(a.CourseID), shortW, false)),
		tint(stRowDim, band).Render(listDate(a)),
	}
	if idW > 0 {
		parts = append(parts, tint(stRowDim, band).Render(pad(a.ID, idW, false)))
	}
	if attW > 0 {
		att := m.attCount(a)
		attStyle := stRowDim
		text := ""
		if att > 0 {
			attStyle = stCode
			text = fmt.Sprintf("⌗%d", att)
		}
		parts = append(parts, tint(attStyle, band).Render(pad(text, attW, false)))
	}
	parts = append(parts, tint(nameStyle, band).Render(truncate(a.Name, nameW)))
	row := strings.Join(parts, gap(band))

	return row + blanks(band, contentW-lipgloss.Width(row))
}

// courseRow is a course parent row: chevron, code, how much of the course is
// decided (shown/total under the active filter), the name. The cursor can rest
// on it; space and t/m/i/u act on the whole course through it.
func (m *App) courseRow(n node, shown, total, contentW int, cursor bool) string {
	band := cursor

	margin := gap(band)
	gutter := gap(band)
	if cursor {
		gutter = tint(stCursor, band).Render("▌")
	}

	shortW, _, _, nameW := treeRowCols(contentW, false)

	chev := "▸"
	if m.isOpen(courseKey(n.courseID)) {
		chev = "▾"
	}
	counts := fmt.Sprintf("%d", shown)
	if shown != total {
		counts = fmt.Sprintf("%d/%d", shown, total)
	}
	name := m.shortCode(n.courseID)
	if c, ok := m.courseByID(n.courseID); ok && c.Name != "" {
		name = c.Name
	}

	parts := []string{
		margin + gutter + gap(band) + tint(stFaint, band).Render(chev) + gap(band),
		tint(stCode, band).Render(pad(m.shortCode(n.courseID), shortW, false)),
		tint(stRowDim, band).Render(pad(counts, 5, false)),
		tint(stRow, band).Render(truncate(name, nameW)),
	}
	row := strings.Join(parts, gap(band))

	return row + blanks(band, contentW-lipgloss.Width(row))
}

// courseDetail fills the right pane when the cursor rests on a course row:
// what the course is, and where its decisions stand.
func (m *App) courseDetail(courseID, width int) []string {
	name := m.shortCode(courseID)
	c, ok := m.courseByID(courseID)
	if ok && c.Name != "" {
		name = c.Name
	}
	kids := m.courseAll(courseID)
	total := len(kids)
	if total == 0 {
		return []string{" " + stEmpty.Render("nothing in this course")}
	}
	und, tr, mi, ig := 0, 0, 0, 0
	for _, a := range kids {
		switch a.Triage {
		case model.Tracked:
			tr++
		case model.Missed:
			mi++
		case model.Ignored:
			ig++
		default:
			und++
		}
	}

	due, dueStyle := m.courseNextDue(kids)
	textW := maxInt(8, width-1)
	rows := []string{
		" " + stRow.Bold(true).Render(truncate(name, textW)),
		"",
		" " + kv("items", fmt.Sprintf("%d", total), stRow),
		" " + kv("next due", due, dueStyle),
	}
	if ok {
		if c.Code != "" {
			rows = append(rows, " "+kv("code", c.Code, stCode))
		}
		if c.Term != "" {
			rows = append(rows, " "+kv("term", c.Term, stRowDim))
		}
		if c.Source != "" {
			rows = append(rows, " "+kv("kind", c.Source, stRowDim))
		}
	}
	rows = append(rows, "",
		bar("tracked", tr, total, width),
		bar("missed", mi, total, width),
		bar("ignored", ig, total, width),
		bar("undecided", und, total, width),
	)
	return rows
}

// courseNextDue is the soonest deadline still undecided in the course, or "—".
func (m *App) courseNextDue(kids []model.Assignment) (string, lipgloss.Style) {
	now := time.Now()
	best, found := model.Assignment{}, false
	for _, a := range kids {
		if a.Triage != model.Undecided || !a.HasDue {
			continue
		}
		if !found || a.Due.Before(best.Due) {
			best, found = a, true
		}
	}
	if !found {
		return "—", stRowDim
	}
	return dueInfo(best, now)
}

// ---- Materials rows ----

const matStateW = 8

// matRow renders one Materials row: course, week or file. Parent rows carry
// their counts in the state slot (n files); files carry their state there.
func (m *App) matRow(n node, contentW int, cursor, selected bool) string {
	band := cursor || selected

	margin := gap(band)
	gutter := gap(band)
	if cursor {
		gutter = tint(stCursor, band).Render("▌")
	}
	pick := gap(band)
	if selected {
		pick = tint(stSelMark, band).Render("▪")
	}
	lead := margin + gutter + pick

	state, stateStyle := "", stRowDim
	var files []model.Material
	switch n.kind {
	case nodeCourse:
		files = m.courseFiles(n.courseID)
		state = fmt.Sprintf("%d files", len(files))
		stateStyle = stFaint
	case nodeWeek:
		files = m.sectionFiles(n.courseID, n.mat.Section)
		state = fmt.Sprintf("%d files", len(files))
		stateStyle = stFaint
	case nodeFile:
		state, stateStyle = materialState(n.mat)
	}

	depth := n.depth()
	indent := blanks(band, depth*2)
	textW := maxInt(4, contentW-3-depth*2-1-matStateW-1)

	var mid string
	switch n.kind {
	case nodeCourse:
		chev := "▸"
		if m.isOpen(courseKey(n.courseID)) {
			chev = "▾"
		}
		name := ""
		if c, ok := m.courseByID(n.courseID); ok {
			name = c.Name
		}
		if name == "" {
			name = m.shortCode(n.courseID)
		}
		local := 0
		for _, f := range files {
			if f.Downloaded {
				local++
			}
		}
		mid = tint(stFaint, band).Render(chev) + gap(band) +
			tint(stRow, band).Render(truncate(name, maxInt(1, textW-3))) +
			tint(stRowDim, band).Render(fmt.Sprintf(" · %d local", local))
	case nodeWeek:
		chev := "▸"
		if m.isOpen(weekKey(n.courseID, n.mat.Section)) {
			chev = "▾"
		}
		mid = tint(stFaint, band).Render(chev) + gap(band) +
			tint(stCode, band).Render(truncate(sectionLabel(n.mat.Section), maxInt(1, textW-3)))
	case nodeFile:
		mid = tint(stRow, band).Render(truncate(n.mat.Name, maxInt(1, textW-2)))
	}
	mid = ansiTruncate(mid, textW)
	// Pad to textW with band-aware blanks — plain spaces would punch a hole
	// through the cursor band (TestCursorBandIsContinuous exists for this).
	mid += blanks(band, maxInt(0, textW-lipgloss.Width(mid)))

	stateCell := tint(stateStyle, band).Render(pad(truncate(state, matStateW), matStateW, false))
	row := lead + indent + mid + gap(band) + stateCell

	return row + blanks(band, contentW-lipgloss.Width(row))
}

// matGroupDetail fills the right pane for a course or week row: how the files
// stand, and the first few of them.
func (m *App) matGroupDetail(title string, files []model.Material, width int) []string {
	textW := maxInt(8, width-1)
	var size int64
	local := 0
	for _, f := range files {
		size += f.Size
		if f.Downloaded {
			local++
		}
	}
	rows := []string{
		" " + stRow.Bold(true).Render(truncate(title, textW)),
		"",
		" " + kv("files", fmt.Sprintf("%d", len(files)), stRow),
		" " + kv("local", fmt.Sprintf("%d", local), stDone),
		" " + kv("size", humanSize(size), stRowDim),
	}
	if len(files) > 0 {
		rows = append(rows, "", " "+stFaint.Render("files"))
		for i, f := range files {
			if i == 8 {
				rows = append(rows, " "+stRowDim.Render(fmt.Sprintf("… %d more", len(files)-i)))
				break
			}
			state, style := materialState(f)
			rows = append(rows, " "+stRowDim.Render(pad(truncate(f.Name, textW-12), textW-12, false))+" "+
				style.Render(truncate(state, 6)))
		}
	}
	return rows
}
