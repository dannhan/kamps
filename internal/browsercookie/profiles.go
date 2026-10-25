package browsercookie

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

func home() string {
	dir, _ := os.UserHomeDir()
	return dir
}

func firefoxRoots() []string {
	roots := []string{
		filepath.Join(home(), ".mozilla", "firefox"),
		filepath.Join(home(), "snap", "firefox", "common", ".mozilla", "firefox"),
		filepath.Join(home(), ".var", "app", "org.mozilla.firefox", ".mozilla", "firefox"),
	}
	return unique(roots)
}

func chromiumRoots() []struct {
	name string
	root string
} {
	names := []string{"google-chrome", "chromium", "brave-browser", "microsoft-edge"}
	var roots []struct {
		name string
		root string
	}
	for _, name := range names {
		roots = append(roots, struct {
			name string
			root string
		}{name, filepath.Join(home(), ".config", name)})
		roots = append(roots, struct {
			name string
			root string
		}{name, filepath.Join(home(), "snap", name, "common", ".config", name)})
	}
	return roots
}

func unique(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func firefoxCookies(host string) []Cookie {
	var out []Cookie
	for _, root := range firefoxRoots() {
		_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if entry.IsDir() {
				if entry.Name() == "storage" || entry.Name() == "cache" {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.Name() != "cookies.sqlite" {
				return nil
			}
			cookies, _ := readFirefox(path, host)
			out = append(out, cookies...)
			return nil
		})
	}
	return out
}

func readFirefox(path, host string) ([]Cookie, error) {
	db, cleanup, err := openCopy(path)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	rows, err := db.Query(`SELECT host, value, expiry FROM moz_cookies WHERE name = 'MoodleSession'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Cookie
	for rows.Next() {
		var cookieHost, value string
		var expiry int64
		if err := rows.Scan(&cookieHost, &value, &expiry); err != nil {
			continue
		}
		if value == "" || !hostMatches(cookieHost, host) {
			continue
		}
		out = append(out, Cookie{
			Value:     value,
			ExpiresAt: time.Unix(expiry, 0),
			Browser:   "firefox",
			Profile:   path,
		})
	}
	return out, nil
}

func chromiumCookies(host string) []Cookie {
	var out []Cookie
	for _, candidate := range chromiumRoots() {
		entries, err := os.ReadDir(candidate.root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			if name != "Default" && !strings.HasPrefix(name, "Profile ") {
				continue
			}
			path := filepath.Join(candidate.root, name, "Cookies")
			cookies, _ := readChromium(path, host, candidate.name)
			out = append(out, cookies...)
		}
	}
	return out
}

func readChromium(path, host, browser string) ([]Cookie, error) {
	db, cleanup, err := openCopy(path)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	rows, err := db.Query(`SELECT host_key, value, encrypted_value, expires_utc FROM cookies WHERE name = 'MoodleSession'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Cookie
	for rows.Next() {
		var cookieHost, value string
		var encrypted []byte
		var expires int64
		if err := rows.Scan(&cookieHost, &value, &encrypted, &expires); err != nil {
			continue
		}
		if !hostMatches(cookieHost, host) {
			continue
		}
		if value == "" && len(encrypted) > 0 {
			value, _ = decryptChromium(encrypted, browser)
		}
		if value == "" {
			continue
		}
		out = append(out, Cookie{
			Value:     value,
			ExpiresAt: chromeExpiry(expires),
			Browser:   browser,
			Profile:   path,
		})
	}
	return out, nil
}

// openCopy copies a possibly locked SQLite database (and its WAL) to a temp
// directory and opens it read-only.
func openCopy(path string) (*sql.DB, func(), error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, func() {}, err
	}
	dir, err := os.MkdirTemp("", "kamps-cookies-*")
	if err != nil {
		return nil, func() {}, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	target := filepath.Join(dir, "cookies.sqlite")
	if err := os.WriteFile(target, data, 0o600); err != nil {
		cleanup()
		return nil, func() {}, err
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		if extra, readErr := os.ReadFile(path + suffix); readErr == nil {
			_ = os.WriteFile(target+suffix, extra, 0o600)
		}
	}
	db, err := sql.Open("sqlite", "file:"+target+"?mode=ro&immutable=1")
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		cleanup()
		return nil, func() {}, err
	}
	return db, func() { _ = db.Close(); cleanup() }, nil
}

var errNoKey = errors.New("no browser cookie key")
