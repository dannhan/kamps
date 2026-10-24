// Package source defines where kamps gets its data.
//
// The whole UI is built against this one interface. Today it is satisfied by
// an in-memory Demo source; later a Moodle source (HTTP + parsing) will
// implement exactly the same Load method, and the UI will not change at all.
package source

import (
	"context"

	"github.com/ruyam/kamps/internal/model"
)

// Snapshot is one coherent view of a person's coursework.
type Snapshot struct {
	Courses     []model.Course
	Assignments []model.Assignment
	Materials   []model.Material
}

// Source is a read-only view of a person's courses, assignments and materials.
type Source interface {
	// Label is a short human description, e.g. "demo" or "kulon2.undip.ac.id".
	Label() string

	// Load returns all coursework in one pass. A network source can fetch
	// shared pages once, and the caller gets one consistent error path.
	Load(ctx context.Context) (Snapshot, error)
}
