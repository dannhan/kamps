package tui

// applyState folds the local store into freshly loaded data. The store wins
// over whatever the source reports, because it is the user's own decision.
func (m *App) applyState() {
	m.state.Apply(m.assignments, m.materials)
}

// saveState persists decisions and download flags. Errors go to the toast
// rather than interrupting the action; the next successful save retries.
func (m *App) saveState() {
	if m.statePath == "" {
		return
	}
	if err := m.state.Save(m.statePath); err != nil {
		m.toast = "could not save state: " + err.Error()
		m.toastTicks = 40
	}
}
