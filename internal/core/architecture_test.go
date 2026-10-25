package core

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCoreHasNoUIImports(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "list", "-f", `{{join .Imports "\n"}}`, "./internal/core")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, imported := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if imported == "" {
			continue
		}
		if strings.HasPrefix(imported, "github.com/ruyam/kamps/internal/tui") ||
			strings.HasPrefix(imported, "github.com/ruyam/kamps/internal/plain") ||
			strings.HasPrefix(imported, "github.com/ruyam/kamps/cmd/kamps") ||
			strings.Contains(imported, "charmbracelet") {
			t.Errorf("core imports UI package %q", imported)
		}
	}
}
