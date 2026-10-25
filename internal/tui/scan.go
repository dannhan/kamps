package tui

import (
	"fmt"

	"github.com/ruyam/kamps/internal/library"
	"github.com/ruyam/kamps/internal/store"
)

// reconcileLibrary stat-checks the files behind every material and folds the
// result into both the in-memory list and the persisted download records.
func (m *App) reconcileLibrary() {
	if m.cfg.LibraryDir == "" || len(m.materials) == 0 {
		return
	}
	// Known files are resolved through materialDestination (including mapped
	// workspaces); only the library root is walked for unexpected files. A
	// workspace is a user's repo, so its source files are not "unknown".
	result := library.Scan(m.materials, m.materialDestination, []string{m.cfg.LibraryDir})
	m.unknownFiles = result.Unknown
	changed := false
	for i := range m.materials {
		material := &m.materials[i]
		if _, ok := result.Downloaded[material.ID]; ok {
			material.Downloaded = true
			if _, exists := m.state.Downloads[material.ID]; !exists {
				changed = true
			}
			continue
		}
		if m.state.Downloaded(material.ID) {
			material.Downloaded = false
			changed = true
		}
	}
	if changed {
		m.updateState(func(s *store.State) {
			for id, record := range result.Downloaded {
				s.SetDownloadRecord(id, record)
			}
			for _, id := range result.Missing {
				s.ClearDownload(id)
			}
		})
	}
	m.scanSummary = fmt.Sprintf("%d local · %d missing · %d unknown", result.Found, len(result.Missing), len(result.Unknown))
}
