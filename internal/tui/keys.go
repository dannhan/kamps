package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ruyam/kamps/internal/model"
)

// KeyMap is the single source of truth for keybindings. The footer hints are
// built by hand (see hints in frame.go); this drives the "?" overlay.
type KeyMap struct {
	Up, Down         key.Binding
	Top, Bottom      key.Binding
	PageUp, PageDown key.Binding
	Left, Right      key.Binding
	NextTab          key.Binding
	PrevTab          key.Binding
	Tabs             key.Binding
	Select           key.Binding
	Track            key.Binding
	Missed           key.Binding
	Ignore           key.Binding
	Undecide         key.Binding
	All              key.Binding
	Range            key.Binding
	Invert           key.Binding
	Undo             key.Binding
	Commit           key.Binding
	Write            key.Binding
	Copy             key.Binding
	Drop             key.Binding
	DropAll          key.Binding
	Search           key.Binding
	Filter           key.Binding
	Sort             key.Binding
	ShowAll          key.Binding
	Sync             key.Binding
	Download         key.Binding
	Open             key.Binding
	Confirm          key.Binding
	Back             key.Binding
	Help             key.Binding
	Palette          key.Binding
	Quit             key.Binding
}

// DefaultKeys returns the default bindings.
func DefaultKeys() KeyMap {
	return KeyMap{
		Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Left:     key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "left · fold")),
		Right:    key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "right · open")),
		Top:      key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g", "top")),
		Bottom:   key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G", "bottom")),
		PageUp:   key.NewBinding(key.WithKeys("ctrl+u", "pgup"), key.WithHelp("ctrl+u", "page up")),
		PageDown: key.NewBinding(key.WithKeys("ctrl+d", "pgdown"), key.WithHelp("ctrl+d", "page down")),
		NextTab:  key.NewBinding(key.WithKeys("tab", "]"), key.WithHelp("tab", "next pane (] next view)")),
		PrevTab:  key.NewBinding(key.WithKeys("shift+tab", "["), key.WithHelp("⇧tab", "previous pane ([ previous view)")),
		Tabs:     key.NewBinding(key.WithKeys("1", "2", "3", "4"), key.WithHelp("1-4", "tabs")),
		Select:   key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "select")),
		Track:    key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "track")),
		Missed:   key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "missed")),
		Ignore:   key.NewBinding(key.WithKeys("i", "I"), key.WithHelp("i", "ignore")),
		Undecide: key.NewBinding(key.WithKeys("n", "0"), key.WithHelp("n", "new")),
		All:      key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "select all")),
		Range:    key.NewBinding(key.WithKeys("v", "V"), key.WithHelp("v", "range select")),
		Invert:   key.NewBinding(key.WithKeys("*"), key.WithHelp("*", "invert visible")),
		Undo:     key.NewBinding(key.WithKeys("u", "ctrl+z"), key.WithHelp("u", "undo triage")),
		Commit:   key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "review queue")),
		Write:    key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "write queue")),
		Copy:     key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy preview")),
		Drop:     key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "drop from queue")),
		DropAll:  key.NewBinding(key.WithKeys("X"), key.WithHelp("X", "drop all queued")),
		Search:   key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		Sort:     key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "reverse Materials week order")),
		Filter:   key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "filter")),
		ShowAll:  key.NewBinding(key.WithKeys("A"), key.WithHelp("A", "all tasks")),
		Sync:     key.NewBinding(key.WithKeys("s", "r"), key.WithHelp("s", "sync")),
		Download: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "download")),
		Open:     key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open link")),
		Confirm:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "details / fold / commit")),
		Back:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "all keys")),
		Palette:  key.NewBinding(key.WithKeys(":", "ctrl+k"), key.WithHelp(":", "commands")),
		Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

// ShortHelp satisfies help.KeyMap.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.NextTab, k.Select, k.Track, k.Missed, k.Ignore, k.Commit, k.Help, k.Quit}
}

// FullHelp satisfies help.KeyMap (the "?" overlay).
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Top, k.Bottom, k.PageUp, k.PageDown, k.Left, k.Right, k.NextTab, k.PrevTab, k.Tabs},
		{k.Select, k.All, k.Range, k.Invert, k.Track, k.Missed, k.Ignore, k.Undecide, k.Undo},
		{k.Search, k.Filter, k.Sort, k.ShowAll, k.Commit, k.Write, k.Copy, k.Drop, k.DropAll, k.Sync, k.Download, k.Open},
		{k.Back, k.Help, k.Palette, k.Quit},
	}
}

