// Package cache stores the last successful snapshot so kamps remains readable
// when the LMS is down. It never stores credentials.
package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ruyam/kamps/internal/source"
)

const version = 1

type file struct {
	Version  int             `json:"version"`
	SiteURL  string          `json:"site_url"`
	SavedAt  time.Time       `json:"saved_at"`
	Snapshot source.Snapshot `json:"snapshot"`
}

// Save writes the snapshot atomically with mode 0600.
func Save(path, siteURL string, snapshot source.Snapshot) error {
	if path == "" {
		return errors.New("empty cache path")
	}
	if len(snapshot.Courses) == 0 && len(snapshot.Assignments) == 0 && len(snapshot.Materials) == 0 {
		return errors.New("refusing to cache an empty snapshot")
	}
	data, err := json.MarshalIndent(file{
		Version:  version,
		SiteURL:  siteURL,
		SavedAt:  time.Now(),
		Snapshot: snapshot,
	}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".cache-*.json")
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

// Load reads a cached snapshot. A site mismatch or unsupported version is an
// error so a cache from another LMS is never shown.
func Load(path, siteURL string) (source.Snapshot, time.Time, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return source.Snapshot{}, time.Time{}, err
	}
	var cached file
	if err := json.Unmarshal(data, &cached); err != nil {
		return source.Snapshot{}, time.Time{}, fmt.Errorf("parse cache: %w", err)
	}
	if cached.Version != version {
		return source.Snapshot{}, time.Time{}, fmt.Errorf("unsupported cache version %d", cached.Version)
	}
	if cached.SiteURL != siteURL {
		return source.Snapshot{}, time.Time{}, errors.New("cache belongs to a different site")
	}
	return cached.Snapshot, cached.SavedAt, nil
}
