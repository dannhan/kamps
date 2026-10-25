// Package lockfile provides a small cross-platform advisory lock used to
// serialize state writes and destination reservations.
package lockfile

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
)

// Lock is a held advisory lock.
type Lock struct {
	file *flock.Flock
}

// Acquire takes an exclusive lock, creating parent directories as needed.
func Acquire(path string) (*Lock, error) {
	if path == "" {
		return nil, fmt.Errorf("empty lock path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create lock directory: %w", err)
	}
	file := flock.New(path)
	if err := file.Lock(); err != nil {
		return nil, fmt.Errorf("lock %s: %w", path, err)
	}
	return &Lock{file: file}, nil
}

// Unlock releases the lock. It is safe to call more than once.
func (l *Lock) Unlock() {
	if l == nil || l.file == nil {
		return
	}
	_ = l.file.Unlock()
	l.file = nil
}
