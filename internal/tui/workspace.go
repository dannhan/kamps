package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ruyam/kamps/internal/model"
)

func (m *App) sidebarCourses() []model.Course {
	courses := append([]model.Course(nil), m.courses...)
	known := map[int]bool{}
	for _, c := range courses {
		known[c.ID] = true
	}
	add := func(id int) {
		if !known[id] {
			courses = append(courses, model.Course{ID: id, Name: fmt.Sprintf("Course %d", id)})
			known[id] = true
		}
	}
	for _, a := range m.assignments {
		add(a.CourseID)
	}
	for _, a := range m.materials {
		add(a.CourseID)
	}
	if len(m.unknownFiles) > 0 {
		courses = append(courses, model.Course{ID: localCourseID, Name: "Local library"})
	}
	sort.SliceStable(courses, func(i, j int) bool { return courses[i].Name < courses[j].Name })
	return courses
}
func (m *App) courseLabel() string {
	if !m.courseScoped {
		return "All courses"
	}
	for _, c := range m.sidebarCourses() {
		if c.ID == m.courseID {
			if c.Name != "" {
				return c.Name
			}
			return c.Short
		}
	}
	return m.shortCode(m.courseID)
}
func (m *App) cyclePane(direction int) {
	if m.tab != TabTasks && m.tab != TabMaterials {
		return
	}
	// Traversal order is courses, list, inspector.
	order := []int{1, 0}
	if m.detailVisible() {
		order = append(order, 2)
	}
	index := 0
	for i, p := range order {
		if p == m.paneFocus {
			index = i
		}
	}
	m.paneFocus = order[(index+direction+len(order))%len(order)]
}
func (m *App) chooseCourse() {
	courses := m.sidebarCourses()
	previousCourseID := m.courseID
	m.courseID = 0
	m.courseScoped = m.courseCursor > 0
	if m.courseCursor > 0 && m.courseCursor <= len(courses) {
		m.courseID = courses[m.courseCursor-1].ID
	}
	m.taskCursor = 0
	m.matCursor = 0
	m.detailScroll = 0
	m.paneFocus = 0
	if m.courseID == localCourseID {
		m.tabFilters[TabMaterials] = "unknown"
	} else if previousCourseID == localCourseID && m.tabFilters[TabMaterials] == "unknown" {
		m.tabFilters[TabMaterials] = "all"
	}
	if m.tab == TabMaterials {
		m.filter = m.tabFilters[TabMaterials]
	}
	m.clampCursor()
	m.cancelTriageArm()
	m.visual = false
}
func (m *App) sidebar(width, height int) []string {
	labels := []string{"All courses"}
	for _, c := range m.sidebarCourses() {
		name := c.Name
		if name == "" {
			name = c.Short
		}
		labels = append(labels, name)
	}
	rows := []string{stFaint.Render(" Courses")}
	start, end := windowRange(len(labels), m.courseCursor, maxInt(1, height-1))
	for i := start; i < end; i++ {
		active := i == 0 && !m.courseScoped
		if i > 0 {
			active = m.courseScoped && m.sidebarCourses()[i-1].ID == m.courseID
		}
		marker := "  "
		if active {
			marker = "· "
		}
		style := stDim
		if i > 0 {
			style = m.courseStyle(m.sidebarCourses()[i-1].ID)
		}
		if active {
			style = style.Bold(true)
		}
		rows = append(rows, paneRow(style.Render(pad(marker+labels[i], width-1, false)), width, rowHighlight{cursor: m.paneFocus == 1 && i == m.courseCursor}))
	}
	return rows
}
func readableDeadline(a model.Assignment, m *App) string {
	if !a.HasDue {
		return "No deadline"
	}
	days := dueDays(a, m.currentTime())
	if days < 0 {
		if a.Triage != model.Undecided {
			return fmt.Sprintf("%dd ago", -days)
		}
		return fmt.Sprintf("%dd overdue", -days)
	}
	if days == 0 {
		if a.Overdue(m.currentTime()) {
			return "Overdue today"
		}
		return "Today"
	}
	return fmt.Sprintf("In %dd", days)
}
func cleanMaterialLabel(name string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(name, "_", " ")), " ")
}

