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
	// StateDir holds kamps' own data (decisions, download flags, session, cache).
	StateDir string
	// ConfigDir holds config.toml.
	ConfigDir string
	// Source is the in-tree source name selected at startup.
	Source string
	// WorkspaceRoots are folders scanned for course.json mapping files.
	WorkspaceRoots []string
	// CommitFile is the default export target for the Commit tab.
	CommitFile string
	// ExportFormat is markdown, todo, or taskwarrior.
	ExportFormat string
	// AutoTrackSubmitted derives Tracked for assignments the LMS marks submitted.
	AutoTrackSubmitted bool
	// NoAnim disables cheerful motion.
	NoAnim bool
}

// Default returns the XDG-ish default configuration.
func Default() Config {
	home, _ := os.UserHomeDir()
	return Config{
		SiteURL:            defaultSiteURL,
		LibraryDir:         filepath.Join(home, "Documents", appName),
		StateDir:           xdg("XDG_STATE_HOME", filepath.Join(home, ".local", "state")),
		ConfigDir:          xdg("XDG_CONFIG_HOME", filepath.Join(home, ".config")),
		Source:             "kulon2",
		ExportFormat:       "markdown",
		AutoTrackSubmitted: true,
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
	if source := strings.TrimSpace(file.Source); source != "" {
		cfg.Source = source
	}
	for _, root := range file.WorkspaceRoots {
		expanded, err := expandHome(strings.TrimSpace(root))
		if err != nil {
			return cfg, fmt.Errorf("%s: workspace_roots: %w", path, err)
		}
		if !filepath.IsAbs(expanded) {
			return cfg, fmt.Errorf("%s: workspace_roots must be absolute or start with ~", path)
		}
		cfg.WorkspaceRoots = append(cfg.WorkspaceRoots, expanded)
	}
	if target := strings.TrimSpace(file.CommitFile); target != "" {
		expanded, err := expandHome(target)
		if err != nil {
			return cfg, fmt.Errorf("%s: commit_file: %w", path, err)
		}
		if !filepath.IsAbs(expanded) {
			return cfg, fmt.Errorf("%s: commit_file must be absolute or start with ~", path)
		}
		cfg.CommitFile = expanded
	}
	if format := strings.ToLower(strings.TrimSpace(file.ExportFormat)); format != "" {
		switch format {
		case "markdown", "todo", "taskwarrior":
			cfg.ExportFormat = format
		default:
			return cfg, fmt.Errorf("%s: export_format must be markdown, todo, or taskwarrior", path)
		}
	}
	if file.AutoTrackSubmitted != nil {
		cfg.AutoTrackSubmitted = *file.AutoTrackSubmitted
	}
	return cfg, nil
}

type fileConfig struct {
	SiteURL            string   `toml:"site_url"`
	LibraryDir         string   `toml:"library_dir"`
	Source             string   `toml:"source"`
	WorkspaceRoots     []string `toml:"workspace_roots"`
	CommitFile         string   `toml:"commit_file"`
	ExportFormat       string   `toml:"export_format"`
	AutoTrackSubmitted *bool    `toml:"auto_track_submitted"`
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

// LockPath is the advisory lock that serializes state.json updates.
func (c Config) LockPath() string {
	if c.StateDir == "" {
		return ""
	}
	return filepath.Join(c.StateDir, "state.lock")
}

// CachePath is the last successful snapshot used for offline mode.
func (c Config) CachePath() string {
	if c.StateDir == "" {
		return ""
	}
	return filepath.Join(c.StateDir, "cache.json")
}

// EnsureLibraryDir creates and validates the material library root.
func (c Config) EnsureLibraryDir() error {
	if c.LibraryDir == "" {
		return errors.New("library_dir is not configured")
	}
	if err := os.MkdirAll(c.LibraryDir, 0o755); err != nil {
		return fmt.Errorf("create library_dir: %w", err)
	}
	info, err := os.Stat(c.LibraryDir)
	if err != nil {
		return fmt.Errorf("inspect library_dir: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("library_dir is not a directory: %s", c.LibraryDir)
	}
	probe, err := os.CreateTemp(c.LibraryDir, ".kamps-write-*")
	if err != nil {
		return fmt.Errorf("library_dir is not writable: %w", err)
	}
	name := probe.Name()
	_ = probe.Close()
	_ = os.Remove(name)
	return nil
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
