package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/core"
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
	src     source.Source
	cfg     config.Config
	service *core.Service
	ver     string
	keys    KeyMap
	program *tea.Program

	courses      []model.Course
	assignments  []model.Assignment
	materials    []model.Material
	loaded       bool
	loading      bool
	loadError    string
	loadNotice   string
	downloads    map[string]bool
	scanSummary  string
	unknownFiles []string
	lastSync     time.Time
	offline      bool
	cacheNotice  string

	restoreTaskKey       string
	restoreTaskFallbacks []string
	restoreMatKey        string
	restoreCommitID      string

	tab    Tab
	filter string
	search string

	searching     bool
	paletteOn     bool
	paletteQuery  string
	paletteCursor int

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
	queue     *core.Queue

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
	// statusErr is a durable failure shown in the footer until dismissed.
	statusErr string

	// pendingDecisions/pendingDownloads are the in-memory changes waiting for
	// the single persistence command. persistInFlight serializes disk writes.
	pendingDecisions map[string]model.Triage
	pendingDownloads map[string]bool
	persistInFlight  bool

	width      int
	height     int
	helpOn     bool
	helpScroll int
	noAnim     bool
	showAll    bool

	triageArmed       bool
	armedTriage       model.Triage
	armedTriageIDs    []string
	armedTriageHidden int

	// now is time.Now in production; tests inject a fixed clock so relative
	// due labels and snapshots are deterministic.
	now func() time.Time
}

// New builds the root model.
func New(src source.Source, cfg config.Config, version string) *App {
	applyTheme(cfg.Theme)
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = stCursor

	service := core.New(src, cfg)
	state := service.State()
	statePath := service.StatePath()
	stateErr := service.StateErr()

	return &App{
		src:              src,
		cfg:              cfg,
		service:          service,
		ver:              version,
		keys:             DefaultKeys(),
		filter:           "all",
		returnTab:        TabTasks,
		detailPane:       -1,
		tabFilters:       [tabCount]string{"all", "all", "all", "all"},
		tabExpanded:      [tabCount]map[string]bool{0: {}, 1: {}, 2: {}, 3: {}},
		expanded:         map[string]bool{},
		selected:         map[string]bool{},
		selectedM:        map[string]bool{},
		queue:            core.NewQueue(),
		undoDecisions:    map[string]model.Triage{},
		downloads:        map[string]bool{},
		downloadCancels:  map[string]context.CancelFunc{},
		state:            state,
		statePath:        statePath,
		stateErr:         stateErr,
		pendingDecisions: map[string]model.Triage{},
		pendingDownloads: map[string]bool{},
		lastSync:         state.LastSync,
		spinner:          sp,
		width:            100,
		height:           30,
		noAnim:           cfg.NoAnim,
		now:              time.Now,
	}
}

func (m *App) currentTime() time.Time {
	if m.now != nil {
		return m.now()
	}
	return time.Now()
}

type loadedMsg struct{ core.Snapshot }

type errMsg struct{ err error }

func loadCmd(service *core.Service) tea.Cmd {
	return func() tea.Msg {
		snapshot, err := service.Load(context.Background())
		if err != nil {
			return errMsg{err}
		}
		return loadedMsg{snapshot}
	}
}

func (m *App) Init() tea.Cmd {
	m.loading = true
	if !strings.HasPrefix(m.src.Label(), "demo") && m.loadCache() {
		m.loaded = true
	}
	return loadCmd(m.service)
}

func (m *App) applySnapshot(snapshot core.Snapshot) {
	m.courses = snapshot.Courses
	m.assignments = snapshot.Assignments
	m.materials = snapshot.Materials
	m.loadNotice = snapshot.Notice
	m.lastSync = snapshot.LoadedAt
	m.scanSummary = snapshot.Scan.Text
	m.unknownFiles = append([]string(nil), snapshot.Scan.UnknownFiles...)
	m.state = m.service.State()
}

func (m *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.helpScroll = clampInt(m.helpScroll, 0, m.helpMaxScroll())
		return m, nil

	case loadedMsg:
		m.loaded = true
		m.loading = false
		m.syncing = false
		m.loadError = ""
		m.offline = false
		m.cacheNotice = ""
		m.statusErr = ""
		m.applySnapshot(msg.Snapshot)
		m.clampCursor()
		m.restoreCursors()
		if len(msg.Warnings) > 0 {
			m.toast = msg.Warnings[0]
			m.toastTicks = 40
		}
		if m.stateErr != "" {
			m.statusErr = "state: " + m.stateErr
			m.toast = ""
			m.stateErr = ""
		}
		m.warnSessionExpiry()
		return m, nil

	case errMsg:
		m.loaded = true
		m.loading = false
		m.syncing = false
		m.loadError = msg.err.Error()
		m.statusErr = ""
		return m, loadCachedCmd(m.service)

	case cacheLoadedMsg:
		// Ignore a cache result that lost the race to a fresh load or a new
		// sync; only the active load error should fall back to the cache.
		if !msg.ok || m.loadError == "" || m.syncing {
			return m, nil
		}
		m.offline = true
		m.loadError = ""
		m.statusErr = ""
		m.applyCachedSnapshot(msg.snapshot)
		m.toast = "offline · cached from " + m.cacheNotice
		m.toastTicks = 40
		m.clampCursor()
		m.restoreCursors()
		return m, nil

	case stateSavedMsg:
		return m, m.handleStateSaved(msg)

	case clipboardMsg:
		if msg.err != nil {
			m.toast = "copy failed"
			m.statusErr = "copy failed: " + msg.err.Error()
		} else {
			m.statusErr = ""
			m.toast = fmt.Sprintf("copied %d %s", msg.lines, plural(msg.lines, "line", "lines"))
		}
		m.toastTicks = 24
		return m, nil

	case exportMsg:
		if msg.err != nil {
			m.toast = "export failed"
			m.statusErr = "export failed: " + msg.err.Error()
		} else {
			m.statusErr = ""
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
			m.toast = "download failed"
			m.statusErr = "download failed: " + downloadErrorText(msg.err)
			m.toastTicks = 40
			return m, nil
		}
		m.statusErr = ""
		m.toast = "saved " + savedFileLabel(msg.path)
		m.toastTicks = 24
		return m, nil

	case openURLMsg:
		if msg.err != nil {
			m.toast = "could not open browser"
			m.statusErr = "could not open browser: " + msg.err.Error()
		} else {
			m.statusErr = ""
			m.toast = "opened in browser"
		}
		m.toastTicks = 24
		return m, nil

	case openURLsMsg:
		if msg.err != nil {
			m.toast = "could not open all links"
			m.statusErr = "could not open all links: " + msg.err.Error()
		} else {
			m.statusErr = ""
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
