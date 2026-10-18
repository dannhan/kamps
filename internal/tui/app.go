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

	tab    Tab
	layout Layout
	filter string

	taskCursor  int
	matCursor   int
	boardCol    int
	boardCursor [4]int // per triage column

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
	sp.Style = lipgloss.NewStyle().Foreground(colAccent)

	h := help.New()

	return &App{
		src:          src,
		cfg:          cfg,
		ver:          version,
		keys:         DefaultKeys(),
		help:         h,
		filter:       "all",
		layout:       viewCycle[0],
		returnLayout: LayList,
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
		return m, nil

	case errMsg:
		m.toast = "error: " + msg.err.Error()
		m.toastTicks = 40
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

	// The wizard owns the arrow/space/enter keys.
	if m.tab == TabTasks && m.layout == LayWizard {
		if cmd, handled := m.wizardKey(msg); handled {
			return m, cmd
		}
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "?":
		m.helpOn = true
	case "tab":
		m.tab = (m.tab + 1) % tabCount
	case "shift+tab":
		m.tab = (m.tab + tabCount - 1) % tabCount
	case "1", "2", "3", "4":
		if idx := int(msg.String()[0] - '1'); idx < int(tabCount) {
			m.tab = Tab(idx)
		}
	case "v":
		m.layout = m.layout.next()
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
	case "/":
		m.cycleFilter()
	case "s":
		return m, m.startSync()
	case "d":
		m.download()
	case "enter":
		m.enter()
	case "esc":
		m.back()
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
		m.taskCursor = clampInt(m.taskCursor+delta, 0, maxInt(0, n-1))
	case TabMaterials:
		m.matCursor = clampInt(m.matCursor+delta, 0, maxInt(0, len(m.materials)-1))
	}
}

func (m *App) moveX(delta int) {
	if m.tab != TabTasks || m.layout != LayBoard {
		return
	}
	m.boardCol = clampInt(m.boardCol+delta, 0, len(boardOrder)-1)
}

func (m *App) toggleSelect() {
	id := m.cursorAssignmentID()
	if id != "" {
		m.selected[id] = !m.selected[id]
		if !m.selected[id] {
			delete(m.selected, id)
		}
		return
	}
	if m.tab == TabMaterials {
		if i := m.matCursor; i >= 0 && i < len(m.materials) {
			id := m.materials[i].ID
			m.selectedM[id] = !m.selectedM[id]
			if !m.selectedM[id] {
				delete(m.selectedM, id)
			}
		}
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
	cur := 0
	for i, f := range filterNames {
		if f == m.filter {
			cur = i
		}
	}
	m.filter = filterNames[(cur+1)%len(filterNames)]
	m.taskCursor = 0
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
	default:
		return m.visible()
	}
}

// cursorAssignmentID is the item the keyboard is pointing at, or "".
func (m *App) cursorAssignmentID() string {
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
func (m *App) targetIDs() []string {
	var ids []string
	for _, a := range m.assignments {
		if m.selected[a.ID] {
			ids = append(ids, a.ID)
		}
	}
	if len(ids) == 0 {
		if id := m.cursorAssignmentID(); id != "" {
			ids = append(ids, id)
		}
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
	n := 0
	for _, mt := range m.materials {
		if m.selectedM[mt.ID] && !mt.Downloaded {
			n++
		}
	}
	if n == 0 {
		if i := m.matCursor; i >= 0 && i < len(m.materials) && !m.materials[i].Downloaded {
			n = 1
		}
	}
	if n == 0 {
		m.toast = "nothing new to download"
		m.toastTicks = 20
		return
	}
	for i := range m.materials {
		if m.selectedM[m.materials[i].ID] || (n == 1 && i == m.matCursor) {
			m.materials[i].Downloaded = true
		}
	}
	m.selectedM = map[string]bool{}
	m.toast = fmt.Sprintf("downloaded %d %s", n, plural(n, "file", "files"))
	m.toastTicks = 24
}

// ---- commit / wizard ----

func (m *App) openCommit() {
	if m.tab != TabTasks {
		m.toast = "nothing to commit here"
		m.toastTicks = 20
		return
	}
	if m.layout != LayWizard {
		m.returnLayout = m.layout
		if !m.returnLayout.inViewCycle() {
			m.returnLayout = viewCycle[0]
		}
	}
	m.layout = LayWizard
	if len(m.pending) > 0 {
		m.wizStep = 2
	} else {
		m.wizStep = 0
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
	w := m.boxWidth()
	h := m.boxHeight()

	var body []string
	if m.helpOn {
		body = m.helpLines(w)
	} else {
		body = m.bodyLines(w, h-2)
	}
	if len(body) > h-2 {
		body = body[:h-2]
	}

	lines := make([]string, 0, h)
	lines = append(lines, m.tabBar(w))
	for _, b := range body {
		lines = append(lines, fitLine(b, w))
	}
	for len(lines) < h-1 {
		lines = append(lines, strings.Repeat(" ", w))
	}
	lines = append(lines, m.helpBar(w))
	return strings.Join(lines, "\n")
}

// tabBar is the top line: the app name, the tabs, and a right-aligned status.
// No box — gh-dash lets the top and bottom bars sit flush against the screen.
func (m *App) tabBar(w int) string {
	type item struct {
		label  string
		active bool
	}
	items := []item{
		{fmt.Sprintf("Tasks %d", len(m.assignments)), m.tab == TabTasks},
		{fmt.Sprintf("Materials %d", len(m.materials)), m.tab == TabMaterials},
		{"Sync", m.tab == TabSync},
	}

	left := stTitle.Render("kamps")
	for _, it := range items {
		seg := stTabOff.Render(it.label)
		if it.active {
			seg = stTabOn.Render(it.label)
		}
		left += " " + seg
	}
	return " " + barLine(left, m.statusLabel(), w-1)
}

// helpBar is the bottom line: the keys that work right now on the left, and the
// commit queue — or whatever the app just did — on the right.
func (m *App) helpBar(w int) string {
	inner := w - 1
	right := m.queueLabel()
	if f := m.footer(); f != "" {
		right = f
	}
	return " " + barLine(m.hintsLine(inner, m.helpWhere()), right, inner)
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

func (m *App) bodyLines(w, budget int) []string {
	switch m.tab {
	case TabTasks:
		switch m.layout {
		case LayDashboard:
			return m.dashboardLines(w, budget)
		case LayBoard:
			return m.boardLines(w, budget)
		case LayWizard:
			return m.wizardLines(w, budget)
		default:
			return m.listLines(w, budget)
		}
	case TabMaterials:
		return m.materialsLines(w, budget)
	case TabSync:
		return m.syncLines(w, budget)
	}
	return nil
}

func (m *App) helpLines(w int) []string {
	full := m.help.FullHelpView(m.keys.FullHelp())
	box := stHelpBox.Render(stTitle.Render("kamps — keys") + "\n\n" + full)
	return strings.Split(box, "\n")
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
