package tui

import "github.com/ruyam/kamps/internal/store"

// updateState applies fn through the shared core service and refreshes the
// local copy used by rendering. Errors go to the toast rather than interrupting
// the action.
func (m *App) updateState(fn func(*store.State)) {
	if err := m.service.Update(fn); err != nil {
		m.toast = "could not save state: " + err.Error()
		m.toastTicks = 40
	}
	m.state = m.service.State()
}
