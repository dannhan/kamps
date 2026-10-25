package core

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ruyam/kamps/internal/export"
	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/source"
	"github.com/ruyam/kamps/internal/store"
)

// Decision describes one assignment changed by Decide.
type Decision struct {
	ID     string       `json:"id"`
	Triage model.Triage `json:"triage"`
}

// Decide persists a triage decision for every matching assignment.
func (s *Service) Decide(ids []string, triage model.Triage) ([]Decision, error) {
	wanted := make(map[string]bool, len(ids))
	for _, id := range ids {
		wanted[id] = true
	}
	var changed []Decision
	for i := range s.snapshot.Assignments {
		assignment := &s.snapshot.Assignments[i]
		if !wanted[assignment.ID] {
			continue
		}
		assignment.Triage = triage
		changed = append(changed, Decision{ID: assignment.ID, Triage: triage})
	}
	if len(changed) == 0 {
		return nil, fmt.Errorf("no matching assignments")
	}
	err := s.Update(func(state *store.State) {
		for _, decision := range changed {
			state.SetDecision(decision.ID, decision.Triage)
		}
	})
	return changed, err
}

// DownloadProgress receives per-material byte progress.
type DownloadProgress func(id string, done, total int64)

// DownloadResult reports one completed transfer.
type DownloadResult struct {
	ID    string `json:"id"`
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}

// Download fetches the requested materials through the source downloader.
func (s *Service) Download(ctx context.Context, ids []string, progress DownloadProgress) ([]DownloadResult, error) {
	downloader, ok := s.src.(source.Downloader)
	if !ok {
		return nil, fmt.Errorf("source %q cannot download files", s.src.Label())
	}
	wanted := make(map[string]bool, len(ids))
	for _, id := range ids {
		wanted[id] = true
	}
	var results []DownloadResult
	var failures []string
	records := map[string]store.DownloadRecord{}
	for i := range s.snapshot.Materials {
		material := &s.snapshot.Materials[i]
		if !wanted[material.ID] {
			continue
		}
		path, err := downloader.Download(ctx, *material, s.Destination(*material), func(done, total int64) {
			if progress != nil {
				progress(material.ID, done, total)
			}
		})
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", material.ID, err))
			continue
		}
		material.Downloaded = true
		size := int64(0)
		if info, err := os.Stat(path); err == nil {
			size = info.Size()
		}
		records[material.ID] = store.DownloadRecord{Path: path, Size: size}
		results = append(results, DownloadResult{ID: material.ID, Path: path, Bytes: size})
	}
	if len(records) > 0 {
		if err := s.Update(func(state *store.State) {
			for id, record := range records {
				state.SetDownloadRecord(id, record)
			}
		}); err != nil {
			failures = append(failures, "save state: "+err.Error())
		}
	}
	if len(failures) > 0 {
		return results, fmt.Errorf("%s", strings.Join(failures, "; "))
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no matching materials")
	}
	return results, nil
}

// ExportItems builds export items from explicit IDs, or from tracked
// assignments when ids is empty.
func (s *Service) ExportItems(ids []string) []export.Item {
	wanted := make(map[string]bool, len(ids))
	for _, id := range ids {
		wanted[id] = true
	}
	items := make([]export.Item, 0, len(s.snapshot.Assignments))
	for _, assignment := range s.snapshot.Assignments {
		if len(wanted) > 0 {
			if !wanted[assignment.ID] {
				continue
			}
		} else if assignment.Triage != model.Tracked {
			continue
		}
		courseLabel := assignment.CourseCode
		if course, ok := findCourse(s.snapshot.Courses, assignment.CourseID); ok {
			if course.Short != "" {
				courseLabel = course.Short
			} else if course.Code != "" {
				courseLabel = course.Code
			}
		}
		items = append(items, export.Item{
			ID:     assignment.ID,
			Course: courseLabel,
			Name:   assignment.Name,
			Due:    assignment.Due,
			HasDue: assignment.HasDue,
			Action: assignment.Triage,
		})
	}
	return items
}

// Export renders and writes the selected items using the configured writer.
func (s *Service) Export(format, path string, ids []string) (export.Result, error) {
	if format == "" {
		format = s.cfg.ExportFormat
	}
	writer, err := export.For(format)
	if err != nil {
		return export.Result{}, err
	}
	if path == "" {
		path = s.cfg.CommitFile
	}
	return writer.Write(path, s.ExportItems(ids))
}

// Status summarizes the loaded snapshot and session.
type Status struct {
	Source           string     `json:"source"`
	SessionOK        bool       `json:"session_ok"`
	SessionExpiresAt *time.Time `json:"session_expires_at,omitempty"`
	LastSync         *time.Time `json:"last_sync,omitempty"`
	FromCache        bool       `json:"from_cache"`
	CachedAt         *time.Time `json:"cached_at,omitempty"`
	Courses          int        `json:"courses"`
	Assignments      int        `json:"assignments"`
	Materials        int        `json:"materials"`
	Downloaded       int        `json:"downloaded"`
}

// Status returns the current summary without exposing the session cookie.
func (s *Service) Status() Status {
	status := Status{
		Source:      s.src.Label(),
		FromCache:   s.snapshot.FromCache,
		Courses:     len(s.snapshot.Courses),
		Assignments: len(s.snapshot.Assignments),
		Materials:   len(s.snapshot.Materials),
	}
	if !s.state.LastSync.IsZero() {
		value := s.state.LastSync
		status.LastSync = &value
	}
	if !s.snapshot.CachedAt.IsZero() {
		value := s.snapshot.CachedAt
		status.CachedAt = &value
	}
	for _, material := range s.snapshot.Materials {
		if material.Downloaded {
			status.Downloaded++
		}
	}
	if reporter, ok := s.src.(source.SessionReporter); ok {
		session := reporter.SessionStatus()
		status.SessionOK = session.OK
		if !session.ExpiresAt.IsZero() {
			value := session.ExpiresAt
			status.SessionExpiresAt = &value
		}
	}
	return status
}
