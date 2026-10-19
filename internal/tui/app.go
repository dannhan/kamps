// Package tui is the Bubble Tea application: a small, cheerful terminal UI
// over a source.Source.
//
// Architecture (The Elm Architecture):
//
//	Init   -> kick off async work (load data, start animations)
//	Update -> react to messages (keys, window resize, ticks, loaded data)
//	View   -> render the model to a string
//
// The screen is deliberately NOT full-screen: it renders inline, capped at
// maxBoxW x maxBoxH and shrinking to fit smaller terminals, so your shell
// history stays visible above it.
package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/source"
)

// ---- top-level navigation ----

// Tab is one top-level screen.
type Tab int

const (
	TabTasks Tab = iota
	TabMaterials
	TabSync
	TabCommit
	tabCount
)

func (t Tab) String() string {
	switch t {
	case TabTasks:
		return "Tasks"
	case TabMaterials:
		return "Materials"
	case TabSync:
		return "Sync"
	case TabCommit:
		return "Commit"
	}
	return "?"
}

// Layout is how the Tasks screen is drawn. All four share the same data and
// keys; they are just different ways of looking at the same backlog.
type Layout int

const (
	LayList      Layout = iota // two-pane: list + detail (the default)
	LayDashboard               // backlog overview: counts, per-course, oldest first
	LayBoard                   // 2x2 buckets by decision
	LayWizard                  // guided review & commit
	layCount
)

func (l Layout) String() string {
	switch l {
	case LayList:
		return "list"
	case LayDashboard:
		return "dashboard"
	case LayBoard:
		return "board"
	case LayWizard:
		return "wizard"
	}
	return "?"
}

// viewCycle is what `v` steps through. The board and the wizard are not in the
// daily rotation: the wizard opens with `c`, the board is parked for now.
var viewCycle = []Layout{LayDashboard, LayList}

// nextLayout returns the view `v` should switch to from the current one.
func (l Layout) next() Layout {
	for i, cand := range viewCycle {
		if cand == l {
			return viewCycle[(i+1)%len(viewCycle)]
		}
	}
	return viewCycle[0]
}

// inViewCycle reports whether a layout is reachable with `v`.
func (l Layout) inViewCycle() bool {
	for _, cand := range viewCycle {
		if cand == l {
			return true
		}
	}
	return false
}

// filters, cycled with "/".
var filterNames = []string{"all", "undecided", "overdue", "tracked", "missed", "ignored"}
var materialFilterNames = []string{"all", "new", "downloaded", "link"}

// ---- model ----

// App is the root model.
type App struct {
	src  source.Source
	cfg  config.Config
	ver  string
	keys KeyMap
	help help.Model

	courses     []model.Course
	assignments []model.Assignment
	materials   []model.Material
	loaded      bool
	loadError   string

	tab    Tab
	layout Layout
	filter string

	taskCursor   int
	matCursor    int
	syncCursor   int
	commitCursor int
	boardCol     int
	boardCursor  [4]int // per triage column

	tabFilters  [tabCount]string
	tabExpanded [tabCount]map[string]bool
	returnTab   Tab

	// expanded holds the fold state of the tree panes, keyed by courseKey /
	// weekKey. A missing key means open — collapsing is the explicit act.
	expanded map[string]bool

	selected  map[string]bool // assignment id -> checked
	selectedM map[string]bool // material id -> checked
	pending   map[string]model.Triage

	// wizard
	wizStep   int // 0 select, 1 action, 2 review, 3 done
	wizAction model.Triage

	// returnLayout is where `esc` goes back to when the wizard closes.
	returnLayout Layout

	spinner    spinner.Model
	syncing    bool
	syncTicks  int
	toast      string
	toastTicks int

	width  int
	height int
	helpOn bool
	dryRun bool
	noAnim bool
}

// New builds the root model.
func New(src source.Source, cfg config.Config, version string) *App {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(themeTokens.Accent)

	h := help.New()

	return &App{
		src:          src,
		cfg:          cfg,
		ver:          version,
		keys:         DefaultKeys(),
		help:         h,
		filter:       "all",
		layout:       LayList,
		returnTab:    TabTasks,
		tabFilters:   [tabCount]string{"all", "all", "all", "all"},
		tabExpanded:  [tabCount]map[string]bool{0: {}, 1: {}, 2: {}, 3: {}},
		returnLayout: LayList,
		expanded:     map[string]bool{},
		selected:     map[string]bool{},
		selectedM:    map[string]bool{},
		pending:      map[string]model.Triage{},
		spinner:      sp,
		wizAction:    model.Tracked,
		width:        100,
		height:       30,
		dryRun:       true,
		noAnim:       cfg.NoAnim,
	}
}

// ---- messages ----

type loadedMsg struct {
	courses     []model.Course
	assignments []model.Assignment
	materials   []model.Material
}

type errMsg struct{ err error }

func loadCmd(src source.Source) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		cs, err := src.Courses(ctx)
		if err != nil {
			return errMsg{err}
		}
		as, err := src.Assignments(ctx)
		if err != nil {
			return errMsg{err}
		}
		ms, err := src.Materials(ctx)
		if err != nil {
			return errMsg{err}
		}
		return loadedMsg{cs, as, ms}
	}
}

