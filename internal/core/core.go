// Package core owns kamps' application workflow: loading a source, merging
// persisted state, auto-triaging submitted work, reconciling the library, and
// exposing display-neutral snapshots to the TUI and headless commands.
package core

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ruyam/kamps/internal/cache"
	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/library"
	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/source"
	"github.com/ruyam/kamps/internal/store"
	"github.com/ruyam/kamps/internal/workspace"
)

// ScanSummary describes the last library reconciliation.
type ScanSummary struct {
	Found        int
	Missing      int
	Unknown      int
	UnknownFiles []string
	Text         string
}

// Snapshot is the complete, display-neutral result of a load. The TUI and
// plain renderer both consume this value.
type Snapshot struct {
	source.Snapshot
	LoadedAt  time.Time
	FromCache bool
	CachedAt  time.Time
	Scan      ScanSummary
	Warnings  []string
}

// Service is the UI-free application layer.
type Service struct {
	cfg       config.Config
	src       source.Source
	state     store.State
	statePath string
	stateErr  string
	snapshot  Snapshot
}

// New builds a service and loads the persisted state once.
func New(src source.Source, cfg config.Config) *Service {
	state := store.New()
	statePath := cfg.StatePath()
	stateErr := ""
	if statePath != "" {
		loaded, err := store.Load(statePath)
		if err != nil {
			stateErr = err.Error()
		} else {
			state = loaded
		}
	}
	return &Service{
		cfg:       cfg,
		src:       src,
		state:     state,
		statePath: statePath,
		stateErr:  stateErr,
	}
}

// Source returns the configured source.
func (s *Service) Source() source.Source { return s.src }

// Config returns the runtime configuration.
func (s *Service) Config() config.Config { return s.cfg }

// State returns the current in-memory persisted state.
func (s *Service) State() store.State { return s.state }

// StatePath returns the state file path, or "" when persistence is disabled.
func (s *Service) StatePath() string { return s.statePath }

// StateErr reports a state load failure from construction.
func (s *Service) StateErr() string { return s.stateErr }

// Snapshot returns the most recent loaded snapshot.
func (s *Service) Snapshot() Snapshot { return s.snapshot }

// Update mutates the in-memory state and merges the same change into the
// on-disk state under the cross-process lock.
func (s *Service) Update(fn func(*store.State)) error {
	fn(&s.state)
	if s.statePath == "" {
		return nil
	}
	return store.Update(s.statePath, s.cfg.LockPath(), fn)
}

