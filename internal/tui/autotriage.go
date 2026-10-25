package tui

import (
	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/store"
)

// applyAutoTriage derives Tracked for assignments the LMS marks submitted,
// without ever overriding an explicit user decision. It runs after applyState
// so a user override always wins.
func (m *App) applyAutoTriage() {
	if !m.cfg.AutoTrackSubmitted {
		return
	}
	var set, clear []string
	for i := range m.assignments {
		assignment := &m.assignments[i]
		switch assignment.Submission {
		case model.SubmissionSubmitted:
			if m.state.Overrides[assignment.ID] {
				continue
			}
			if m.state.AutoDecisions[assignment.ID] == model.Tracked {
				continue
			}
			set = append(set, assignment.ID)
			assignment.Triage = model.Tracked
		case model.SubmissionNotSubmitted:
			if _, ok := m.state.AutoDecisions[assignment.ID]; !ok {
				continue
			}
			clear = append(clear, assignment.ID)
			if m.state.Decisions[assignment.ID] == model.Undecided && !m.state.Overrides[assignment.ID] {
				assignment.Triage = model.Undecided
			}
		}
	}
	if len(set) == 0 && len(clear) == 0 {
		return
	}
	m.updateState(func(s *store.State) {
		for _, id := range set {
			s.SetAutoDecision(id, model.Tracked)
		}
		for _, id := range clear {
			s.ClearAutoDecision(id)
		}
	})
}