func (m *App) Init() tea.Cmd { return loadCmd(m.src) }

// ---- update ----

func (m *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.help.Width = msg.Width
		return m, nil

	case loadedMsg:
		m.loaded = true
		m.loadError = ""
		m.courses = msg.courses
		m.assignments = msg.assignments
		m.materials = msg.materials
		sort.Slice(m.courses, func(i, j int) bool { return m.courses[i].Name < m.courses[j].Name })
		sort.Slice(m.materials, func(i, j int) bool {
			a, b := m.materials[i], m.materials[j]
			if a.CourseCode != b.CourseCode {
				return a.CourseCode < b.CourseCode
			}
			if a.Section != b.Section {
				return a.Section < b.Section
			}
			return a.Name < b.Name
		})
		m.clampCursor()
		return m, nil

	case errMsg:
		m.loaded = true
		m.loadError = msg.err.Error()
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		m.advance()
		if m.animating() {
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *App) switchTab(tab Tab) {
	if tab < 0 || tab >= tabCount {
		return
	}
	m.tabFilters[m.tab] = m.filter
	m.tabExpanded[m.tab] = m.expanded
	if tab == TabCommit && m.tab != TabCommit {
		m.returnTab = m.tab
	}
	m.tab = tab
	m.filter = m.tabFilters[tab]
	if m.filter == "" {
		m.filter = "all"
	}
	if m.tabExpanded[tab] != nil {
		m.expanded = m.tabExpanded[tab]
	} else {
		m.tabExpanded[tab] = m.expanded
	}
	m.clampCursor()
}

func (m *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.helpOn {
		switch msg.String() {
		case "?", "esc", "q":
			m.helpOn = false
			return m, nil
		case "ctrl+c":
			return m, tea.Quit
		}
		return m, nil
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "?":
		m.helpOn = true
	case "tab":
		m.switchTab((m.tab + 1) % tabCount)
	case "shift+tab":
		m.switchTab((m.tab + tabCount - 1) % tabCount)
	case "1", "2", "3", "4":
		m.switchTab(Tab(msg.String()[0] - '1'))
	case "up", "k":
		m.move(-1)
	case "down", "j":
		m.move(1)
	case "left", "h":
		m.moveX(-1)
	case "right", "l":
		m.moveX(1)
	case " ":
		m.toggleSelect()
	case "a":
		m.selectAll()
	case "t":
		m.applyTriage(model.Tracked)
	case "m":
		m.applyTriage(model.Missed)
	case "i":
		m.applyTriage(model.Ignored)
	case "u":
		m.applyTriage(model.Undecided)
	case "c":
		m.openCommit()
	case "esc":
		if m.tab == TabCommit {
			m.closeCommit()
		} else {
			m.back()
		}
	case "enter":
		if m.tab == TabCommit {
			m.toast = "preview only · nothing written"
			m.toastTicks = 24
		} else {
			m.enter()
		}
	case "/":
		m.cycleFilter()
	case "s":
		return m, m.startSync()
	case "d":
		m.download()
	}
	return m, nil
}

// ---- navigation ----

func (m *App) move(delta int) {
	switch m.tab {
	case TabTasks:
		if m.layout == LayBoard {
			items := m.boardItems(m.boardCol)
			m.boardCursor[m.boardCol] = clampInt(m.boardCursor[m.boardCol]+delta, 0, maxInt(0, len(items)-1))
			return
		}
		n := len(m.focusedList())
		if m.layout == LayList {
			n = len(m.taskNodes())
		}
		m.taskCursor = clampInt(m.taskCursor+delta, 0, maxInt(0, n-1))
	case TabMaterials:
		m.matCursor = clampInt(m.matCursor+delta, 0, maxInt(0, len(m.matNodes())-1))
	case TabSync:
		m.syncCursor = clampInt(m.syncCursor+delta, 0, 5)
	case TabCommit:
		m.commitCursor = clampInt(m.commitCursor+delta, 0, maxInt(0, len(m.commitItems())-1))
	}
}

// clampCursor keeps the per-tab cursors inside whatever the current view draws.
// Views with different row models — dashboard items, tree nodes — share the
// same cursor, and a stale index must never point past the end.
func (m *App) clampCursor() {
	switch {
	case m.tab == TabTasks && m.layout == LayList:
		m.taskCursor = clampInt(m.taskCursor, 0, maxInt(0, len(m.taskNodes())-1))
	case m.tab == TabMaterials:
		m.matCursor = clampInt(m.matCursor, 0, maxInt(0, len(m.matNodes())-1))
	case m.tab == TabTasks && m.layout != LayBoard:
		m.taskCursor = clampInt(m.taskCursor, 0, maxInt(0, len(m.focusedList())-1))
	case m.tab == TabSync:
		m.syncCursor = clampInt(m.syncCursor, 0, 5)
	case m.tab == TabCommit:
		m.commitCursor = clampInt(m.commitCursor, 0, maxInt(0, len(m.commitItems())-1))
	}
}

// moveX is horizontal: board columns on the Tasks board, folding everywhere the
// trees are. `l` opens the cursor's container, `h` closes it.
func (m *App) moveX(delta int) {
	var nodes []node
	var cursor *int
	switch m.tab {
	case TabTasks:
		if m.layout == LayBoard {
			m.boardCol = clampInt(m.boardCol+delta, 0, len(boardOrder)-1)
			return
		}
		if m.layout != LayList {
			return
		}
		nodes, cursor = m.taskNodes(), &m.taskCursor
	case TabMaterials:
		nodes, cursor = m.matNodes(), &m.matCursor
	default:
		return
	}
	if *cursor < 0 || *cursor >= len(nodes) {
		return
	}
	n := nodes[*cursor]
	if delta > 0 {
		if n.key != "" && (m.isTaskGroup(n) || m.materialHasChildren(n)) {
			m.expanded[n.key] = true
			m.tabExpanded[m.tab] = m.expanded
		}
		return
	}
	// h/left: fold the cursor's container while it actually shows children,
	// keeping the highlight on it; on leaves and already-folded containers,
	// walk up to the parent row instead.
	if n.key != "" && (m.isTaskGroup(n) || m.materialHasChildren(n)) && hasVisibleChildren(nodes, n.key) {
		m.expanded[n.key] = false
		m.tabExpanded[m.tab] = m.expanded
		return
	}
	if len(n.parents) == 0 {
		return
	}
	parentKey := n.parents[len(n.parents)-1]
	for i, candidate := range nodes {
		if candidate.key == parentKey {
			*cursor = i
			return
		}
	}
}

// hasVisibleChildren reports whether the flat node list currently renders any
// child rows for the node with the given key, i.e. whether folding it would
// hide anything at all right now.
func hasVisibleChildren(nodes []node, key string) bool {
	for _, n := range nodes {
		for _, parent := range n.parents {
			if parent == key {
				return true
			}
		}
	}
	return false
}

func (m *App) toggleSelect() {
	switch m.tab {
	case TabTasks:
		if m.layout == LayList {
			nodes := m.taskNodes()
			if m.taskCursor < 0 || m.taskCursor >= len(nodes) {
				return
			}
			n := nodes[m.taskCursor]
			if n.kind == nodeTaskMaterial {
				return
			}
			if n.kind == nodeTask {
				toggleSet(m.selected, []string{n.task.ID})
				return
			}
			toggleSet(m.selected, m.taskGroupIDs(n))
			return
		}
		if id := m.cursorAssignmentID(); id != "" {
			toggleSet(m.selected, []string{id})
		}
	case TabMaterials:
		nodes := m.matNodes()
		if m.matCursor < 0 || m.matCursor >= len(nodes) {
			return
		}
		n := nodes[m.matCursor]
		ids := []string{}
		switch n.kind {
		case nodeFile:
			ids = []string{n.mat.ID}
		case nodeMaterialBucket:
			for _, material := range m.courseFiles(n.courseID) {
				if n.bucket == "downloaded" && material.Downloaded || n.bucket == "new" && !material.Downloaded {
					ids = append(ids, material.ID)
				}
			}
		case nodeCourse:
			ids = idsOf(m.courseFiles(n.courseID))
		}
		toggleSet(m.selectedM, ids)
	}
}

func (m *App) selectAll() {
	switch m.tab {
	case TabTasks:
		for _, a := range m.focusedList() {
			m.selected[a.ID] = true
		}
	case TabMaterials:
		for _, mt := range m.materials {
			m.selectedM[mt.ID] = true
		}
	}
}

func (m *App) cycleFilter() {
	var names []string
	switch m.tab {
	case TabTasks:
		names = filterNames
	case TabMaterials:
		names = materialFilterNames
	default:
		return
	}
	cur := 0
	for i, f := range names {
		if f == m.filter {
			cur = i
		}
	}
	m.filter = names[(cur+1)%len(names)]
	m.tabFilters[m.tab] = m.filter
	if m.tab == TabTasks {
		m.taskCursor = 0
	} else {
		m.matCursor = 0
	}
	m.toast = "filter: " + m.filter
	m.toastTicks = 20
}

// ---- task data ----

var boardOrder = []model.Triage{model.Undecided, model.Tracked, model.Missed, model.Ignored}

// visible returns the Tasks-tab assignments after the active filter, sorted.
func (m *App) visible() []model.Assignment {
	now := time.Now()
	out := make([]model.Assignment, 0, len(m.assignments))
	for _, a := range m.assignments {
		if !m.matchesFilter(a, now) {
			continue
		}
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return lessAssignment(out[i], out[j]) })
	return out
}

