package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type paletteAction struct {
	label string
	run   func(*App) tea.Cmd
}

func (m *App) paletteActions() []paletteAction {
	actions := []paletteAction{
		{"inspect details (Enter)", func(app *App) tea.Cmd { app.toggleDetailPane(); return nil }},
		{"sync status (3)", func(app *App) tea.Cmd { app.switchTab(TabSync); return nil }},
		{"sync now", func(app *App) tea.Cmd { return app.startSync() }},
		{"go to tasks", func(app *App) tea.Cmd { app.switchTab(TabTasks); return nil }},
		{"go to materials", func(app *App) tea.Cmd { app.switchTab(TabMaterials); return nil }},
		{"go to commit", func(app *App) tea.Cmd { app.openCommit(); return nil }},
		{"download focused", func(app *App) tea.Cmd { return app.download() }},
		{"open local file (O)", func(app *App) tea.Cmd { return app.openLocalFile() }},
		{"open link", func(app *App) tea.Cmd { return app.openExternal() }},
		{"write queue", func(app *App) tea.Cmd { return app.writeCommitPreview() }},
		{"copy queue", func(app *App) tea.Cmd { return app.copyCommitPreview() }},
		{"help", func(app *App) tea.Cmd { app.helpOn = true; app.helpScroll = 0; return nil }},
		{"quit", func(app *App) tea.Cmd { app.cancelDownloads(); return tea.Quit }},
	}
	switch m.tab {
	case TabTasks:
		actions = append(actions,
			paletteAction{"filter all", func(app *App) tea.Cmd { app.setPaletteFilter("all"); return nil }},
			paletteAction{"filter undecided", func(app *App) tea.Cmd { app.setPaletteFilter("undecided"); return nil }},
			paletteAction{"filter overdue", func(app *App) tea.Cmd { app.setPaletteFilter("overdue"); return nil }},
			paletteAction{"filter tracked", func(app *App) tea.Cmd { app.setPaletteFilter("tracked"); return nil }},
			paletteAction{"filter missed", func(app *App) tea.Cmd { app.setPaletteFilter("missed"); return nil }},
			paletteAction{"filter ignored", func(app *App) tea.Cmd { app.setPaletteFilter("ignored"); return nil }},
		)
	case TabMaterials:
		actions = append(actions,
			paletteAction{"reverse week order (S)", func(app *App) tea.Cmd { app.toggleMaterialSort(); return nil }},
			paletteAction{"filter all", func(app *App) tea.Cmd { app.setPaletteFilter("all"); return nil }},
			paletteAction{"filter new", func(app *App) tea.Cmd { app.setPaletteFilter("new"); return nil }},
			paletteAction{"filter downloaded", func(app *App) tea.Cmd { app.setPaletteFilter("downloaded"); return nil }},
			paletteAction{"filter links", func(app *App) tea.Cmd { app.setPaletteFilter("link"); return nil }},
		)
	}
	return actions
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
	for _, action := range m.paletteActions() {
		if query == "" || strings.Contains(strings.ToLower(action.label), query) {
			out = append(out, action)
		}
	}
	return out
}

func (m *App) paletteLines(width int) []string {
	actions := m.filteredPaletteActions()
	cursor := clampInt(m.paletteCursor, 0, maxInt(0, len(actions)-1))
	lines := []string{stFaint.Render("> ") + stRow.Render(m.paletteQuery), ""}
	if len(actions) == 0 {
		return append(lines, stEmpty.Render("no matching commands"))
	}
	start, end := windowRange(len(actions), cursor, maxInt(1, bodyHeightFor(m.width, m.height)-2))
	for i := start; i < end; i++ {
		action := actions[i]
		line := "  " + action.label
		if i == cursor {
			lines = append(lines, paneRow(stCursor.Render(line), width, rowHighlight{cursor: true}))
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
