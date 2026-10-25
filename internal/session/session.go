// Package session stores the Moodle session cookie and the small identity
// values obtained from the LMS. It is deliberately separate from store.State:
// the cookie is a credential, not user intent.
package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const version = 1

// ErrNotFound means no session has been saved yet.
var ErrNotFound = errors.New("no saved session")

// DefaultLifetime is Moodle's usual sessiontimeout (8 hours) when the server
// does not report one.
const DefaultLifetime = 8 * time.Hour

// Session is the persisted login. Cookie is the MoodleSession value.
type Session struct {
	Version   int       `json:"version"`
	SiteURL   string    `json:"site_url"`
	Cookie    string    `json:"cookie"`
	UserID    int       `json:"user_id,omitempty"`
	SessKey   string    `json:"sesskey,omitempty"`
	SavedAt   time.Time `json:"saved_at"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// Expiry returns the best-known expiry for the session.
func (s Session) Expiry() time.Time {
	if !s.ExpiresAt.IsZero() {
		return s.ExpiresAt
	}
	if s.SavedAt.IsZero() {
		return time.Time{}
	}
	return s.SavedAt.Add(DefaultLifetime)
}

// Remaining returns the time left before the session is likely expired.
func (s Session) Remaining(now time.Time) time.Duration {
	expiry := s.Expiry()
	if expiry.IsZero() {
		return 0
	}
	return expiry.Sub(now)
}

// Load reads path. A missing file returns ErrNotFound.
func Load(path string) (Session, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, err
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return Session{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if s.Version != version {
		return Session{}, fmt.Errorf("%s: unsupported session version %d", path, s.Version)
	}
	if err := validateCookie(s.Cookie); err != nil {
		return Session{}, fmt.Errorf("%s: %w", path, err)
	}
	return s, nil
}

// Save writes the session atomically with mode 0600.
func Save(path string, s Session) error {
	if path == "" {
		return errors.New("empty session path")
	}
	if err := validateCookie(s.Cookie); err != nil {
		return err
	}
	s.Version = version
	if s.SavedAt.IsZero() {
		s.SavedAt = time.Now()
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".session-*.json")
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

// Delete removes the saved session. It reports whether a file existed.
func Delete(path string) (bool, error) {
	if path == "" {
		return false, nil
	}
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func validateCookie(value string) error {
	if value == "" {
		return errors.New("empty MoodleSession cookie")
	}
	cookie := &http.Cookie{Name: "MoodleSession", Value: value}
	if err := cookie.Valid(); err != nil {
		return fmt.Errorf("invalid MoodleSession cookie: %w", err)
	}
	return nil
}