func (m *App) matchesFilter(a model.Assignment, now time.Time) bool {
	switch m.filter {
	case "undecided":
		return a.Triage == model.Undecided
	case "overdue":
		return a.Overdue(now)
	case "tracked":
		return a.Triage == model.Tracked
	case "missed":
		return a.Triage == model.Missed
	case "ignored":
		return a.Triage == model.Ignored
	}
	return true
}

// boardItems returns the assignments in one triage column, sorted.
func (m *App) boardItems(col int) []model.Assignment {
	if col < 0 || col >= len(boardOrder) {
		return nil
	}
	want := boardOrder[col]
	out := make([]model.Assignment, 0, 4)
	for _, a := range m.assignments {
		if a.Triage == want {
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool { return lessAssignment(out[i], out[j]) })
	return out
}

// ---- dashboard (today) data ----

// dashContentHeight is how many rows the dashboard's today pane has to show
// items in. dashAlloc and the renderer both read it, so what is drawn and what
// the keyboard walks can never disagree.
func (m *App) dashContentHeight() int {
	return clampInt(m.boxHeight()-6, 5, 40) // tab bar + pane chrome
}

// dashAlloc decides how many items each today section shows. Every non-empty
// section is guaranteed a row, then whatever is left goes to the most urgent
// section first — so a pile of overdue work can never hide what is coming up.
func (m *App) dashAlloc(overdue, dueToday, soon []model.Assignment) [3]int {
	items := [3][]model.Assignment{overdue, dueToday, soon}

	content := m.dashContentHeight()
	left := maxInt(3, content-5) // three section headers + two spacer lines

	var alloc [3]int
	for i := range items {
		if len(items[i]) > 0 && left > 0 {
			alloc[i] = 1
			left--
		}
	}
	for i := range items {
		if grow := minInt(len(items[i])-alloc[i], left); grow > 0 {
			alloc[i] += grow
			left -= grow
		}
	}
	return alloc
}

// todayBuckets classifies the still-undecided assignments the dashboard shows:
// past due, due today, and the next three days. Decided items are deliberately
// absent — the dashboard is about what still needs you.
func (m *App) todayBuckets() (overdue, dueToday, soon []model.Assignment) {
	now := time.Now()
	for _, a := range m.assignments {
		switch {
		case a.Overdue(now):
			overdue = append(overdue, a)
		case a.Triage != model.Undecided:
			// decided: not the dashboard's business
		case a.HasDue && sameDay(a.Due, now):
			dueToday = append(dueToday, a)
		case a.HasDue && a.Due.After(now) && a.Due.Before(now.AddDate(0, 0, 4)):
			soon = append(soon, a)
		}
	}
	sortSlice(overdue)
	sortSlice(dueToday)
	sortSlice(soon)
	return
}

// dashShown is the ordered list of items the dashboard cursor walks, matching
// exactly what the today panel draws.
func (m *App) dashShown() []model.Assignment {
	overdue, dueToday, soon := m.todayBuckets()
	alloc := m.dashAlloc(overdue, dueToday, soon)
	out := []model.Assignment{}
	for i, items := range [3][]model.Assignment{overdue, dueToday, soon} {
		out = append(out, items[:shownCount(alloc[i], len(items))]...)
	}
	return out
}

// focusedList is the assignment list the keyboard is currently driving. Every
// Tasks layout exposes its rows through this, so Up/Down, space and t/m/i/u
// behave the same everywhere.
func (m *App) focusedList() []model.Assignment {
	if m.tab != TabTasks {
		return nil
	}
	switch m.layout {
	case LayBoard:
		return m.boardItems(m.boardCol)
	case LayDashboard:
		return m.dashShown()
	case LayList:
		var out []model.Assignment
		for _, n := range m.taskNodes() {
			if n.kind == nodeTask {
				out = append(out, n.task)
			}
		}
		return out
	default:
		return m.visible()
	}
}

// cursorAssignmentID is the item the keyboard is pointing at, or "". On the
// tree, the cursor may rest on a course row — that is no single assignment.
func (m *App) cursorAssignmentID() string {
	if m.tab != TabTasks {
		return ""
	}
	if m.layout == LayList {
		nodes := m.taskNodes()
		if m.taskCursor >= 0 && m.taskCursor < len(nodes) && nodes[m.taskCursor].kind == nodeTask {
			return nodes[m.taskCursor].task.ID
		}
		return ""
	}
	items := m.focusedList()
	cursor := m.taskCursor
	if m.layout == LayBoard && m.tab == TabTasks {
		cursor = m.boardCursor[m.boardCol]
	}
	if cursor >= 0 && cursor < len(items) {
		return items[cursor].ID
	}
	return ""
}

// targetIDs is what a triage key acts on: the selection, else the cursor item.
// On the tree, a course row targets all its visible children.
func (m *App) targetIDs() []string {
	var ids []string
	for _, a := range m.assignments {
		if m.selected[a.ID] {
			ids = append(ids, a.ID)
		}
	}
	if len(ids) > 0 {
		return ids
	}
	if m.tab == TabTasks && m.layout == LayList {
		if id := m.cursorAssignmentID(); id != "" {
			return []string{id}
		}
		nodes := m.taskNodes()
		if m.taskCursor >= 0 && m.taskCursor < len(nodes) && m.isTaskGroup(nodes[m.taskCursor]) {
			return m.taskGroupIDs(nodes[m.taskCursor])
		}
		return nil
	}
	if id := m.cursorAssignmentID(); id != "" {
		return []string{id}
	}
	return ids
}

func (m *App) byID(id string) *model.Assignment {
	for i := range m.assignments {
		if m.assignments[i].ID == id {
			return &m.assignments[i]
		}
	}
	return nil
}

// applyTriage sets the decision on the target items and queues them for commit.
func (m *App) applyTriage(t model.Triage) {
	if m.tab != TabTasks {
		return
	}
	ids := m.targetIDs()
	if len(ids) == 0 {
		return
	}
	for _, id := range ids {
		if a := m.byID(id); a != nil {
			a.Triage = t
		}
		if t == model.Undecided {
			delete(m.pending, id)
		} else {
			m.pending[id] = t
		}
	}
	m.selected = map[string]bool{}
	m.toast = fmt.Sprintf("%d %s %s", len(ids), plural(len(ids), "item", "items"), t.String())
	m.toastTicks = 24
}

// pendingCounts returns how many uncommitted decisions of each kind are queued.
func (m *App) pendingCounts() (tracked, missed, ignored int) {
	for _, t := range m.pending {
		switch t {
		case model.Tracked:
			tracked++
		case model.Missed:
			missed++
		case model.Ignored:
			ignored++
		}
	}
	return
}

func (m *App) triageCounts() (undecided, tracked, missed, ignored int) {
	for _, a := range m.assignments {
		switch {
		case a.Triage == model.Tracked:
			tracked++
		case a.Triage == model.Missed:
			missed++
		case a.Triage == model.Ignored:
			ignored++
		default:
			undecided++
		}
	}
	return
}

func (m *App) overdueCount() int {
	now := time.Now()
	n := 0
	for _, a := range m.assignments {
		if a.Overdue(now) {
			n++
		}
	}
	return n
}

// ---- materials ----

func (m *App) download() {
	if m.tab != TabMaterials {
		m.toast = "switch to Materials to download"
		m.toastTicks = 20
		return
	}
	// The selection first; with nothing selected, the cursor does the work —
	// on a course or week row, d fetches everything underneath it.
	var targets []string
	for _, mt := range m.materials {
		if m.selectedM[mt.ID] {
			targets = append(targets, mt.ID)
		}
	}
	if len(targets) == 0 {
		nodes := m.matNodes()
		if m.matCursor >= 0 && m.matCursor < len(nodes) {
			var files []model.Material
			n := nodes[m.matCursor]
			switch n.kind {
			case nodeFile:
				files = []model.Material{n.mat}
			case nodeMaterialBucket:
				for _, material := range m.courseFiles(n.courseID) {
					if n.bucket == "downloaded" && material.Downloaded || n.bucket == "new" && !material.Downloaded {
						files = append(files, material)
					}
				}
			case nodeCourse:
				files = m.courseFiles(n.courseID)
			}
			for _, f := range files {
				if !f.Downloaded {
					targets = append(targets, f.ID)
				}
			}
		}
	}
	if len(targets) == 0 {
		m.toast = "nothing new to download"
		m.toastTicks = 20
		return
	}
	for i := range m.materials {
		for _, id := range targets {
			if m.materials[i].ID == id && !m.materials[i].Downloaded {
				m.materials[i].Downloaded = true
				break
			}
		}
	}
	m.selectedM = map[string]bool{}
	m.toast = fmt.Sprintf("downloaded %d %s", len(targets), plural(len(targets), "file", "files"))
	m.toastTicks = 24
}

// ---- commit / wizard ----

func (m *App) openCommit() {
	m.switchTab(TabCommit)
}

func (m *App) closeCommit() {
	if m.tab == TabCommit {
		m.switchTab(m.returnTab)
	}
}

func (m *App) wizardKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch msg.String() {
	case "esc":
		switch m.wizStep {
		case 0, 3:
			m.layout = m.returnLayout
			m.wizStep = 0
		default:
			m.wizStep--
		}
		return nil, true
	case "enter":
		switch m.wizStep {
		case 0:
			m.wizStep = 1
		case 1:
			if len(m.selected) == 0 {
				m.toast = "select at least one assignment"
				m.toastTicks = 20
				return nil, true
			}
			for id := range m.selected {
				if a := m.byID(id); a != nil {
					a.Triage = m.wizAction
				}
				m.pending[id] = m.wizAction
			}
			m.selected = map[string]bool{}
			m.wizStep = 2
		case 2:
			if m.dryRun {
				m.toast = fmt.Sprintf("committed %d %s (dry-run: nothing written yet)", len(m.pending), plural(len(m.pending), "line", "lines"))
			} else {
				m.toast = fmt.Sprintf("committed %d %s", len(m.pending), plural(len(m.pending), "line", "lines"))
			}
			m.pending = map[string]model.Triage{}
			m.toastTicks = 30
			m.wizStep = 3
		default:
			m.layout = m.returnLayout
			m.wizStep = 0
		}
		return nil, true
	case "t":
		m.wizAction = model.Tracked
		return nil, true
	case "m":
		m.wizAction = model.Missed
		return nil, true
	case "i":
		m.wizAction = model.Ignored
		return nil, true
	case "u":
		m.wizAction = model.Undecided
		return nil, true
	}
	return nil, false
}

