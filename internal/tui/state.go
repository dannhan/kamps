package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ruyam/kamps/internal/core"
	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/store"
)

// stateSavedMsg reports the result of the one in-flight persistence command.
// The maps name exactly which values were written so later decisions made while
// the write was running are kept pending.
type stateSavedMsg struct {
	decisions map[string]model.Triage
	downloads map[string]bool
	err       error
}

// persistStateCmd is the only place the TUI writes state to disk. It runs in a
// Bubble Tea command goroutine, never in Update.
func persistStateCmd(service *core.Service, decisions map[string]model.Triage, downloads map[string]bool) tea.Cmd {
	return func() tea.Msg {
		if len(decisions) == 0 && len(downloads) == 0 {
			return stateSavedMsg{}
		}
		err := service.Persist(func(state *store.State) {
			for id, triage := range decisions {
				state.SetDecision(id, triage)
			}
			for id := range downloads {
				state.SetDownloaded(id, true)
			}
		})
		return stateSavedMsg{decisions: decisions, downloads: downloads, err: err}
	}
}

// queueStatePersist starts one persistence command for the pending changes.
// Keeping a single command in flight preserves the order of same-key updates.
func (m *App) queueStatePersist() tea.Cmd {
	if m.persistInFlight || (len(m.pendingDecisions) == 0 && len(m.pendingDownloads) == 0) {
		return nil
	}
	decisions := make(map[string]model.Triage, len(m.pendingDecisions))
	for id, triage := range m.pendingDecisions {
		decisions[id] = triage
	}
	downloads := make(map[string]bool, len(m.pendingDownloads))
	for id := range m.pendingDownloads {
		downloads[id] = true
	}
	m.persistInFlight = true
	return persistStateCmd(m.service, decisions, downloads)
}

// handleStateSaved releases the in-flight slot and starts another write when a
// newer decision arrived while the previous one was on disk.
func (m *App) handleStateSaved(msg stateSavedMsg) tea.Cmd {
	m.persistInFlight = false
	if msg.err != nil {
		m.toast = "could not save state"
		m.statusErr = "could not save state: " + msg.err.Error()
		m.toastTicks = 40
		return nil
	}
	if strings.HasPrefix(m.statusErr, "could not save state") {
		m.statusErr = ""
	}
	if m.toast == "could not save state" {
		m.toast = ""
		m.toastTicks = 0
	}
	for id, triage := range msg.decisions {
		if current, ok := m.pendingDecisions[id]; ok && current == triage {
			delete(m.pendingDecisions, id)
		}
	}
	for id := range msg.downloads {
		if m.pendingDownloads[id] {
			delete(m.pendingDownloads, id)
		}
	}
	return m.queueStatePersist()
}
