package tui

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"

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
	nodeTerm
	nodeBucket
	nodeTaskMaterial
	nodeMaterialBucket
	nodeMaterialAssignment
)

type node struct {
	kind     nodeKind
	courseID int
	term     string
	bucket   string
	key      string
	parents  []string
	task     model.Assignment
	mat      model.Material
}

func (n node) depth() int {
	if n.parents != nil {
		return len(n.parents)
	}
	switch n.kind {
	case nodeTask, nodeWeek, nodeMaterialBucket:
		return 1
	case nodeFile:
		return 2
	}
	return 0
}

func courseKey(id int) string { return fmt.Sprintf("c:%d", id) }

func weekKey(id int, section string) string { return fmt.Sprintf("w:%d:%s", id, section) }

func bucketKey(id int, bucket string) string { return fmt.Sprintf("b:%d:%s", id, bucket) }

func taskKey(id string) string { return "a:" + id }

func taskMaterialKey(id string) string { return "m:" + id }

func (m *App) taskIsOpen(key string, defaultOpen bool) bool {
	if open, ok := m.expanded[key]; ok {
		return open
	}
	return defaultOpen
}

func taskNodeDefaultOpen(n node) bool {
	if n.kind == nodeTask {
		return false
	}
	return n.kind != nodeBucket || n.bucket != "closed"
}

func (m *App) taskGroupIDs(group node) []string {
	var ids []string
	for _, assignment := range m.visible() {
		include := false
		switch group.kind {
		case nodeTerm:
			course, ok := m.courseByID(assignment.CourseID)
			term := course.Term
			if term == "" {
				term = "other"
			}
			include = term == group.term || !ok && group.term == "other"
		case nodeCourse:
			include = assignment.CourseID == group.courseID
		case nodeBucket:
			include = assignment.CourseID == group.courseID && (group.bucket == "needs decision" && assignment.Triage == model.Undecided || group.bucket == "tracked" && assignment.Triage == model.Tracked || group.bucket == "closed" && (assignment.Triage == model.Missed || assignment.Triage == model.Ignored))
		case nodeTask:
			include = assignment.ID == group.task.ID
		}
		if include {
			ids = append(ids, assignment.ID)
		}
	}
	return ids
}

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

func (m *App) taskNodes() []node {
	visible := m.visible()
	byCourse := make(map[int][]model.Assignment)
	for _, assignment := range visible {
		byCourse[assignment.CourseID] = append(byCourse[assignment.CourseID], assignment)
	}
	if len(byCourse) == 0 {
		return nil
	}

	courses := make(map[int]model.Course, len(m.courses))
	byTerm := make(map[string][]int)
	for _, course := range m.courses {
		if len(byCourse[course.ID]) == 0 {
			continue
		}
		courses[course.ID] = course
		term := course.Term
		if term == "" {
			term = "other"
		}
		byTerm[term] = append(byTerm[term], course.ID)
	}
	for courseID := range byCourse {
		if _, ok := courses[courseID]; ok {
			continue
		}
		courses[courseID] = model.Course{ID: courseID, Short: fmt.Sprintf("course-%d", courseID), Term: "other"}
		byTerm["other"] = append(byTerm["other"], courseID)
	}

	terms := make([]string, 0, len(byTerm))
	for term := range byTerm {
		terms = append(terms, term)
	}
	sort.Strings(terms)

	out := make([]node, 0, len(visible)*2)
	for _, term := range terms {
		termKey := "term:" + term
		out = append(out, node{kind: nodeTerm, term: term, key: termKey})
		if !m.taskIsOpen(termKey, true) {
			continue
		}
		courseIDs := byTerm[term]
		sort.Slice(courseIDs, func(i, j int) bool {
			left, right := courses[courseIDs[i]], courses[courseIDs[j]]
			leftName, rightName := left.Short, right.Short
			if leftName == "" {
				leftName = left.Code
			}
			if rightName == "" {
				rightName = right.Code
			}
			return leftName < rightName
		})
		for _, courseID := range courseIDs {
			courseKeyValue := courseKey(courseID)
			parents := []string{termKey}
			out = append(out, node{kind: nodeCourse, courseID: courseID, term: term, key: courseKeyValue, parents: parents})
			if !m.taskIsOpen(courseKeyValue, true) {
				continue
			}
			buckets := []struct {
				name string
				keep func(model.Assignment) bool
				open bool
			}{
				{"needs decision", func(a model.Assignment) bool { return a.Triage == model.Undecided }, true},
				{"tracked", func(a model.Assignment) bool { return a.Triage == model.Tracked }, true},
				{"closed", func(a model.Assignment) bool { return a.Triage == model.Missed || a.Triage == model.Ignored }, false},
			}
			for _, bucket := range buckets {
				var assignments []model.Assignment
				for _, assignment := range byCourse[courseID] {
					if bucket.keep(assignment) {
						assignments = append(assignments, assignment)
					}
				}
				if len(assignments) == 0 {
					continue
				}
				key := bucketKey(courseID, bucket.name)
				bucketParents := append(append([]string(nil), parents...), courseKeyValue)
				out = append(out, node{kind: nodeBucket, courseID: courseID, term: term, bucket: bucket.name, key: key, parents: bucketParents})
				if !m.taskIsOpen(key, bucket.open) {
					continue
				}
				for _, assignment := range assignments {
					taskNodeKey := taskKey(assignment.ID)
					taskParents := append(append([]string(nil), bucketParents...), key)
					out = append(out, node{kind: nodeTask, courseID: courseID, term: term, bucket: bucket.name, key: taskNodeKey, parents: taskParents, task: assignment})
					if !m.taskIsOpen(taskNodeKey, false) {
						continue
					}
					for _, material := range m.assignmentMaterials(assignment) {
						materialParents := append(append([]string(nil), taskParents...), taskNodeKey)
						out = append(out, node{kind: nodeTaskMaterial, courseID: courseID, term: term, bucket: bucket.name, key: taskMaterialKey(material.ID), parents: materialParents, mat: material, task: assignment})
					}
				}
			}
		}
	}
	return out
}