func (m *App) enter() {
	switch m.tab {
	case TabTasks:
		m.openCommit()
	case TabMaterials:
		m.download()
	}
}

func (m *App) back() {
	m.selected = map[string]bool{}
	m.selectedM = map[string]bool{}
}

// ---- sync / animation ----

func (m *App) startSync() tea.Cmd {
	if m.syncing {
		return nil
	}
	m.syncing = true
	m.syncTicks = 0
	return m.spinner.Tick
}

func (m *App) animating() bool {
	return m.syncing || m.toast != ""
}

func (m *App) advance() {
	if m.syncing {
		m.syncTicks++
		limit := 12
		if m.noAnim {
			limit = 1
		}
		if m.syncTicks >= limit {
			m.syncing = false
			m.toast = "Sync complete · " + m.src.Label()
			m.toastTicks = 24
		}
	}
	if m.toast != "" && m.toastTicks > 0 {
		m.toastTicks--
		if m.toastTicks == 0 {
			m.toast = ""
		}
	}
}

// ---- view ----

func (m *App) boxWidth() int {
	w := m.width
	if w <= 0 {
		w = 100
	}
	return clampInt(w, 40, 400)
}

func (m *App) boxHeight() int {
	h := m.height
	if h <= 0 {
		h = 30
	}
	return clampInt(h, 10, 200)
}

