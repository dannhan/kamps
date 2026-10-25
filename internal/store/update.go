package store

import (
	"fmt"

	"github.com/ruyam/kamps/internal/lockfile"
)

// Update runs fn against a freshly loaded state while holding the state lock,
// then saves the result. This preserves decisions written by another kamps
// process between our last load and this write.
func Update(statePath, lockPath string, fn func(*State)) error {
	if statePath == "" {
		return fmt.Errorf("empty state path")
	}
	lock, err := lockfile.Acquire(lockPath)
	if err != nil {
		return err
	}
	defer lock.Unlock()

	state, err := Load(statePath)
	if err != nil {
		return err
	}
	fn(&state)
	return state.Save(statePath)
}
