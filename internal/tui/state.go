package tui

import "github.com/ruyam/kamps/internal/store"

// applyState folds the local store into freshly loaded data. The store wins
// over whatever the source reports, because it is the user's own decision.
func (m *App) applyState() {
	if m.cfg.AutoTrackSubmitted {
		m.state.Apply(m.assignments, m.materials)
		return
	}
	m.state.ApplyUserOnly(m.assignments, m.materials)
}

// updateState applies fn to the in-memory state and merges the same change into
// the on-disk state under the cross-process lock. Errors go to the toast rather
// than interrupting the action.
func (m *App) updateState(fn func(*store.State)) {
	fn(&m.state)
	if m.statePath == "" {
		return
	}
	err := store.Update(m.statePath, m.cfg.LockPath(), fn)
	if err != nil {
		m.toast = "could not save state: " + err.Error()
		m.toastTicks = 40
	}
}