func (m *App) View() string {
	w, h := m.width, m.height
	if w <= 0 {
		w = 100
	}
	if h <= 0 {
		h = 30
	}
	if w < 80 {
		w = maxInt(1, w)
	}
	if h < 24 {
		h = maxInt(1, h)
	}

	title, right, hints, rows := m.tabContent(maxInt(1, w-4), maxInt(1, h-5))
	if m.loadError != "" {
		rows = []string{stOverdue.Render("error · " + m.loadError)}
	} else if !m.loaded {
		rows = []string{stDim.Render("loading…")}
	}
	if m.helpOn {
		title, right, hints, rows = "help", "", []string{"↑↓ move", "tab switch", "? close"}, m.helpLines(maxInt(1, w-2))
	}
	if w < 80 || h < 24 {
		title, right, rows = "terminal too small", "", []string{"terminal too small"}
		hints = []string{"? more"}
	}
	lines := []string{
		m.tabBar(w),
		titleLine(title, right, w),
	}
	lines = append(lines, bodyFrame(rows, w, h-3)...)
	lines = append(lines, m.sharedFooter(w, hints))
	return strings.Join(lines, "\n")
}

// tabBar is the top line: the app name, the tabs, and a right-aligned status.
// No box — gh-dash lets the top and bottom bars sit flush against the screen.
func (m *App) tabBar(w int) string {
	labels := []string{"Tasks", "Materials", "Sync", "Commit"}
	const cellWidth = 14
	var cells []string
	for i, label := range labels {
		count := ""
		switch Tab(i) {
		case TabTasks:
			count = fmt.Sprint(len(m.assignments))
		case TabMaterials:
			count = fmt.Sprint(len(m.materials))
		case TabCommit:
			count = fmt.Sprint(len(m.commitItems()))
		}
		text := pad(label, 9, false) + " " + pad(count, 2, true)
		cells = append(cells, fixedTabCell(text, m.tab == Tab(i), cellWidth))
	}
	left := strings.Join(cells, "")
	status := stFaint.Render("backup · sync · cookie")
	statusWidth := lipgloss.Width(status)
	if w <= len(cells)*cellWidth+statusWidth {
		return fitLine(left+status, w)
	}
	return fitLine(left+strings.Repeat(" ", w-len(cells)*cellWidth-statusWidth)+status, w)
}

