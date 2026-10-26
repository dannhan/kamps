package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ruyam/kamps/internal/core"
)

// cacheLoadedMsg carries the result of reading the offline cache. The read
// happens in a command so a slow disk never blocks Update.
type cacheLoadedMsg struct {
	snapshot core.Snapshot
	ok       bool
}

func loadCachedCmd(service *core.Service) tea.Cmd {
	return func() tea.Msg {
		snapshot, ok := service.LoadCached()
		return cacheLoadedMsg{snapshot: snapshot, ok: ok}
	}
}

// loadCache applies the cache synchronously during Init, before the first
// frame. Init is outside the input loop, so a small local read is acceptable
// there; Update uses loadCachedCmd instead.
func (m *App) loadCache() bool {
	snapshot, ok := m.service.LoadCached()
	if !ok {
		return false
	}
	m.applyCachedSnapshot(snapshot)
	return true
}

func (m *App) applyCachedSnapshot(snapshot core.Snapshot) {
	m.applySnapshot(snapshot)
	m.cacheNotice = snapshot.CachedAt.Format("2006-01-02 15:04")
}