// materialBlock keeps each file's identity and state together, even on narrow terminals.
func (m *App) materialBlock(n node, index, width int) []string {
	if n.kind == nodeSpacer {
		return []string{fitLine("", width)}
	}
	marked := m.materialNodeSelected(n)
	h := rowHighlight{cursor: m.paneFocus == 0 && index == m.matCursor, mark: marked,
		partial: !marked && partialSelected(m.selectedM, m.materialNodeIDs(n))}
	depth := len(n.parents)
	prefix := h.gutter() + h.markCell() + h.blanks(1+2*depth)
	caret := "  "
	if m.materialHasChildren(n) {
		caret = caretGlyph(m.isOpen(n.key)) + " "
	}
	prefix += h.style(stDim).Render(caret)
	indent := lipgloss.Width(prefix)
	available := maxInt(1, width-indent-1)
	title, meta := "", ""
	titleStyle, metaStyle := h.primaryStyle(), stFaint
	switch n.kind {
	case nodeCourse:
		title = m.fullCourseName(n.courseID)
		if n.courseID == localCourseID {
			title = "Local library"
		}
		titleStyle = h.style(m.courseStyle(n.courseID)).Bold(true)
	case nodeMaterialWeek, nodeMaterialSection:
		title = strings.Replace(n.section, "week-", "Week ", 1)
		count := len(m.materialNodeFiles(n))
		meta = fmt.Sprintf("%d %s", count, plural(count, "file", "files"))
		titleStyle = h.style(stDim).Bold(true)
		if n.kind == nodeMaterialWeek {
			courses := map[int]bool{}
			for _, file := range m.materialNodeFiles(n) {
				courses[file.CourseID] = true
			}
			meta += fmt.Sprintf(" · %d %s", len(courses), plural(len(courses), "course", "courses"))
			titleStyle = h.style(stSectionTitle)
		}
	case nodeMaterialFolder:
		title = cleanMaterialLabel(n.mat.Folder) + "/"
	case nodeFile:
		title = cleanMaterialLabel(n.mat.Name)
		kind := strings.TrimPrefix(strings.ToUpper(filepath.Ext(n.mat.Name)), ".")
		if len(kind) > 8 {
			kind = ""
		}
		state := "Not downloaded"
		switch {
		case m.downloadErrors[n.mat.ID] != "":
			state, metaStyle = "Failed · d retry", stOverdue
		case m.downloads[n.mat.ID]:
			state, metaStyle = strings.TrimSpace("Downloading "+m.downloadProgress[n.mat.ID]), stKey
		case n.mat.Downloaded:
			state, metaStyle = "Local", stDone
		case n.mat.Kind == "url" || n.mat.Kind == "page" || n.mat.Kind == "book" || n.mat.Kind == "praktikum":
			state = "LMS link"
		}
		meta = state
		if kind != "" {
			meta = kind + " · " + state
		}
	}
	// Counts and file states sit next to their labels, never at the far edge.
	if lipgloss.Width(title)+2+lipgloss.Width(meta) <= available {
		content := prefix + titleStyle.Render(title)
		if meta != "" {
			content += h.blanks(2) + h.style(metaStyle).Render(meta)
		}
		return []string{content + h.blanks(maxInt(0, width-lipgloss.Width(content)))}
	}
	titleLines := wrapCell(title, available)
	if len(titleLines) > 2 {
		titleLines = titleLines[:2]
		titleLines[1] = truncate(titleLines[1], maxInt(1, available-1)) + "…"
	}
	rows := []string{}
	for i, line := range titleLines {
		rowPrefix := h.blanks(indent)
		if i == 0 {
			rowPrefix = prefix
		}
		rows = append(rows, fitLine(rowPrefix+titleStyle.Render(pad(line, available, false))+h.gap(), width))
	}
	if meta != "" {
		rows = append(rows, fitLine(h.blanks(indent)+h.style(metaStyle).Render(pad(meta, available, false))+h.gap(), width))
	}
	return rows
}

func (m *App) materialListWindow(nodes []node, width, height int) ([]string, int, int) {
	blocks := make([][]string, len(nodes))
	for i, n := range nodes {
		blocks[i] = m.materialBlock(n, i, width)
		if height > 0 && len(blocks[i]) > height {
			blocks[i] = blocks[i][:height]
		}
	}
	return m.courseContextWindow(nodes, blocks, m.matCursor, width, height)
}

func (m *App) quietMaterialRows(nodes []node, width, height int) []string {
	rows, _, _ := m.materialListWindow(nodes, width, height)
	if len(nodes) == 0 {
		return []string{fitLine(stEmpty.Render(" No materials match · f filters · / search"), width)}
	}
	return rows
}

func workspaceSidebarWidth(width int) int {
	if width < 100 {
		return 0
	}
	return clampInt(width/5, 23, 32)
}