func (m *App) sharedFooter(w int, items []string) string {
	queue := m.taskQueueLabel()
	if m.tab == TabCommit {
		queue = stKey.Render(fmt.Sprintf("queue: %d", len(m.pending)))
	}
	if lipgloss.Width(queue) >= w {
		return fitLine(queue, w)
	}
	if len(items) > 6 {
		items = items[:6]
	}
	items = append(items, "? more")
	available := maxInt(0, w-lipgloss.Width(queue)-3)
	for lipgloss.Width(hints(items)) > available && len(items) > 1 {
		items = append(items[:len(items)-2], items[len(items)-1])
	}
	left := hints(items)
	if lipgloss.Width(left)+3+lipgloss.Width(queue) > w {
		left = ""
	}
	return barLine(left, queue, w)
}

// helpBar is the bottom line: the keys that work right now on the left, and the
// commit queue — or whatever the app just did — on the right.
func (m *App) helpBar(w int) string {
	inner := w - 1
	if m.tab == TabTasks && m.layout == LayList {
		return " " + m.taskFooter(inner)
	}
	right := m.queueLabel()
	if f := m.footer(); f != "" {
		right = f
	}
	return " " + barLine(m.hintsLine(inner, m.helpWhere()), right, inner)
}

func (m *App) taskFooter(width int) string {
	queue := m.taskQueueLabel()
	queueWidth := lipgloss.Width(queue)
	if queueWidth >= width {
		return fitLine(queue, width)
	}
	hintWidth := maxInt(0, width-queueWidth-2)
	hints := taskFooterHints(hintWidth)
	gap := maxInt(1, width-lipgloss.Width(hints)-queueWidth)
	return fitLine(hints+strings.Repeat(" ", gap)+queue, width)
}