func (m *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.helpOn {
		switch msg.String() {
		case "?", "esc", "q":
			m.helpOn = false
			m.helpScroll = 0
			return m, nil
		case "ctrl+c":
			return m, tea.Quit
		case "down", "j":
			m.helpScroll++
		case "up", "k":
			m.helpScroll--
		case "g", "home":
			m.helpScroll = 0
		case "G", "end":
			m.helpScroll = 1 << 20
		case "ctrl+d", "pgdown":
			m.helpScroll += maxInt(1, (m.height-6)/2)
		case "ctrl+u", "pgup":
			m.helpScroll -= maxInt(1, (m.height-6)/2)
		}
		m.helpScroll = clampInt(m.helpScroll, 0, m.helpMaxScroll())
		return m, nil
	}

	if m.filterOn {
		return m.handleFilterKey(msg)
	}

	if m.paletteOn {
		return m.handlePaletteKey(msg)
	}

	if m.searching {
		return m.handleSearchKey(msg)
	}

	if m.paneFocus == 1 && (m.tab == TabTasks || m.tab == TabMaterials) {
		switch msg.String() {
		case " ", "a", "v", "V", "*", "t", "m", "i", "I", "n", "0", "d", "o":
			return m, nil
		}
	}
	switch msg.String() {
	case "q", "ctrl+c":
		m.cancelDownloads()
		return m, tea.Quit
	case "?":
		m.helpOn = true
		m.helpScroll = 0
	case ":", "ctrl+k":
		m.openPalette()
	case "tab":
		m.cyclePane(1)
	case "shift+tab":
		m.cyclePane(-1)
	case "1", "2", "3", "4":
		m.switchTab(Tab(msg.String()[0] - '1'))
	case "]":
		m.switchTab((m.tab + 1) % tabCount)
	case "[":
		m.switchTab((m.tab + tabCount - 1) % tabCount)
	case "up", "k":
		m.move(-1)
	case "down", "j":
		m.move(1)
	case "g", "home":
		m.moveTop()
	case "G", "end":
		m.moveBottom()
	case "ctrl+u", "pgup":
		m.movePage(-1)
	case "ctrl+d", "pgdown":
		m.movePage(1)
	case "left", "h":
		m.moveX(-1)
	case "right", "l":
		m.moveX(1)
	case " ":
		m.toggleSelect()
	case "a":
		m.selectAll()
	case "v", "V":
		m.startRange()
	case "*":
		m.invertVisible()
	case "t":
		return m, m.applyTriage(model.Tracked)
	case "m":
		return m, m.applyTriage(model.Missed)
	case "i", "I":
		return m, m.applyTriage(model.Ignored)
	case "n", "0":
		return m, m.applyTriage(model.Undecided)
	case "u", "ctrl+z":
		return m, m.undoTriage()
	case "c":
		m.openCommit()
	case "esc":
		if m.triageArmed {
			m.cancelTriageArm()
			m.toast = "triage cancelled"
			m.toastTicks = 20
			return m, nil
		}
		if m.statusErr != "" {
			m.statusErr = ""
			return m, nil
		}
		if m.detailVisible() {
			m.detailPane = 0
			m.paneFocus = 0
			return m, nil
		}
		if m.paneFocus != 0 {
			m.paneFocus = 0
			return m, nil
		}
		if m.visual {
			m.back()
		} else if m.tab == TabCommit {
			m.closeCommit()
		} else if !m.clearSearch() && !m.clearFilter() {
			if m.detailVisible() {
				m.toggleDetailPane()
			} else {
				m.back()
			}
		}
	case "enter":
		if m.paneFocus == 1 && (m.tab == TabTasks || m.tab == TabMaterials) {
			m.chooseCourse()
			return m, nil
		}
		if m.tab == TabCommit {
			if m.cfg.CommitFile != "" || m.cfg.ExportFormat == "taskwarrior" {
				return m, m.writeCommitPreview()
			}
			return m, m.copyCommitPreview()
		}
		m.enter()
	case "f":
		if m.tab == TabTasks || m.tab == TabMaterials {
			m.filterOn = true
			m.filterCursor = 0
			for i, f := range m.filterOptions() {
				if f == m.filter {
					m.filterCursor = i
				}
			}
		}
	case "S":
		m.toggleMaterialSort()
	case "A":
		m.toggleShowAll()
	case "/":
		if m.tab == TabTasks || m.tab == TabMaterials {
			m.cancelTriageArm()
			m.searching = true
			m.search = ""
			m.resetTreeCursor()
		}
	case "s", "r":
		return m, m.startSync()
	case "d":
		return m, m.download()
	case "o":
		return m, m.openExternal()
	case "O":
		return m, m.openLocalFile()
	case "w":
		if m.tab == TabCommit {
			return m, m.writeCommitPreview()
		}
	case "y":
		if m.tab == TabCommit {
			return m, m.copyCommitPreview()
		}
	case "x":
		if m.tab == TabCommit {
			m.dropCommitItem()
		}
	case "X":
		if m.tab == TabCommit {
			m.dropAllCommit()
		}
	}
	return m, nil
}