func (m *App) assignmentMaterials(assignment model.Assignment) []model.Material {
	prefix := "M" + strings.TrimPrefix(assignment.ID, "K") + ":"
	var materials []model.Material
	for _, material := range m.materials {
		if strings.HasPrefix(material.ID, prefix) {
			materials = append(materials, material)
		}
	}
	sort.Slice(materials, func(i, j int) bool { return materials[i].Name < materials[j].Name })
	return materials
}

func (m *App) courseByID(id int) (model.Course, bool) {
	for _, c := range m.courses {
		if c.ID == id {
			return c, true
		}
	}
	return model.Course{}, false
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
	byCourse := map[int][]model.Material{}
	for _, material := range m.materials {
		if m.tab == TabMaterials && m.filter != "all" {
			matches := m.filter == "new" && !material.Downloaded || m.filter == "downloaded" && material.Downloaded || m.filter == "link" && material.Kind == "url"
			if !matches {
				continue
			}
		}
		byCourse[material.CourseID] = append(byCourse[material.CourseID], material)
	}
	courseIDs := make([]int, 0, len(byCourse))
	for id := range byCourse {
		courseIDs = append(courseIDs, id)
	}
	sort.Ints(courseIDs)
	out := []node{}
	for _, courseID := range courseIDs {
		course := node{kind: nodeCourse, courseID: courseID, key: courseKey(courseID)}
		out = append(out, course)
		if !m.isOpen(course.key) {
			continue
		}
		buckets := []struct {
			label string
			keep  func(model.Material) bool
		}{{"downloaded", func(mt model.Material) bool { return mt.Downloaded }}, {"new", func(mt model.Material) bool { return !mt.Downloaded }}}
		for _, bucket := range buckets {
			var files []model.Material
			for _, material := range byCourse[courseID] {
				if bucket.keep(material) {
					files = append(files, material)
				}
			}
			if len(files) == 0 {
				continue
			}
			key := materialBucketKey(courseID, bucket.label)
			parents := []string{course.key}
			out = append(out, node{kind: nodeMaterialBucket, courseID: courseID, bucket: bucket.label, key: key, parents: parents})
			if !m.isOpen(key) {
				continue
			}
			sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
			for _, file := range files {
				fileNode := node{kind: nodeFile, courseID: courseID, mat: file, key: materialKey(file.ID), parents: []string{course.key, key}}
				out = append(out, fileNode)
				if assignment := m.assignmentForMaterial(file); assignment != nil && m.isOpen(fileNode.key) {
					out = append(out, node{kind: nodeMaterialAssignment, courseID: courseID, task: *assignment, key: "ma:" + assignment.ID + ":" + file.ID, parents: []string{course.key, key, fileNode.key}})
				}
			}
		}
	}
	return out
}

