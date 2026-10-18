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
	TabToday
	TabSync
	tabCount
)

func (t Tab) String() string {
	switch t {
	case TabTasks:
		return "Tasks"
	case TabMaterials:
		return "Materials"
	case TabToday:
		return "Today"
	case TabSync:
		return "Sync"
	}
	return "?"
}

// Layout is how the Tasks screen is drawn. All four share the same data and
// keys; they are just different ways of looking at the same backlog.
type Layout int

const (
	LayList   Layout = iota // two-pane: list + detail
	LayBoard                // 2x2 buckets
	LayWizard               // guided review & commit
	layCount
)

func (l Layout) String() string {
	switch l {
	case LayList:
		return "list"
	case LayBoard:
		return "board"
	case LayWizard:
		return "wizard"
	}
	return "?"
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
		src:       src,
		cfg:       cfg,
		ver:       version,
		keys:      DefaultKeys(),
		help:      h,
		filter:    "all",
		selected:  map[string]bool{},
		selectedM: map[string]bool{},
		pending:   map[string]model.Triage{},
		spinner:   sp,
		wizAction: model.Tracked,
		width:     88,
		height:    24,
		dryRun:    true,
		noAnim:    cfg.NoAnim,
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
		m.tab = Tab(int(msg.String()[0] - '1'))
	case "v":
		m.layout = (m.layout + 1) % layCount
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
		n := len(m.visible())
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
		for _, a := range m.visible() {
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

// cursorAssignmentID is the item the keyboard is pointing at, or "".
func (m *App) cursorAssignmentID() string {
	switch {
	case m.tab == TabTasks && m.layout == LayBoard:
		items := m.boardItems(m.boardCol)
		if c := m.boardCursor[m.boardCol]; c >= 0 && c < len(items) {
			return items[c].ID
		}
	case m.tab == TabTasks:
		items := m.visible()
		if m.taskCursor >= 0 && m.taskCursor < len(items) {
			return items[m.taskCursor].ID
		}
	}
	return ""
}

// targetIDs is what a triage key acts on: the selection, else the cursor item.
func (m *App) targetIDs() []string {
	var ids []string
	for _, a := range m.visible() {
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
			m.layout = LayList
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
			m.layout = LayList
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

const (
	maxBoxW = 100
	maxBoxH = 30
	minBoxW = 72
	minBoxH = 10
)

func (m *App) boxWidth() int {
	w := m.width
	if w <= 0 {
		w = 88
	}
	return clampInt(w, minBoxW, maxBoxW)
}

func (m *App) boxHeight() int {
	h := m.height
	if h <= 0 {
		h = 24
	}
	return clampInt(h-1, minBoxH, maxBoxH)
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
	right := m.headerRight()
	if lipgloss.Width(right)+lipgloss.Width("kamps")+10 > w {
		right = ""
	}
	left := m.headerLeft(w - lipgloss.Width(right) - 10)
	lines = append(lines, frameTop(left, right, w))
	for _, b := range body {
		lines = append(lines, frameLine(b, w))
	}
	for len(lines) < h-1 {
		lines = append(lines, frameLine("", w))
	}
	lines = append(lines, frameBottom(w))
	return strings.Join(lines, "\n")
}

// headerLeft renders "kamps" plus as many tabs as fit in maxW columns.
func (m *App) headerLeft(maxW int) string {
	type item struct {
		label  string
		active bool
	}
	items := []item{
		{fmt.Sprintf("Tasks %d", len(m.assignments)), m.tab == TabTasks},
		{fmt.Sprintf("Materials %d", len(m.materials)), m.tab == TabMaterials},
		{"Today", m.tab == TabToday},
		{"Sync", m.tab == TabSync},
	}
	head := stHeader.Render("kamps")
	parts := []string{head}
	used := lipgloss.Width(head)
	for _, it := range items {
		seg := stTabOff.Render(it.label)
		if it.active {
			seg = stTabOn.Render(it.label)
		}
		if used+2+lipgloss.Width(seg) > maxW {
			break
		}
		used += 2 + lipgloss.Width(seg)
		parts = append(parts, seg)
	}
	return strings.Join(parts, "  ")
}

func (m *App) headerRight() string {
	if m.syncing {
		if m.noAnim {
			return stSoon.Render("syncing…")
		}
		return m.spinner.View() + stSoon.Render(" syncing")
	}
	return stLabel.Render("backup 0d · cookie ok")
}

func (m *App) bodyLines(w, budget int) []string {
	switch m.tab {
	case TabTasks:
		switch m.layout {
		case LayBoard:
			return m.boardLines(w, budget)
		case LayWizard:
			return m.wizardLines(w, budget)
		default:
			return m.listLines(w, budget)
		}
	case TabMaterials:
		return m.materialsLines(w, budget)
	case TabToday:
		return m.todayLines(w, budget)
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

func (m *App) statusLine(w int) string {
	t, mi, ig := m.pendingCounts()
	left := stRowDim.Render("queue: ")
	if t == 0 && mi == 0 && ig == 0 {
		left += stRowDim.Render("nothing to commit")
	} else {
		left += stFlagT.Render(fmt.Sprintf("%d track", t)) + stRowDim.Render(" · ") +
			stFlagM.Render(fmt.Sprintf("%d missed", mi)) + stRowDim.Render(" · ") +
			stFlagI.Render(fmt.Sprintf("%d ignore", ig))
	}
	right := stKey.Render("[c]") + " " + stRowDim.Render("review & commit")
	return spread(left, right, w)
}

func (m *App) footer() string {
	if m.toast != "" {
		return stToast.Render(m.toast)
	}
	return ""
}
