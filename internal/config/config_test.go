package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFromMissingFileUsesDefaults(t *testing.T) {
	cfg, err := LoadFrom(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LibraryDir == "" {
		t.Fatal("default library dir is empty")
	}
	if cfg.SiteURL != defaultSiteURL {
		t.Fatalf("site URL = %q, want %q", cfg.SiteURL, defaultSiteURL)
	}
}

func TestLoadFromOverrides(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(t.TempDir(), "config.toml")
	body := `
site_url = "https://kulon2.undip.ac.id/"
library_dir = "~/projects/university"
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SiteURL != "https://kulon2.undip.ac.id" {
		t.Fatalf("site URL = %q", cfg.SiteURL)
	}
	want := filepath.Join(home, "projects", "university")
	if cfg.LibraryDir != want {
		t.Fatalf("library dir = %q, want %q", cfg.LibraryDir, want)
	}
}

func TestNormalizeTheme(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
		ok   bool
	}{
		{"", ThemeAuto, true},
		{"AUTO", ThemeAuto, true},
		{"dark", ThemeDark, true},
		{" Light ", ThemeLight, true},
		{"mono", ThemeMono, true},
		{"solarized", "", false},
	} {
		got, ok := NormalizeTheme(tc.in)
		if got != tc.want || ok != tc.ok {
			t.Errorf("NormalizeTheme(%q) = %q, %v; want %q, %v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestLoadFromTheme(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(`theme = "light"`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Theme != ThemeLight {
		t.Fatalf("theme = %q, want light", cfg.Theme)
	}

	if err := os.WriteFile(path, []byte(`theme = "neon"`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFrom(path); err == nil || !strings.Contains(err.Error(), "theme must be") {
		t.Fatalf("invalid theme error = %v", err)
	}
}

func TestLoadFromRejectsUnknownKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(`library_dri = "~/x"`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadFrom(path)
	if err == nil || !strings.Contains(err.Error(), "unknown key") {
		t.Fatalf("error = %v, want unknown key", err)
	}
}

func TestLoadFromRejectsRelativeLibraryDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(`library_dir = "materials"`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadFrom(path)
	if err == nil || !strings.Contains(err.Error(), "absolute") {
		t.Fatalf("error = %v, want absolute path error", err)
	}
}

func TestPaths(t *testing.T) {
	cfg := Config{ConfigDir: "/config", StateDir: "/state"}
	if got := cfg.ConfigPath(); got != "/config/config.toml" {
		t.Fatalf("ConfigPath() = %q", got)
	}
	if got := cfg.StatePath(); got != "/state/state.json" {
		t.Fatalf("StatePath() = %q", got)
	}
	if got := cfg.SessionPath(); got != "/state/session.json" {
		t.Fatalf("SessionPath() = %q", got)
	}
}
