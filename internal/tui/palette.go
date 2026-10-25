package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type paletteAction struct {
	label string
	run   func(*App) tea.Cmd
}

func paletteActions() []paletteAction {
	return []paletteAction{
		{"sync now", func(m *App) tea.Cmd { return m.startSync() }},
		{"go to tasks", func(m *App) tea.Cmd { m.switchTab(TabTasks); return nil }},
		{"go to materials", func(m *App) tea.Cmd { m.switchTab(TabMaterials); return nil }},
		{"go to commit", func(m *App) tea.Cmd { m.openCommit(); return nil }},
		{"filter all", func(m *App) tea.Cmd { m.setPaletteFilter("all"); return nil }},
		{"filter undecided", func(m *App) tea.Cmd { m.setPaletteFilter("undecided"); return nil }},
		{"filter overdue", func(m *App) tea.Cmd { m.setPaletteFilter("overdue"); return nil }},
		{"filter tracked", func(m *App) tea.Cmd { m.setPaletteFilter("tracked"); return nil }},
		{"filter missed", func(m *App) tea.Cmd { m.setPaletteFilter("missed"); return nil }},
		{"filter ignored", func(m *App) tea.Cmd { m.setPaletteFilter("ignored"); return nil }},
		{"download focused", func(m *App) tea.Cmd { return m.download() }},
		{"open link", func(m *App) tea.Cmd { return m.openExternal() }},
		{"write queue", func(m *App) tea.Cmd { return m.writeCommitPreview() }},
		{"copy queue", func(m *App) tea.Cmd { return m.copyCommitPreview() }},
		{"help", func(m *App) tea.Cmd { m.helpOn = true; return nil }},
		{"quit", func(m *App) tea.Cmd { m.cancelDownloads(); return tea.Quit }},
	}
}

func (m *App) openPalette() {
	m.paletteOn = true
	m.paletteQuery = ""
	m.paletteCursor = 0
}

func (m *App) closePalette() {
	m.paletteOn = false
	m.paletteQuery = ""
	m.paletteCursor = 0
}

func (m *App) setPaletteFilter(filter string) {
	m.filter = filter
	m.tabFilters[m.tab] = filter
	if m.tab == TabTasks {
		m.taskCursor = 0
	} else {
		m.matCursor = 0
	}
}

func (m *App) filteredPaletteActions() []paletteAction {
	query := strings.ToLower(strings.TrimSpace(m.paletteQuery))
	var out []paletteAction
	for _, action := range paletteActions() {
		if query == "" || strings.Contains(strings.ToLower(action.label), query) {
			out = append(out, action)
		}
	}
	return out
}

func (m *App) paletteLines(width int) []string {
	actions := m.filteredPaletteActions()
	m.paletteCursor = clampInt(m.paletteCursor, 0, maxInt(0, len(actions)-1))
	lines := []string{stFaint.Render("> ") + stRow.Render(m.paletteQuery), ""}
	if len(actions) == 0 {
		return append(lines, stEmpty.Render("no matching commands"))
	}
	for i, action := range actions {
		line := "  " + action.label
		if i == m.paletteCursor {
			lines = append(lines, sharedFocusRow(stCursor.Render(line), width, true))
		} else {
			lines = append(lines, line)
		}
	}
	return lines
}

func (m *App) handlePaletteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.closePalette()
		return m, nil
	case "enter":
		actions := m.filteredPaletteActions()
		if len(actions) == 0 {
			return m, nil
		}
		m.paletteCursor = clampInt(m.paletteCursor, 0, len(actions)-1)
		action := actions[m.paletteCursor]
		m.closePalette()
		return m, action.run(m)
	case "up", "ctrl+p":
		m.paletteCursor--
	case "down", "ctrl+n":
		m.paletteCursor++
	case "backspace":
		runes := []rune(m.paletteQuery)
		if len(runes) > 0 {
			m.paletteQuery = string(runes[:len(runes)-1])
		}
	case "ctrl+u":
		m.paletteQuery = ""
	default:
		if msg.Type == tea.KeyRunes {
			m.paletteQuery += string(msg.Runes)
		} else if msg.Type == tea.KeySpace {
			m.paletteQuery += " "
		}
	}
	m.paletteCursor = clampInt(m.paletteCursor, 0, maxInt(0, len(m.filteredPaletteActions())-1))
	return m, nil
}