func (m *App) taskQueueLabel() string {
	tracked, missed, ignored := m.pendingCounts()
	if tracked+missed+ignored == 0 {
		return stDim.Render("queue empty")
	}
	var parts []string
	if tracked > 0 {
		parts = append(parts, fmt.Sprintf("%d track", tracked))
	}
	if missed > 0 {
		parts = append(parts, fmt.Sprintf("%d missed", missed))
	}
	if ignored > 0 {
		parts = append(parts, fmt.Sprintf("%d ignore", ignored))
	}
	return stKey.Render("queue: " + strings.Join(parts, " · "))
}

// barLine lays a left and a right segment on one line of exactly w columns,
// dropping the right segment when the two would collide.
func barLine(left, right string, w int) string {
	if right == "" || lipgloss.Width(left)+3+lipgloss.Width(right) > w {
		return fitLine(left, w)
	}
	gap := maxInt(1, w-lipgloss.Width(left)-lipgloss.Width(right))
	return fitLine(left+strings.Repeat(" ", gap)+right, w)
}

// statusLabel is the right-hand side of the top bar: sync and cookie health.
func (m *App) statusLabel() string {
	if m.syncing {
		if m.noAnim {
			return stSoon.Render("syncing…")
		}
		return m.spinner.View() + stSoon.Render(" syncing ")
	}
	return stFaint.Render("backup 0d · sync 1/2 · cookie ok")
}

// queueLabel summarises what `c` would commit.
func (m *App) queueLabel() string {
	t, mi, ig := m.pendingCounts()
	if t == 0 && mi == 0 && ig == 0 {
		return stFaint.Render("queue empty ")
	}
	parts := []string{}
	if t > 0 {
		parts = append(parts, stFlagT.Render(fmt.Sprintf("%d track", t)))
	}
	if mi > 0 {
		parts = append(parts, stFlagM.Render(fmt.Sprintf("%d missed", mi)))
	}
	if ig > 0 {
		parts = append(parts, stFlagI.Render(fmt.Sprintf("%d ignore", ig)))
	}
	return strings.Join(parts, stFaint.Render(" · ")) + stFaint.Render(" to commit ")
}

// helpWhere names the key set for the help bar.
func (m *App) helpWhere() string {
	switch {
	case m.tab == TabMaterials:
		return "materials"
	case m.tab == TabSync:
		return "sync"
	case m.layout == LayWizard:
		return "wizard"
	default:
		return "tasks"
	}
}

func (m *App) tabContent(width, height int) (string, string, []string, []string) {
	var title, right string
	var hints []string
	var rows []string
	switch m.tab {
	case TabTasks:
		title = fmt.Sprintf("courses · %s · %d assignments", m.filter, len(m.visible()))
		right = taskLegend(maxInt(0, width-lipgloss.Width(title)-4))
		hints = []string{"↑↓ move", "h/l fold", "space select", "t/m/i decide", "/ filter", "c commit"}
		rows = m.taskBody(width, height)
	case TabMaterials:
		title = fmt.Sprintf("materials · %d files · %d local", len(m.materials), m.downloadedMaterials())
		hints = []string{"↑↓ move", "h/l fold", "space select", "d download"}
		rows = m.materialBody(width, height)
	case TabSync:
		title = "sync status · " + m.src.Label()
		hints = []string{"↑↓ move", "s sync"}
		rows = m.syncBody(width, height)
	case TabCommit:
		title = "review & commit · preview"
		hints = []string{"↑↓ move", "enter preview", "esc return"}
		rows = m.commitBody(width, height)
	}
	return title, right, hints, rows
}

func (m *App) downloadedMaterials() int {
	n := 0
	for _, material := range m.materials {
		if material.Downloaded {
			n++
		}
	}
	return n
}

type commitItem struct {
	assignment model.Assignment
	action     model.Triage
}

func (m *App) commitItems() []commitItem {
	items := make([]commitItem, 0, len(m.pending))
	for id, action := range m.pending {
		if assignment := m.byID(id); assignment != nil {
			items = append(items, commitItem{assignment: *assignment, action: action})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].action != items[j].action {
			return items[i].action < items[j].action
		}
		return items[i].assignment.ID < items[j].assignment.ID
	})
	return items
}

