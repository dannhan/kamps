package tui

import (
	"sort"
	"strings"

	"github.com/ruyam/kamps/internal/cache"
	"github.com/ruyam/kamps/internal/workspace"
)

// loadCache replaces the current view with the last successful snapshot. It
// returns false when there is no usable cache for this site.
func (m *App) loadCache() bool {
	snapshot, savedAt, err := cache.Load(m.cfg.CachePath(), m.cfg.SiteURL)
	if err != nil {
		return false
	}
	m.loadNotice = snapshot.Notice
	m.cacheNotice = "cached from " + savedAt.Format("2006-01-02 15:04")
	m.lastSync = savedAt
	m.courses = snapshot.Courses
	resolved, notices := workspace.Resolve(m.courses, m.cfg.WorkspaceRoots)
	m.courses = resolved
	if len(notices) > 0 {
		if m.loadNotice != "" {
			m.loadNotice += "; "
		}
		m.loadNotice += strings.Join(notices, "; ")
	}
	m.assignments = snapshot.Assignments
	m.materials = snapshot.Materials
	sort.Slice(m.courses, func(i, j int) bool { return m.courses[i].Name < m.courses[j].Name })
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
	m.applyState()
	m.applyAutoTriage()
	m.reconcileLibrary()
	m.clampCursor()
	m.restoreCursors()
	return true
}
