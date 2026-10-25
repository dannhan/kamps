// Package export renders the Commit queue for Markdown, todo.txt, or
// Taskwarrior. Rendering is pure; Write performs an idempotent, locked append.
package export

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ruyam/kamps/internal/lockfile"
	"github.com/ruyam/kamps/internal/model"
)

// Item is one queued decision.
type Item struct {
	ID     string
	Course string
	Name   string
	Due    time.Time
	HasDue bool
	Action model.Triage
}

// Result reports what a write did.
type Result struct {
	Added   int
	Skipped int
	Path    string
}

// Writer renders and writes one export format.
type Writer interface {
	Format() string
	Render(items []Item, existing []byte) (string, Result)
	Write(path string, items []Item) (Result, error)
}

// Previewer is implemented by writers that need their own existing-target
// lookup (Taskwarrior queries the task CLI).
type Previewer interface {
	Preview(items []Item) (string, Result)
}

// For returns the writer for a configured format.
func For(format string) (Writer, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "markdown", "md":
		return Markdown{}, nil
	case "todo", "todo.txt":
		return Todo{}, nil
	case "taskwarrior", "task":
		return Taskwarrior{}, nil
	default:
		return nil, fmt.Errorf("unknown export format %q", format)
	}
}

func dueSuffix(item Item) string {
	if !item.HasDue || item.Due.IsZero() {
		return ""
	}
	return " due:" + item.Due.Format("2006-01-02")
}

func readOptional(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return data
}

func withTrailingNewline(text string) string {
	if text == "" {
		return ""
	}
	if strings.HasSuffix(text, "\n") {
		return text
	}
	return text + "\n"
}

// writeWithLock renders against the locked file contents and writes atomically.
func writeWithLock(path string, render func([]byte) (string, Result)) (Result, error) {
	if path == "" {
		return Result{}, fmt.Errorf("no export path configured")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Result{}, err
	}
	lock, err := lockfile.Acquire(path + ".lock")
	if err != nil {
		return Result{}, err
	}
	defer lock.Unlock()

	existing := readOptional(path)
	newText, result := render(existing)
	result.Path = path
	if newText == "" {
		return result, nil
	}
	merged := withTrailingNewline(string(existing))
	if merged != "" && newText != "" {
		merged += "\n"
	}
	merged += withTrailingNewline(newText)

	info, statErr := os.Stat(path)
	mode := os.FileMode(0o644)
	if statErr == nil {
		mode = info.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".export-*.tmp")
	if err != nil {
		return result, err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.WriteString(merged); err != nil {
		tmp.Close()
		return result, err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return result, err
	}
	if err := tmp.Close(); err != nil {
		return result, err
	}
	if err := os.Chmod(name, mode); err != nil {
		return result, err
	}
	if err := os.Rename(name, path); err != nil {
		return result, err
	}
	return result, nil
}
