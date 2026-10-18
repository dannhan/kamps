// Package tui is the Bubble Tea application: a small, cheerful terminal UI
// over a source.Source.
//
// Architecture (The Elm Architecture):
//
//	Init   -> kick off async work (load data, start animations)
//	Update -> react to messages (keys, window resize, ticks, loaded data)
//	View   -> render the model to a string
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
)

// Section is one top-level view.
type Section int

const (
	SecCourses Section = iota
	SecAssignments
	SecMaterials
	SecDownloads
	secCount
)

func (s Section) String() string {
	switch s {
	case SecCourses:
		return "Courses"
	case SecAssignments:
		return "Assignments"
	case SecMaterials:
		return "Materials"
	case SecDownloads:
		return "Downloads"
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

	section  Section
	cursors  [secCount]int
	selected map[string]bool // material id -> selected

	queue    []string           // material ids being "downloaded"
	progress map[string]float64 // material id -> 0..1

	spinner spinner.Model
	syncing bool
	ticks   int

	toast string

	width  int
	height int
	helpOn bool
}

// New builds the root model.
func New(src source.Source, cfg config.Config, version string) *App {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colPink)

	h := help.New()

	return &App{
		src:      src,
		cfg:      cfg,
		ver:      version,
		keys:     DefaultKeys(),
		help:     h,
		selected: map[string]bool{},
		progress: map[string]float64{},
		spinner:  sp,
		width:    80,
		height:   24,
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

// ---- Init ----

func (m *App) Init() tea.Cmd {
	return loadCmd(m.src)
}

// ---- Update ----

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
		sort.Slice(m.assignments, func(i, j int) bool { return lessAssignment(m.assignments[i], m.assignments[j]) })
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
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		m.ticks++
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

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "?":
		m.helpOn = true
	case "tab", "l":
		m.section = (m.section + 1) % secCount
	case "shift+tab", "h":
		m.section = (m.section + secCount - 1) % secCount
	case "1", "2", "3", "4":
		m.section = Section(int(msg.String()[0] - '1'))
	case "up", "k":
		m.move(-1)
	case "down", "j":
		m.move(1)
	case "esc":
		m.selected = map[string]bool{}
	case " ":
		m.toggleSelect()
	case "x":
		m.toggleDone()
	case "d":
		if cmd := m.enqueue(); cmd != nil {
			return m, cmd
		}
	case "s":
		if !m.syncing {
			m.syncing = true
			m.ticks = 0
			return m, m.spinner.Tick
		}
	}
	return m, nil
}

// ---- behaviours ----

func (m *App) move(delta int) {
	n := m.len(m.section)
	if n == 0 {
		return
	}
	c := m.cursors[m.section] + delta
	if c < 0 {
		c = 0
	}
	if c >= n {
		c = n - 1
	}
	m.cursors[m.section] = c
}

func (m *App) toggleSelect() {
	if m.section != SecMaterials {
		return
	}
	i := m.cursors[SecMaterials]
	if i < 0 || i >= len(m.materials) {
		return
	}
	id := m.materials[i].ID
	m.selected[id] = !m.selected[id]
	if !m.selected[id] {
		delete(m.selected, id)
	}
}

func (m *App) toggleDone() {
	if m.section != SecAssignments {
		return
	}
	i := m.cursors[SecAssignments]
	if i < 0 || i >= len(m.assignments) {
		return
	}
	a := &m.assignments[i]
	if a.Status == model.Done {
		a.Status = model.Todo
		m.toast = "reopened " + a.ID
	} else {
		a.Status = model.Done
		m.toast = "Done! 🎉  " + a.ID
	}
}

// enqueue starts a "download" for selected materials (or the cursor one).
func (m *App) enqueue() tea.Cmd {
	if m.section != SecMaterials {
		m.toast = "switch to Materials to download"
		return nil
	}
	var ids []string
	for _, mat := range m.materials {
		if m.selected[mat.ID] {
			ids = append(ids, mat.ID)
		}
	}
	if len(ids) == 0 {
		i := m.cursors[SecMaterials]
		if i >= 0 && i < len(m.materials) && !m.materials[i].Downloaded {
			ids = append(ids, m.materials[i].ID)
		}
	}
	if len(ids) == 0 {
		m.toast = "nothing new to download"
		return nil
	}
	m.queue = append(m.queue, ids...)
	for _, id := range ids {
		if _, ok := m.progress[id]; !ok {
			m.progress[id] = 0
		}
	}
	m.toast = fmt.Sprintf("queued %d file(s)", len(ids))
	m.section = SecDownloads
	m.selected = map[string]bool{}
	return m.spinner.Tick
}

func (m *App) animating() bool {
	return m.syncing || len(m.queue) > 0 || m.toast != ""
}

// advance moves animations forward by one tick.
func (m *App) advance() {
	// fake sync finishes after ~1.2s
	if m.syncing && m.ticks%12 == 0 {
		m.syncing = false
		m.toast = "Sync complete · " + m.src.Label()
	}
	// progress bars
	if len(m.queue) > 0 {
		done := m.queue[:0]
		for _, id := range m.queue {
			m.progress[id] += 0.05
			if m.progress[id] >= 1 {
				m.progress[id] = 1
				m.markDownloaded(id)
			} else {
				done = append(done, id)
			}
		}
		m.queue = append([]string(nil), done...)
		if len(m.queue) == 0 {
			m.toast = "Download complete ✓"
		}
	}
	// age the toast
	if m.toast != "" && m.ticks%8 == 0 {
		m.toast = ""
	}
}

func (m *App) markDownloaded(id string) {
	for i := range m.materials {
		if m.materials[i].ID == id {
			m.materials[i].Downloaded = true
		}
	}
	delete(m.progress, id)
}

func (m *App) len(s Section) int {
	switch s {
	case SecCourses:
		return len(m.courses)
	case SecAssignments:
		return len(m.assignments)
	case SecMaterials:
		return len(m.materials)
	case SecDownloads:
		return len(m.queue)
	}
	return 0
}

func lessAssignment(a, b model.Assignment) bool {
	if a.HasDue != b.HasDue {
		return a.HasDue // dated items first
	}
	if a.HasDue && !a.Due.Equal(b.Due) {
		return a.Due.Before(b.Due)
	}
	return a.ID < b.ID
}