// Load fetches a snapshot, runs the shared pipeline, and saves it to the cache.
func (s *Service) Load(ctx context.Context) (Snapshot, error) {
	raw, err := s.src.Load(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	snapshot := s.build(raw, time.Now(), false, time.Time{})
	if !strings.HasPrefix(s.src.Label(), "demo") {
		_ = cache.Save(s.cfg.CachePath(), s.cfg.SiteURL, raw)
	}
	return snapshot, nil
}

// LoadCached restores the last successful snapshot through the same pipeline.
func (s *Service) LoadCached() (Snapshot, bool) {
	raw, savedAt, err := cache.Load(s.cfg.CachePath(), s.cfg.SiteURL)
	if err != nil {
		return Snapshot{}, false
	}
	return s.build(raw, savedAt, true, savedAt), true
}

// Reconcile re-scans the library against the current snapshot.
func (s *Service) Reconcile() Snapshot {
	if err := s.reconcile(); err != nil {
		s.snapshot.Warnings = append(s.snapshot.Warnings, err.Error())
	}
	return s.snapshot
}

// Destination resolves a material's on-disk path, including workspace
// mappings and the schedule/library fallback.
func (s *Service) Destination(material model.Material) string {
	return Destination(s.cfg, s.snapshot.Courses, material)
}

// Destination resolves a material's on-disk path for a set of courses.
func Destination(cfg config.Config, courses []model.Course, material model.Material) string {
	courseName := material.CourseCode
	for _, course := range courses {
		if course.ID != material.CourseID {
			continue
		}
		if course.Short != "" {
			courseName = course.Short
		} else if course.Code != "" {
			courseName = course.Code
		}
		if course.WorkspacePath != "" {
			if path, err := library.SubPath(course.WorkspacePath, SectionLabel(material.Section), material.Folder, material.Name); err == nil {
				return path
			}
		}
		break
	}
	path, err := library.MaterialPath(cfg.LibraryDir, courseName, SectionLabel(material.Section), material.Folder, material.Name)
	if err != nil {
		return library.SectionPath(cfg.LibraryDir, courseName, SectionLabel(material.Section), material.Name)
	}
	return path
}

func (s *Service) build(raw source.Snapshot, loadedAt time.Time, fromCache bool, cachedAt time.Time) Snapshot {
	snapshot := Snapshot{
		Snapshot:  raw,
		LoadedAt:  loadedAt,
		FromCache: fromCache,
		CachedAt:  cachedAt,
	}
	resolved, notices := workspace.Resolve(raw.Courses, s.cfg.WorkspaceRoots)
	snapshot.Courses = resolved
	if len(notices) > 0 {
		if snapshot.Notice != "" {
			snapshot.Notice += "; "
		}
		snapshot.Notice += strings.Join(notices, "; ")
	}
	sort.Slice(snapshot.Courses, func(i, j int) bool { return snapshot.Courses[i].Name < snapshot.Courses[j].Name })
	sort.Slice(snapshot.Materials, func(i, j int) bool {
		a, b := snapshot.Materials[i], snapshot.Materials[j]
		if a.CourseCode != b.CourseCode {
			return a.CourseCode < b.CourseCode
		}
		if a.Section != b.Section {
			return SectionLess(a.Section, b.Section)
		}
		return a.Name < b.Name
	})
	s.snapshot = snapshot
	if !fromCache {
		if err := s.Update(func(state *store.State) { state.LastSync = loadedAt }); err != nil {
			s.snapshot.Warnings = append(s.snapshot.Warnings, "could not save state: "+err.Error())
		}
	}
	s.applyState()
	if err := s.applyAutoTriage(); err != nil {
		s.snapshot.Warnings = append(s.snapshot.Warnings, "could not save state: "+err.Error())
	}
	if err := s.reconcile(); err != nil {
		s.snapshot.Warnings = append(s.snapshot.Warnings, err.Error())
	}
	return s.snapshot
}

func (s *Service) applyState() {
	if s.cfg.AutoTrackSubmitted {
		s.state.Apply(s.snapshot.Assignments, s.snapshot.Materials)
		return
	}
	s.state.ApplyUserOnly(s.snapshot.Assignments, s.snapshot.Materials)
}

// applyAutoTriage derives Tracked for assignments the LMS marks submitted,
// without overriding an explicit user decision.
func (s *Service) applyAutoTriage() error {
	if !s.cfg.AutoTrackSubmitted {
		return nil
	}
	var set, clear []string
	for i := range s.snapshot.Assignments {
		assignment := &s.snapshot.Assignments[i]
		switch assignment.Submission {
		case model.SubmissionSubmitted:
			if s.state.Overrides[assignment.ID] {
				continue
			}
			if s.state.AutoDecisions[assignment.ID] == model.Tracked {
				continue
			}
			set = append(set, assignment.ID)
			assignment.Triage = model.Tracked
		case model.SubmissionNotSubmitted:
			if _, ok := s.state.AutoDecisions[assignment.ID]; !ok {
				continue
			}
			clear = append(clear, assignment.ID)
			if s.state.Decisions[assignment.ID] == model.Undecided && !s.state.Overrides[assignment.ID] {
				assignment.Triage = model.Undecided
			}
		}
	}
	if len(set) == 0 && len(clear) == 0 {
		return nil
	}
	return s.Update(func(state *store.State) {
		for _, id := range set {
			state.SetAutoDecision(id, model.Tracked)
		}
		for _, id := range clear {
			state.ClearAutoDecision(id)
		}
	})
}

// reconcile stat-checks every material and folds the result into the snapshot
// and persisted download records. Only the library root is walked for unexpected
// files; workspaces are user repositories.
func (s *Service) reconcile() error {
	if s.cfg.LibraryDir == "" || len(s.snapshot.Materials) == 0 {
		s.snapshot.Scan = ScanSummary{}
		return nil
	}
	result := library.Scan(s.snapshot.Materials, s.Destination, []string{s.cfg.LibraryDir})
	s.snapshot.Scan = ScanSummary{
		Found:        result.Found,
		Missing:      len(result.Missing),
		Unknown:      len(result.Unknown),
		UnknownFiles: result.Unknown,
		Text:         fmt.Sprintf("%d local · %d missing · %d unknown", result.Found, len(result.Missing), len(result.Unknown)),
	}
	changed := false
	for i := range s.snapshot.Materials {
		material := &s.snapshot.Materials[i]
		if _, ok := result.Downloaded[material.ID]; ok {
			material.Downloaded = true
			if _, exists := s.state.Downloads[material.ID]; !exists {
				changed = true
			}
			continue
		}
		if s.state.Downloaded(material.ID) {
			material.Downloaded = false
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return s.Update(func(state *store.State) {
		for id, record := range result.Downloaded {
			state.SetDownloadRecord(id, record)
		}
		for _, id := range result.Missing {
			state.ClearDownload(id)
		}
	})
}
