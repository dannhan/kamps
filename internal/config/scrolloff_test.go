package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScrolloffConfig(t *testing.T) {
	for _, tc := range []struct {
		value   string
		want    int
		invalid bool
	}{{"", 5, false}, {"scrolloff = 0", 0, false}, {"scrolloff = 8", 8, false}, {"scrolloff = -1", 0, true}} {
		path := filepath.Join(t.TempDir(), "config.toml")
		if err := os.WriteFile(path, []byte(tc.value), 0600); err != nil {
			t.Fatal(err)
		}
		cfg, err := LoadFrom(path)
		if tc.invalid {
			if err == nil {
				t.Fatalf("%q accepted", tc.value)
			}
			continue
		}
		if err != nil || cfg.Scrolloff != tc.want {
			t.Fatalf("%q: got %d, err %v", tc.value, cfg.Scrolloff, err)
		}
	}
}
