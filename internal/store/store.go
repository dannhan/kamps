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
	"time"

	"github.com/ruyam/kamps/internal/model"
)

const version = 2

// DownloadRecord describes a material that has a local copy. Path and Size are
// filled in when kamps performed (or reconciled) the download; older files
// migrated from the v1 boolean map have an empty Path until the next scan.
type DownloadRecord struct {
	Path      string    `json:"path,omitempty"`
	Size      int64     `json:"size,omitempty"`
	ModTime   time.Time `json:"mod_time,omitempty"`
	ETag      string    `json:"etag,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// State is the persisted state file. Decisions and Downloads record user
// intent; AutoDecisions records triage derived from the LMS and Overrides
// protects explicit user choices (including a deliberate reset to undecided)
// from being overwritten on the next load.
type State struct {
	Version       int                       `json:"version"`
	Decisions     map[string]model.Triage   `json:"decisions,omitempty"`
	AutoDecisions map[string]model.Triage   `json:"auto_decisions,omitempty"`
	Overrides     map[string]bool           `json:"overrides,omitempty"`
	Downloads     map[string]DownloadRecord `json:"downloads,omitempty"`
	LastSync      time.Time                 `json:"last_sync,omitempty"`
}

// New returns an empty state with its maps ready to use.
func New() State {
	return State{
		Version:       version,
		Decisions:     map[string]model.Triage{},
		AutoDecisions: map[string]model.Triage{},
		Overrides:     map[string]bool{},
		Downloads:     map[string]DownloadRecord{},
	}
}

// Load reads path. A missing file is not an error: the first run starts empty.
// Version 1 files are migrated in memory; a best-effort .bak copy is written
// before the next save replaces them.
func Load(path string) (State, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return New(), nil
	}
	if err != nil {
		return New(), err
	}
	state, migrated, err := decode(data)
	if err != nil {
		return New(), fmt.Errorf("parse %s: %w", path, err)
	}
	if migrated {
		backupPath := path + ".bak"
		if _, statErr := os.Stat(backupPath); errors.Is(statErr, os.ErrNotExist) {
			_ = os.WriteFile(backupPath, data, 0o600)
		}
	}
	return state, nil
}

func decode(data []byte) (State, bool, error) {
	var probe struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return New(), false, err
	}
	if probe.Version > version {
		return New(), false, fmt.Errorf("unsupported state version %d (this kamps supports up to %d)", probe.Version, version)
	}

	if probe.Version < 2 {
		var old struct {
			Version   int                     `json:"version"`
			Decisions map[string]model.Triage `json:"decisions,omitempty"`
			Downloads map[string]bool         `json:"downloads,omitempty"`
		}
		if err := json.Unmarshal(data, &old); err != nil {
			return New(), false, err
		}
		state := New()
		if old.Decisions != nil {
			state.Decisions = old.Decisions
		}
		for id, downloaded := range old.Downloads {
			if downloaded {
				state.Downloads[id] = DownloadRecord{}
			}
		}
		state.sanitize()
		return state, true, nil
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return New(), false, err
	}
	state.Version = version
	state.ensureMaps()
	state.sanitize()
	return state, false, nil
}

func (s *State) ensureMaps() {
	if s.Decisions == nil {
		s.Decisions = map[string]model.Triage{}
	}
	if s.AutoDecisions == nil {
		s.AutoDecisions = map[string]model.Triage{}
	}
	if s.Overrides == nil {
		s.Overrides = map[string]bool{}
	}
	if s.Downloads == nil {
		s.Downloads = map[string]DownloadRecord{}
	}
}

// sanitize drops values that are not valid persisted decisions.
func (s *State) sanitize() {
	for id, decision := range s.Decisions {
		switch decision {
		case model.Tracked, model.Missed, model.Ignored:
		default:
			delete(s.Decisions, id)
		}
	}
	for id, decision := range s.AutoDecisions {
		switch decision {
		case model.Tracked, model.Missed, model.Ignored:
		default:
			delete(s.AutoDecisions, id)
		}
	}
}

// Decision returns the saved user triage for an assignment. An absent entry is
// undecided.
func (s State) Decision(id string) model.Triage {
	return s.Decisions[id]
}

// SetDecision records a user decision. Undecided is the default, so it deletes
// the entry instead of storing noise; the override marker remains so automatic
// triage cannot override the user's choice later.
func (s *State) SetDecision(id string, t model.Triage) {
	s.ensureMaps()
	s.Overrides[id] = true
	delete(s.AutoDecisions, id)
	if t == model.Undecided {
		delete(s.Decisions, id)
		return
	}
	s.Decisions[id] = t
}

// SetAutoDecision records triage derived from the LMS, unless the user has
// explicitly decided on the item.
func (s *State) SetAutoDecision(id string, t model.Triage) {
	s.ensureMaps()
	if s.Overrides[id] {
		return
	}
	if t == model.Undecided {
		delete(s.AutoDecisions, id)
		return
	}
	s.AutoDecisions[id] = t
}

// ClearAutoDecision removes an automatic decision, leaving user decisions alone.
func (s *State) ClearAutoDecision(id string) {
	delete(s.AutoDecisions, id)
}

// Downloaded reports whether a material has a local copy recorded.
func (s State) Downloaded(id string) bool {
	_, ok := s.Downloads[id]
	return ok
}

// SetDownloaded records whether a material has a local copy. New records carry
// a timestamp; callers with a known path should use SetDownloadRecord.
func (s *State) SetDownloaded(id string, downloaded bool) {
	s.ensureMaps()
	if !downloaded {
		delete(s.Downloads, id)
		return
	}
	if _, ok := s.Downloads[id]; !ok {
		s.Downloads[id] = DownloadRecord{UpdatedAt: time.Now()}
	}
}

// SetDownloadRecord records a material's local copy details.
func (s *State) SetDownloadRecord(id string, record DownloadRecord) {
	s.ensureMaps()
	record.UpdatedAt = time.Now()
	s.Downloads[id] = record
}

// ClearDownload removes a material's local-copy record.
func (s *State) ClearDownload(id string) {
	delete(s.Downloads, id)
}

// Apply folds the saved state into freshly loaded data in place. User decisions
// win; automatic decisions only fill assignments that are still undecided and
// have not been overridden. The store wins over whatever the source reports,
// because it is the user's own decision.
func (s State) Apply(assignments []model.Assignment, materials []model.Material) {
	s.apply(assignments, materials, true)
}

// ApplyUserOnly applies user decisions and download records but ignores
// automatic triage, used when auto_track_submitted is disabled.
func (s State) ApplyUserOnly(assignments []model.Assignment, materials []model.Material) {
	s.apply(assignments, materials, false)
}

func (s State) apply(assignments []model.Assignment, materials []model.Material, auto bool) {
	for i := range assignments {
		id := assignments[i].ID
		if decision := s.Decisions[id]; decision != model.Undecided {
			assignments[i].Triage = decision
			continue
		}
		if auto && !s.Overrides[id] {
			if decision := s.AutoDecisions[id]; decision != model.Undecided {
				assignments[i].Triage = decision
			}
		}
	}
	for i := range materials {
		if _, ok := s.Downloads[materials[i].ID]; ok {
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