func materialBucketKey(courseID int, bucket string) string {
	return fmt.Sprintf("mb:%d:%s", courseID, bucket)
}

// materialNodeFiles lists the materials a row stands for: the file itself,
// every file in a state bucket, every file in a week, or every file in a
// course.
func (m *App) materialNodeFiles(n node) []model.Material {
	switch n.kind {
	case nodeFile:
		return []model.Material{n.mat}
	case nodeMaterialBucket:
		files := []model.Material{}
		for _, material := range m.courseFiles(n.courseID) {
			if n.bucket == "downloaded" && material.Downloaded || n.bucket == "new" && !material.Downloaded {
				files = append(files, material)
			}
		}
		return files
	case nodeWeek:
		return m.sectionFiles(n.courseID, n.mat.Section)
	case nodeCourse:
		return m.courseFiles(n.courseID)
	}
	return nil
}

// materialNodeIDs lists the material IDs a row stands for: the file itself,
// every file in a state bucket, or every file in a course/week.
func (m *App) materialNodeIDs(n node) []string {
	return idsOf(m.materialNodeFiles(n))
}

// materialNodeSelected reports whether every material behind a row is marked,
// so group rows show the same mark state as their file rows.
func (m *App) materialNodeSelected(n node) bool {
	if n.kind == nodeFile {
		return m.selectedM[n.mat.ID]
	}
	return allSelected(m.selectedM, m.materialNodeIDs(n))
}

func materialKey(id string) string { return "material:" + id }

