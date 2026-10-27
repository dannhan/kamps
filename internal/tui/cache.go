package tui

import (
	"github.com/ruyam/kamps/internal/core"
)

// loadCache applies the cache synchronously during Init, before the first
// frame. Init is outside the input loop, so a small local read is acceptable
// there; subsequent reads happen inside loadCmd, outside Update.
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
