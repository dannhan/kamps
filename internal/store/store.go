// Package store keeps the small slice of user intent that has to outlive a
// run: triage decisions and which materials already have a local copy.
// Everything else is regenerated from the source.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ruyam/kamps/internal/model"
)

const version = 1

// State is the persisted state file. Decisions map assignment IDs to triage
// values; downloads records material IDs that are already local. Missing
// entries mean "undecided" and "not downloaded".
type State struct {
	Version   int                     `json:"version"`
	Decisions map[string]model.Triage `json:"decisions,omitempty"`
	Downloads map[string]bool         `json:"downloads,omitempty"`
}

// New returns an empty state with its maps ready to use.
func New() State {
	return State{
		Version:   version,
		Decisions: map[string]model.Triage{},
		Downloads: map[string]bool{},
	}
}

// Load reads path. A missing file is not an error: the first run starts empty.
func Load(path string) (State, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return New(), nil
	}
	if err != nil {
		return New(), err
	}
	state := New()
	if err := json.Unmarshal(data, &state); err != nil {
		return New(), fmt.Errorf("parse %s: %w", path, err)
	}
	state.ensureMaps()
	for id, decision := range state.Decisions {
		switch decision {
		case model.Tracked, model.Missed, model.Ignored:
		default:
			delete(state.Decisions, id)
		}
	}
	return state, nil
}

func (s *State) ensureMaps() {
	if s.Decisions == nil {
		s.Decisions = map[string]model.Triage{}
	}
	if s.Downloads == nil {
		s.Downloads = map[string]bool{}
	}
}

// Decision returns the saved triage for an assignment. An absent entry is
// undecided.
func (s State) Decision(id string) model.Triage {
	return s.Decisions[id]
}

// SetDecision records a decision. Undecided is the default, so it deletes the
// entry instead of storing noise.
func (s *State) SetDecision(id string, t model.Triage) {
	s.ensureMaps()
	if t == model.Undecided {
		delete(s.Decisions, id)
		return
	}
	s.Decisions[id] = t
}

// SetDownloaded records whether a material has a local copy.
func (s *State) SetDownloaded(id string, downloaded bool) {
	s.ensureMaps()
	if downloaded {
		s.Downloads[id] = true
		return
	}
	delete(s.Downloads, id)
}

// Apply folds the saved state into freshly loaded data in place. The store wins
// over whatever the source reports, because it is the user's own decision.
func (s State) Apply(assignments []model.Assignment, materials []model.Material) {
	for i := range assignments {
		if decision := s.Decision(assignments[i].ID); decision != model.Undecided {
			assignments[i].Triage = decision
		}
	}
	for i := range materials {
		if s.Downloads[materials[i].ID] {
			materials[i].Downloaded = true
		}
	}
}

// Save writes the state atomically: a temp file in the same directory, then a
// rename, so a crash mid-write can never leave a half-parsed state file.
func (s State) Save(path string) error {
	s.Version = version
	s.ensureMaps()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".state-*.json")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(name, 0o600); err != nil {
		return err
	}
	return os.Rename(name, path)
}
