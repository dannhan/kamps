// Package config resolves kamps' settings and paths.
//
// Defaults work without a file. A user can override the LMS URL and download
// library in $XDG_CONFIG_HOME/kamps/config.toml.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	appName        = "kamps"
	configFile     = "config.toml"
	defaultSiteURL = "https://kulon2.undip.ac.id"
)

// Config is the resolved runtime configuration.
type Config struct {
	// SiteURL is the LMS base URL (empty until configured).
	SiteURL string
	// LibraryDir is where downloaded materials are stored.
	LibraryDir string
	// StateDir holds kamps' own data (decisions, download flags, session).
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
		SiteURL:    defaultSiteURL,
		LibraryDir: filepath.Join(home, "Documents", appName),
		StateDir:   xdg("XDG_STATE_HOME", filepath.Join(home, ".local", "state")),
		ConfigDir:  xdg("XDG_CONFIG_HOME", filepath.Join(home, ".config")),
	}
}

// Load reads the user's config.toml, layered over Default. A missing file is
// not an error: the defaults are a complete configuration.
func Load() (Config, error) {
	return LoadFrom("")
}

// LoadFrom is Load with an explicit path, mainly for tests. An empty path uses
// the normal XDG config location.
func LoadFrom(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		path = cfg.ConfigPath()
	}

	var file fileConfig
	meta, err := toml.DecodeFile(path, &file)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, fmt.Errorf("read %s: %w", path, err)
	}
	if undecoded := meta.Undecoded(); len(undecoded) > 0 {
		return cfg, fmt.Errorf("%s: unknown key %q", path, undecoded[0].String())
	}

	if site := strings.TrimSpace(file.SiteURL); site != "" {
		u, err := url.Parse(site)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return cfg, fmt.Errorf("%s: site_url must be an absolute http or https URL", path)
		}
		cfg.SiteURL = strings.TrimRight(site, "/")
	}
	if dir := strings.TrimSpace(file.LibraryDir); dir != "" {
		expanded, err := expandHome(dir)
		if err != nil {
			return cfg, fmt.Errorf("%s: library_dir: %w", path, err)
		}
		if !filepath.IsAbs(expanded) {
			return cfg, fmt.Errorf("%s: library_dir must be absolute or start with ~", path)
		}
		cfg.LibraryDir = expanded
	}
	return cfg, nil
}

type fileConfig struct {
	SiteURL    string `toml:"site_url"`
	LibraryDir string `toml:"library_dir"`
}

// ConfigPath is the TOML file read by Load.
func (c Config) ConfigPath() string {
	if c.ConfigDir == "" {
		return ""
	}
	return filepath.Join(c.ConfigDir, configFile)
}

// StatePath is the JSON file that keeps decisions and download flags between
// runs.
func (c Config) StatePath() string {
	if c.StateDir == "" {
		return ""
	}
	return filepath.Join(c.StateDir, "state.json")
}

// SessionPath is the private file that keeps the Moodle session cookie.
func (c Config) SessionPath() string {
	if c.StateDir == "" {
		return ""
	}
	return filepath.Join(c.StateDir, "session.json")
}

func expandHome(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return filepath.Clean(path), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if path == "~" {
		return home, nil
	}
	return filepath.Join(home, path[2:]), nil
}

func xdg(env, fallback string) string {
	if v := os.Getenv(env); v != "" {
		return filepath.Join(v, appName)
	}
	return filepath.Join(fallback, appName)
}