func (m *App) commitBody(width, height int) []string {
	items := m.commitItems()
	type displayRow struct {
		text    string
		focused bool
	}
	var all []displayRow
	lastAction := model.Triage(-1)
	focusIndex := 0
	cursorRow := 0
	for _, item := range items {
		if item.action != lastAction {
			lastAction = item.action
			label, style := commitActionLabel(item.action)
			all = append(all, displayRow{text: style.Render("▾ " + label)})
		}
		a := item.assignment
		course := m.shortCode(a.CourseID)
		if focusIndex == m.commitCursor {
			cursorRow = len(all)
		}
		all = append(all, displayRow{text: stRow.Render("  " + a.ID + " · " + course + " · " + a.Name), focused: focusIndex == m.commitCursor})
		focusIndex++
		if item.action == model.Tracked {
			all = append(all, displayRow{text: "    " + stDim.Render(m.taskLine(a))})
		}
		if a.HasDue && a.Due.Before(time.Now()) {
			all = append(all, displayRow{text: "    " + stOverdue.Render("will show as OVERDUE")})
		}
		if similar := m.similarManualTask(a); similar != "" {
			all = append(all, displayRow{text: "    " + stDim.Render(fmt.Sprintf("similar manual task: %q", similar))})
		}
		all = append(all, displayRow{text: ""})
	}
	if len(all) == 0 {
		all = append(all, displayRow{text: stEmpty.Render("no decisions queued · mark items in Tasks first")})
	}
	start, end := windowRange(len(all), cursorRow, height)
	rows := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		rows = append(rows, sharedFocusRow(all[i].text, width, all[i].focused))
	}
	return rows
}

func commitActionLabel(action model.Triage) (string, lipgloss.Style) {
	switch action {
	case model.Tracked:
		return "done (track)", stFlagT
	case model.Missed:
		return "missed", stFlagM
	case model.Ignored:
		return "ignore", stFlagI
	default:
		return "undecided", stFlagNew
	}
}

func (m *App) taskBody(width, height int) []string {
	nodes := m.taskNodes()
	start, end := windowRange(len(nodes), m.taskCursor, height)
	rows := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		rows = append(rows, m.taskTreeRow(nodes, i, width))
	}
	if len(rows) == 0 {
		rows = append(rows, stEmpty.Render("nothing matches this filter"))
	}
	return rows
}

func sharedFocusRow(content string, width int, focused bool) string {
	content = ansiTruncate(content, maxInt(0, width))
	padding := strings.Repeat(" ", maxInt(0, width-lipgloss.Width(content)))
	if focused {
		content = lipgloss.NewStyle().Background(themeTokens.Surface).Render(content)
		padding = stRow.Background(themeTokens.Surface).Render(padding)
	}
	return content + padding
}

func (m *App) materialBody(width, height int) []string {
	nodes := m.matNodes()
	start, end := windowRange(len(nodes), m.matCursor, height)
	rows := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		n := nodes[i]
		selected := n.kind == nodeFile && m.selectedM[n.mat.ID]
		rows = append(rows, m.matTreeRow(nodes, i, width, selected))
	}
	if len(rows) == 0 {
		rows = append(rows, stEmpty.Render("no materials"))
	}
	return rows
}

func (m *App) syncBody(width, height int) []string {
	values := [][2]string{
		{"state", map[bool]string{true: "syncing", false: "idle"}[m.syncing]},
		{"today", "1/2 runs"},
		{"cookie", "ok (expires in ~6h)"},
		{"backup", "0d old"},
		{"source", m.src.Label()},
		{"last sync", m.lastSyncLine()},
	}
	start, end := windowRange(len(values), m.syncCursor, height)
	rows := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		label := stDim.Render(pad(values[i][0], 10, false))
		marker := "  "
		if i == m.syncCursor {
			marker = "▌ "
		}
		row := marker + label + " " + stPrimary.Render(values[i][1])
		rows = append(rows, sharedFocusRow(row, width, i == m.syncCursor))
	}
	return rows
}

func taskLegend(width int) string {
	var full []string
	for _, item := range []struct {
		glyph, label string
		style        lipgloss.Style
	}{{"○", "new", stFlagNew}, {"◐", "tracked", stFlagT}, {"●", "done", stDone}, {"◆", "ignored", stFlagI}, {"×", "missed", stFlagM}} {
		full = append(full, item.style.Render(item.glyph)+" "+stDim.Render(item.label))
	}
	legend := strings.Join(full, "  ")
	if lipgloss.Width(legend) <= width {
		return legend
	}
	glyphs := stFlagNew.Render("○") + "  " + stFlagT.Render("◐") + "  " + stDone.Render("●") + "  " + stFlagI.Render("◆") + "  " + stFlagM.Render("×")
	if lipgloss.Width(glyphs) <= width {
		return glyphs
	}
	return ""
}

func (m *App) helpLines(w int) []string {
	full := strings.Split(m.help.FullHelpView(m.keys.FullHelp()), "\n")
	legend := stFlagNew.Render("○") + " " + stDim.Render("new") + "  " +
		stFlagT.Render("◐") + " " + stDim.Render("tracked") + "  " +
		stDone.Render("●") + " " + stDim.Render("done") + "  " +
		stFlagI.Render("◆") + " " + stDim.Render("ignored") + "  " +
		stFlagM.Render("×") + " " + stDim.Render("missed")
	return append(append(full, legend), stFaint.Render("mouse off: drag to select text"))
}

// ---- shared footer pieces ----

// footer shows the last thing the app did. It replaces the commit-queue readout
// while it is on screen, the way gh-dash surfaces notifications in its status
// bar rather than in a modal.
func (m *App) footer() string {
	if m.toast != "" {
		return stKey.Render("✻ ") + stFaint.Render(m.toast+" ")
	}
	return ""
}
