package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ruyam/kamps/internal/source"
)

func (m *App) startSync() tea.Cmd {
	if m.syncing || m.loading {
		return nil
	}
	m.syncing = true
	m.loading = true
	m.syncTicks = 0
	m.captureCursors()
	return tea.Batch(m.spinner.Tick, loadCmd(m.src))
}

func (m *App) animating() bool {
	return m.syncing || m.toast != ""
}

func (m *App) advance() {
	if m.syncing {
		m.syncTicks++
	}
	if m.toast != "" && m.toastTicks > 0 {
		m.toastTicks--
		if m.toastTicks == 0 {
			m.toast = ""
		}
	}
}

// captureCursors remembers the stable keys under each cursor so a reload can
// put the highlight back where it was.
func (m *App) captureCursors() {
	m.restoreTaskKey, m.restoreMatKey, m.restoreCommitID = "", "", ""
	if nodes := m.taskNodes(); m.taskCursor >= 0 && m.taskCursor < len(nodes) {
		m.restoreTaskKey = nodes[m.taskCursor].key
	}
	if nodes := m.matNodes(); m.matCursor >= 0 && m.matCursor < len(nodes) {
		m.restoreMatKey = nodes[m.matCursor].key
	}
	if items := m.commitItems(); m.commitCursor >= 0 && m.commitCursor < len(items) {
		m.restoreCommitID = items[m.commitCursor].assignment.ID
	}
}

func (m *App) restoreCursors() {
	if m.restoreTaskKey != "" {
		for i, n := range m.taskNodes() {
			if n.key == m.restoreTaskKey {
				m.taskCursor = i
				break
			}
		}
	}
	if m.restoreMatKey != "" {
		for i, n := range m.matNodes() {
			if n.key == m.restoreMatKey {
				m.matCursor = i
				break
			}
		}
	}
	if m.restoreCommitID != "" {
		for i, item := range m.commitItems() {
			if item.assignment.ID == m.restoreCommitID {
				m.commitCursor = i
				break
			}
		}
	}
	m.restoreTaskKey, m.restoreMatKey, m.restoreCommitID = "", "", ""
	m.clampCursor()
}

func (m *App) sessionLine() string {
	reporter, ok := m.src.(source.SessionReporter)
	if !ok {
		return "not managed by this source"
	}
	status := reporter.SessionStatus()
	if !status.OK || status.ExpiresAt.IsZero() {
		return "unknown"
	}
	remaining := time.Until(status.ExpiresAt)
	if remaining <= 0 {
		return "expired — run kamps login"
	}
	return fmt.Sprintf("ok (expires in %s)", humanDuration(remaining))
}

func humanDuration(d time.Duration) string {
	if d >= 24*time.Hour {
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
	if d >= time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	minutes := int(d.Minutes())
	if minutes < 1 {
		return "<1m"
	}
	return fmt.Sprintf("%dm", minutes)
}

func (m *App) syncBody(width, height int) []string {
	state := "idle"
	if m.syncing {
		state = "syncing"
	}
	if m.offline {
		state = "offline"
	}
	last := "never"
	if !m.lastSync.IsZero() {
		last = m.lastSync.Format("2006-01-02 15:04")
	}
	values := [][2]string{
		{"state", state},
		{"source", m.src.Label()},
		{"last sync", last},
		{"library", m.scanSummary},
		{"cookie", m.sessionLine()},
	}
	if m.loadNotice != "" {
		values = append(values, [2]string{"notice", m.loadNotice})
	}
	if m.cacheNotice != "" {
		values = append(values, [2]string{"offline", m.cacheNotice})
	}
	start, end := windowRange(len(values), m.syncCursor, height)
	rows := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		focused := i == m.syncCursor
		highlight := rowHighlight{cursor: focused}
		background := highlight.style
		label := background(stDim).Render(pad(values[i][0], 10, false))
		value := background(stPrimary).Render(values[i][1])
		row := highlight.gutter() + label + highlight.gap() + value
		rows = append(rows, row+highlight.blanks(maxInt(0, width-lipgloss.Width(row))))
	}
	return rows
}

func (m *App) syncContent(width, height int) (string, string, []string, []string) {
	return "sync status · " + m.src.Label(), "", []string{"↑↓ move", "s sync"}, m.syncBody(width, height)
}
