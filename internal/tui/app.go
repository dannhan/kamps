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
	"github.com/ruyam/kamps/internal/cache"
	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/source"
	"github.com/ruyam/kamps/internal/store"
	"github.com/ruyam/kamps/internal/workspace"
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
	src     source.Source
	cfg     config.Config
	ver     string
	keys    KeyMap
	help    help.Model
	program *tea.Program

	courses     []model.Course
	assignments []model.Assignment
	materials   []model.Material
	loaded      bool
	loading     bool
	loadError   string
	loadNotice  string
	downloads   map[string]bool
	scanSummary string
	lastSync    time.Time
	offline     bool
	cacheNotice string

	restoreTaskKey  string
	restoreMatKey   string
	restoreCommitID string

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

	visual          bool
	rangeAnchor     int
	undoDecisions   map[string]model.Triage
	dropAllArmed    bool
	openAllArmed    bool
	downloadCancels map[string]context.CancelFunc

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
		src:             src,
		cfg:             cfg,
		ver:             version,
		keys:            DefaultKeys(),
		help:            h,
		filter:          "all",
		returnTab:       TabTasks,
		detailPane:      -1,
		tabFilters:      [tabCount]string{"all", "all", "all", "all"},
		tabExpanded:     [tabCount]map[string]bool{0: {}, 1: {}, 2: {}, 3: {}},
		expanded:        map[string]bool{},
		selected:        map[string]bool{},
		selectedM:       map[string]bool{},
		pending:         map[string]model.Triage{},
		undoDecisions:   map[string]model.Triage{},
		downloads:       map[string]bool{},
		downloadCancels: map[string]context.CancelFunc{},
		state:           state,
		statePath:       statePath,
		stateErr:        stateErr,
		lastSync:        state.LastSync,
		spinner:         sp,
		width:           100,
		height:          30,
		noAnim:          cfg.NoAnim,
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
		m.loading = false
		m.syncing = false
		m.loadError = ""
		m.offline = false
		m.cacheNotice = ""
		m.loadNotice = msg.Notice
		m.lastSync = time.Now()
		m.updateState(func(s *store.State) { s.LastSync = m.lastSync })
		m.courses = msg.Courses
		resolved, notices := workspace.Resolve(m.courses, m.cfg.WorkspaceRoots)
		m.courses = resolved
		if len(notices) > 0 {
			combined := append([]string(nil), notices...)
			if m.loadNotice != "" {
				combined = append([]string{m.loadNotice}, combined...)
			}
			m.loadNotice = strings.Join(combined, "; ")
		}
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
		m.applyAutoTriage()
		m.reconcileLibrary()
		m.clampCursor()
		m.restoreCursors()
		if !strings.HasPrefix(m.src.Label(), "demo") {
			_ = cache.Save(m.cfg.CachePath(), m.cfg.SiteURL, msg.Snapshot)
		}
		if m.stateErr != "" {
			m.toast = "state: " + m.stateErr
			m.toastTicks = 40
			m.stateErr = ""
		}
		return m, nil

	case errMsg:
		m.loaded = true
		m.loading = false
		m.syncing = false
		if m.loadCache() {
			m.offline = true
			m.loadError = ""
			m.toast = "offline · " + m.cacheNotice
			m.toastTicks = 40
			return m, nil
		}
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

	case exportMsg:
		if msg.err != nil {
			m.toast = "export failed: " + msg.err.Error()
		} else {
			m.toast = fmt.Sprintf("wrote %d %s (%d already present)", msg.result.Added, plural(msg.result.Added, "line", "lines"), msg.result.Skipped)
		}
		m.toastTicks = 32
		return m, nil

	case downloadProgressMsg:
		if material := m.materialByID(msg.id); material != nil {
			m.toast = downloadProgressText(*material, msg.done, msg.total)
			m.toastTicks = 24
		}
		return m, nil

	case downloadDoneMsg:
		delete(m.downloads, msg.id)
		if cancel := m.downloadCancels[msg.id]; cancel != nil {
			cancel()
			delete(m.downloadCancels, msg.id)
		}
		if msg.err != nil {
			m.toast = "download failed: " + downloadErrorText(msg.err)
			m.toastTicks = 40
			return m, nil
		}
		m.markDownloaded(msg.id, msg.path)
		m.toast = "saved " + savedFileLabel(msg.path)
		m.toastTicks = 24
		return m, nil

	case openURLMsg:
		if msg.err != nil {
			m.toast = "could not open browser: " + msg.err.Error()
		} else {
			m.toast = "opened in browser"
		}
		m.toastTicks = 24
		return m, nil

	case openURLsMsg:
		if msg.err != nil {
			m.toast = "could not open all links: " + msg.err.Error()
		} else {
			m.toast = fmt.Sprintf("opened %d links", msg.count)
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