func (m *App) workspaceContent(width, height int) (string, string, []string, []string) {
	sidebarWidth := workspaceSidebarWidth(width)
	mainWidth := width
	if sidebarWidth > 0 {
		mainWidth -= sidebarWidth + 1
	}
	nodes, cursor := m.taskNodes(), m.taskCursor
	if m.tab == TabMaterials {
		nodes, cursor = m.matNodes(), m.matCursor
	}
	listWidth := mainWidth
	if m.detailMode() == detailSplit {
		listWidth, _ = splitWidths(mainWidth)
	}
	var rows []string
	if m.tab == TabTasks {
		rows = m.taskRows(nodes, listWidth, height)
	} else {
		rows = m.quietMaterialRows(nodes, listWidth, height)
	}
	if m.detailMode() == detailReplace && m.paneFocus == 2 {
		rows = m.detailRows(nodes, cursor, mainWidth, height)
	} else if m.detailMode() == detailSplit {
		rows = m.splitTreeRows(rows, nodes, cursor, mainWidth, height)
	}
	if sidebarWidth > 0 {
		sidebar := m.sidebar(sidebarWidth, height)
		combined := make([]string, 0, height)
		for i := 0; i < height; i++ {
			left, right := "", ""
			if i < len(sidebar) {
				left = sidebar[i]
			}
			if i < len(rows) {
				right = rows[i]
			}
			combined = append(combined, pad(left, sidebarWidth, false)+" "+fitLine(right, mainWidth))
		}
		rows = combined
	} else if m.paneFocus == 1 {
		rows = m.sidebar(width, height)
	}
	title := m.courseLabel() + " · " + m.filter
	if m.tab == TabTasks {
		title += " · Current work first"
	}
	if m.searchActive() {
		title = m.searchTitle()
	}
	count := len(m.visible())
	if m.tab == TabMaterials {
		count = len(m.visibleMaterials())
	}
	right := fmt.Sprintf("%d %s", count, plural(count, "item", "items"))
	if m.searchActive() {
		right = fmt.Sprintf("%d matches", count)
	}
	var start, end int
	if m.tab == TabTasks {
		_, start, end = m.taskListWindow(nodes, listWidth, height)
	} else {
		_, start, end = m.materialListWindow(nodes, listWidth, height)
	}
	if start > 0 {
		right += " · ↑ more"
	}
	if end < len(nodes) {
		right += " · ↓ more"
	}

	if !m.searchActive() || m.triageArmed || m.statusErr != "" {
		status := m.footerSummary()
		if m.toast != "" {
			status = rightAligned(m.selectionLabel(), m.toast)
		}
		if m.statusErr != "" {
			status = m.statusErr + " · esc dismiss"
		}
		if m.triageArmed {
			status = m.triagePrompt()
		}
		if status != "" {
			title = status
			right = ""
		}
	}
	if m.tab == TabMaterials {
		right = rightAligned(right, m.materialSortLabel())
	}
	hints := m.footerHintItems()
	if m.filter != "all" || m.search != "" {
		hints = append(hints, "esc clear")
	}
	return title, right, hints, rows
}
func (m *App) filterOptions() []string {
	if m.tab == TabMaterials {
		return materialFilterNames
	}
	return filterNames
}
func (m *App) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	options := m.filterOptions()
	switch msg.String() {
	case "esc", "f":
		m.filterOn = false
	case "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		m.filterCursor = clampInt(m.filterCursor-1, 0, len(options)-1)
	case "down", "j":
		m.filterCursor = clampInt(m.filterCursor+1, 0, len(options)-1)
	case "enter":
		m.setPaletteFilter(options[m.filterCursor])
		m.filterOn = false
		m.cancelTriageArm()
		m.visual = false
	}
	return m, nil
}
func (m *App) filterRows(width int) []string {
	descriptions := map[string]string{"attention": "Undecided work, including overdue and undated", "upcoming": "Undecided, due today through the next 7 days", "closed": "Missed or ignored; tracked remains separate"}
	rows := []string{}
	options := m.filterOptions()
	start, end := windowRange(len(options), m.filterCursor, maxInt(1, bodyHeightFor(m.width, m.height)))
	for i := start; i < end; i++ {
		f := options[i]
		label := f
		if d := descriptions[f]; d != "" {
			label += " · " + d
		}
		rows = append(rows, paneRow(label, width, rowHighlight{cursor: i == m.filterCursor}))
	}
	return rows
}
