package tui

// loadCache replaces the current view with the last successful snapshot. It
// returns false when there is no usable cache for this site.
func (m *App) loadCache() bool {
	snapshot, ok := m.service.LoadCached()
	if !ok {
		return false
	}
	m.applySnapshot(snapshot)
	m.cacheNotice = snapshot.CachedAt.Format("2006-01-02 15:04")
	return true
}
