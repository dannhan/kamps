// Package config resolves kamps' settings and paths.
//
// For now it only knows sensible defaults. A real config.toml loader, auth
// settings and a configurable download library directory will land with the
// Moodle source; the shape here is deliberately small so nothing depends on
// file parsing yet.
package config

import (
	"os"
	"path/filepath"
)

const appName = "kamps"

// Config is the resolved runtime configuration.
type Config struct {
	// SiteURL is the LMS base URL (empty until configured).
	SiteURL string
	// LibraryDir is where downloaded materials are stored.
	LibraryDir string
	// StateDir holds kamps' own data (decisions, download flags).
	StateDir string
	// ConfigDir holds config.toml.
	ConfigDir string
	// NoAnim disables cheerful motion.
	NoAnim bool
}

// Default returns the XDG-ish default configuration.
func Default() Config {
	home, _ := os.UserHomeDir()
	return Config{
		SiteURL:    "",
		LibraryDir: filepath.Join(home, "Documents", appName),
		StateDir:   xdg("XDG_STATE_HOME", filepath.Join(home, ".local", "state")),
		ConfigDir:  xdg("XDG_CONFIG_HOME", filepath.Join(home, ".config")),
	}
}

// StatePath is the JSON file that keeps decisions and download flags between
// runs.
func (c Config) StatePath() string {
	if c.StateDir == "" {
		return ""
	}
	return filepath.Join(c.StateDir, "state.json")
}

func xdg(env, fallback string) string {
	if v := os.Getenv(env); v != "" {
		return filepath.Join(v, appName)
	}
	return filepath.Join(fallback, appName)
}
