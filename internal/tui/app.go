package tui

import (
	"context"
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/source"
	"github.com/ruyam/kamps/internal/store"
)

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
	loadNotice  string

	tab    Tab
	filter string
	search string

	searching bool

	taskCursor   int
	matCursor    int
	syncCursor   int
	commitCursor int

	// detailPane is the tree detail pane override: -1 follows the terminal
	// width, 0 keeps it closed, 1 keeps it open.
	detailPane int

	tabFilters  [tabCount]string
	tabExpanded [tabCount]map[string]bool
	returnTab   Tab

	// expanded holds the fold state of the tree panes, keyed by stable tree
	// keys. A missing key means open — collapsing is the explicit act.
	expanded map[string]bool

	selected  map[string]bool // assignment id -> checked
	selectedM map[string]bool // material id -> checked
	pending   map[string]model.Triage

	// state is the local store behind decisions and download flags; statePath
	// is empty when the config has no state directory.
	state     store.State
	statePath string
	stateErr  string

	spinner    spinner.Model
	syncing    bool
	syncTicks  int
	toast      string
	toastTicks int

	width  int
	height int
	helpOn bool
	noAnim bool
}

// New builds the root model.
func New(src source.Source, cfg config.Config, version string) *App {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(themeTokens.Accent)

	h := help.New()

	state := store.New()
	statePath := cfg.StatePath()
	stateErr := ""
	if statePath != "" {
		loaded, err := store.Load(statePath)
		if err != nil {
			stateErr = err.Error()
		} else {
			state = loaded
		}
	}

	return &App{
		src:         src,
		cfg:         cfg,
		ver:         version,
		keys:        DefaultKeys(),
		help:        h,
		filter:      "all",
		returnTab:   TabTasks,
		detailPane:  -1,
		tabFilters:  [tabCount]string{"all", "all", "all", "all"},
		tabExpanded: [tabCount]map[string]bool{0: {}, 1: {}, 2: {}, 3: {}},
		expanded:    map[string]bool{},
		selected:    map[string]bool{},
		selectedM:   map[string]bool{},
		pending:     map[string]model.Triage{},
		state:       state,
		statePath:   statePath,
		stateErr:    stateErr,
		spinner:     sp,
		width:       100,
		height:      30,
		noAnim:      cfg.NoAnim,
	}
}

type loadedMsg struct{ source.Snapshot }

type errMsg struct{ err error }

func loadCmd(src source.Source) tea.Cmd {
	return func() tea.Msg {
		snapshot, err := src.Load(context.Background())
		if err != nil {
			return errMsg{err}
		}
		return loadedMsg{snapshot}
	}
}

func (m *App) Init() tea.Cmd { return loadCmd(m.src) }

func (m *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.help.Width = msg.Width
		return m, nil

	case loadedMsg:
		m.loaded = true
		m.loadError = ""
		m.loadNotice = msg.Notice
		m.courses = msg.Courses
		m.assignments = msg.Assignments
		m.materials = msg.Materials
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
		m.applyState()
		m.clampCursor()
		if m.stateErr != "" {
			m.toast = "state: " + m.stateErr
			m.toastTicks = 40
			m.stateErr = ""
		}
		return m, nil

	case errMsg:
		m.loaded = true
		m.loadError = msg.err.Error()
		return m, nil

	case clipboardMsg:
		if msg.err != nil {
			m.toast = "copy failed: " + msg.err.Error()
		} else {
			m.toast = fmt.Sprintf("copied %d %s", msg.lines, plural(msg.lines, "line", "lines"))
		}
		m.toastTicks = 24
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
