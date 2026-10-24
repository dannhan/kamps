package tui

import tea "github.com/charmbracelet/bubbletea"

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

func (m *App) syncContent(width, height int) (string, string, []string, []string) {
	return "sync status · " + m.src.Label(), "", []string{"↑↓ move", "s sync"}, m.syncBody(width, height)
}