func (m *App) assignmentForMaterial(material model.Material) *model.Assignment {
	if material.Kind != "attachment" || !strings.HasPrefix(material.ID, "M") {
		return nil
	}
	id := strings.TrimPrefix(strings.SplitN(material.ID, ":", 2)[0], "M")
	return m.byID("K" + id)
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

func treeConnectors() (string, string, string) {
	term := strings.ToLower(os.Getenv("TERM"))
	if term == "dumb" || strings.Contains(term, "ascii") {
		return "|  ", "|-- ", "`-- "
	}
	return "│  ", "├─ ", "╰─ "
}

func treeGuides(nodes []node, index int) string {
	vertical, branch, last := treeConnectors()
	depth := nodes[index].depth()
	var guides strings.Builder
	for slot := 0; slot < depth; slot++ {
		continues := false
		for next := index + 1; next < len(nodes); next++ {
			if nodes[next].depth() <= slot {
				break
			}
			if nodes[next].depth() == slot+1 {
				continues = true
				break
			}
		}
		if slot+1 < depth {
			if continues {
				guides.WriteString(vertical)
			} else {
				guides.WriteString("   ")
			}
			continue
		}
		if continues {
			guides.WriteString(branch)
		} else {
			guides.WriteString(last)
		}
	}
	return guides.String()
}

// taskRowRightPad keeps the tree's right edge (course counts, state labels)
// off the pane divider and the terminal edge. The padding is painted with the
// row highlight, so a cursor row still reaches the edge as one band.
const taskRowRightPad = 2

func (m *App) taskTreeRow(nodes []node, index, width int) string {
	n := nodes[index]
	selected := false
	if n.kind == nodeTask {
		selected = m.selected[n.task.ID]
	} else if m.isTaskGroup(n) {
		selected = allSelected(m.selected, m.taskGroupIDs(n))
	}
	h := rowHighlight{cursor: index == m.taskCursor, mark: selected}
	background := h.style
	prefix := h.gutter() + background(stFaint).Render(treeGuides(nodes, index))
	available := maxInt(1, width-lipgloss.Width(prefix)-taskRowRightPad)
	group := m.isTaskGroup(n)
	open := group && m.taskIsOpen(n.key, taskNodeDefaultOpen(n))
	content := ""
	counts := m.taskNodeCounts(n)
	switch n.kind {
	case nodeTerm:
		content = h.containerLead(open, selected) + background(stCode.Bold(true)).Render(n.term)
	case nodeCourse:
		name := m.shortCode(n.courseID)
		if course, ok := m.courseByID(n.courseID); ok && course.Short != "" {
			name = course.Short
		}
		content = h.containerLead(open, selected) + background(stCode.Bold(true)).Render(name)
	case nodeBucket:
		content = h.containerLead(open, selected) + background(stRow).Render(fmt.Sprintf("%s (%d)", n.bucket, len(m.taskGroupIDs(n))))
	case nodeTask:
		glyph := background(triageStyle(n.task.Triage)).Render(flagChar(n.task.Triage))
		marker := h.markCell(selected)
		caret := ""
		if group {
			caret = background(stFaint).Render(caretGlyph(open)) + background(stRow).Render(" ")
		}
		stateLabel := n.task.Triage.String()
		if n.task.Triage == model.Undecided {
			stateLabel = "new"
		}
		state := background(triageStyle(n.task.Triage)).Render(stateLabel)
		now := time.Now()
		age := dueAge(n.task, now)
		ageStyle := stRowDim
		if n.task.HasDue && n.task.Due.Before(now) {
			ageStyle = stOverdue
		}
		metadata := background(stRowDim).Render(" · "+n.task.ID+" · ") + background(ageStyle).Render(age)
		fixed := lipgloss.Width(glyph) + 1 + lipgloss.Width(marker) + 1 + lipgloss.Width(caret) + lipgloss.Width(state) + lipgloss.Width(metadata)
		nameWidth := maxInt(1, available-fixed-1)
		name := background(stRow).Render(ansi.Truncate(n.task.Name, nameWidth, "…"))
		space := background(stRow).Render(" ")
		content = glyph + space + marker + space + caret + name + space + state + metadata
	case nodeTaskMaterial:
		state, style := materialTreeState(n.mat)
		glyph := "○"
		if n.mat.Downloaded {
			glyph = "●"
		}
		label := ansi.Truncate(n.mat.Name, maxInt(1, available-lipgloss.Width(state)-5), "…")
		space := background(stRow).Render(" ")
		content = background(style).Render(glyph) + space + background(stFaint).Render("⁝") + space + background(stRow).Render(label) + space + background(style).Render(state)
	}

	if group {
		meta := ""
		if n.kind == nodeCourse {
			meta = formatTaskCounts(counts, h)
		}
		if !open {
			if meta != "" {
				meta += background(stRow).Render("  ")
			}
			meta += background(stRowDim).Render("⋯")
		}
		metaWidth := lipgloss.Width(meta)
		contentWidth := maxInt(1, available-metaWidth-1)
		content = ansiTruncate(content, contentWidth)
		if metaWidth > 0 {
			content += background(stRow).Render(strings.Repeat(" ", maxInt(1, available-lipgloss.Width(content)-metaWidth))) + meta
		}
	} else if n.kind == nodeTask {
		content = ansiTruncate(content, available)
	}
	row := prefix + content
	padding := strings.Repeat(" ", maxInt(0, width-lipgloss.Width(row)))
	if h.band() {
		padding = background(stRow).Render(padding)
	}
	return row + padding
}

func (m *App) isTaskGroup(n node) bool {
	return n.kind == nodeTerm || n.kind == nodeCourse || n.kind == nodeBucket || n.kind == nodeTask && len(m.assignmentMaterials(n.task)) > 0
}

func (m *App) taskNodeCounts(group node) [5]int {
	assignments := make([]model.Assignment, 0, 8)
	for _, id := range m.taskGroupIDs(group) {
		if assignment := m.byID(id); assignment != nil {
			assignments = append(assignments, *assignment)
		}
	}
	return tallyTriage(assignments)
}

// tallyTriage buckets assignments by decision state. Index 2 ("done") is kept
// for the five-glyph legend shape but no triage value maps to it.
func tallyTriage(assignments []model.Assignment) [5]int {
	var counts [5]int
	for _, assignment := range assignments {
		switch assignment.Triage {
		case model.Tracked:
			counts[1]++
		case model.Ignored:
			counts[3]++
		case model.Missed:
			counts[4]++
		default:
			counts[0]++
		}
	}
	return counts
}

// formatTaskCounts renders the per-state counts of a container. It takes the
// row's highlight so every count carries the band background: rendering the
// styled counts under a background wrapper loses it to the inner colour resets.
func formatTaskCounts(counts [5]int, h rowHighlight) string {
	states := []struct {
		count int
		glyph string
		style lipgloss.Style
	}{{counts[0], "○", stFlagNew}, {counts[1], "◐", stFlagT}, {counts[2], "●", stDone}, {counts[3], "◆", stFlagI}, {counts[4], "×", stFlagM}}
	var parts []string
	for _, state := range states {
		if state.count > 0 {
			parts = append(parts, h.style(state.style).Render(fmt.Sprintf("%s %d", state.glyph, state.count)))
		}
	}
	return strings.Join(parts, h.blanks(2))
}

func dueAge(assignment model.Assignment, now time.Time) string {
	if !assignment.HasDue {
		return "—"
	}
	dueDay := time.Date(assignment.Due.Year(), assignment.Due.Month(), assignment.Due.Day(), 0, 0, 0, 0, assignment.Due.Location())
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	days := int(dueDay.Sub(today).Hours() / 24)
	if days < 0 {
		return fmt.Sprintf("−%dd", -days)
	}
	return fmt.Sprintf("+%dd", days)
}

func materialTreeState(material model.Material) (string, lipgloss.Style) {
	if material.Downloaded {
		return "downloaded", stDone
	}
	return "new", stFlagNew
}

func (m *App) matTreeRow(nodes []node, index, width int, selected bool) string {
	n := nodes[index]
	h := rowHighlight{cursor: index == m.matCursor, mark: selected}
	background := h.style
	prefix := h.gutter() + background(stFaint).Render(treeGuides(nodes, index))
	parent := m.materialHasChildren(n)
	open := n.key != "" && m.isOpen(n.key)
	lead := ""
	if parent {
		lead = h.containerLead(open, selected)
	}
	content := ""
	switch n.kind {
	case nodeCourse:
		name := m.shortCode(n.courseID)
		if course, ok := m.courseByID(n.courseID); ok && course.Name != "" {
			name = course.Name
		}
		content = lead + background(stCode.Bold(true)).Render(name)
	case nodeMaterialBucket:
		count := 0
		for _, material := range m.courseFiles(n.courseID) {
			if (n.bucket == "downloaded") == material.Downloaded {
				count++
			}
		}
		content = lead + background(stRow).Render(fmt.Sprintf("%s (%d)", n.bucket, count))
	case nodeFile:
		glyph, stateStyle := materialTreeState(n.mat)
		state := "○"
		if glyph == "downloaded" {
			state = "●"
		}
		caret := ""
		if parent {
			caret = h.style(stFaint).Render(caretGlyph(open)) + h.style(stRow).Render(" ")
		}
		nameWidth := maxInt(1, width-lipgloss.Width(prefix)-12-lipgloss.Width(caret))
		content = background(stateStyle).Render(state) +
			background(stRow).Render(" ") + h.markCell(selected) + background(stRow).Render(" ") + caret +
			background(stRow).Render(ansi.Truncate(n.mat.Name, nameWidth, "…")) +
			background(stateStyle).Render(" "+glyph)
	case nodeWeek:
		content = lead + background(stCode.Bold(true)).Render(sectionLabel(n.mat.Section))
	case nodeMaterialAssignment:
		content = background(triageStyle(n.task.Triage)).Render(flagChar(n.task.Triage)) + background(stFaint).Render(" ⁝ ") + background(stRow).Render("assignment: "+n.task.Name) + background(stRowDim).Render(" · "+n.task.ID)
	}
	if (n.kind == nodeCourse || n.kind == nodeMaterialBucket || n.kind == nodeFile) && !open && parent {
		content += background(stRowDim).Render("  ⋯")
	}
	row := prefix + content
	row = ansiTruncate(row, width)
	return row + background(stRow).Render(strings.Repeat(" ", maxInt(0, width-lipgloss.Width(row))))
}

func (m *App) materialHasChildren(n node) bool {
	switch n.kind {
	case nodeCourse:
		return len(m.courseFiles(n.courseID)) > 0
	case nodeMaterialBucket:
		for _, material := range m.courseFiles(n.courseID) {
			if n.bucket == "downloaded" && material.Downloaded || n.bucket == "new" && !material.Downloaded {
				return true
			}
		}
	case nodeFile:
		return m.assignmentForMaterial(n.mat) != nil
	}
	return false
}
