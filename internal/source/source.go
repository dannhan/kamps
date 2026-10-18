// Package source defines where kamps gets its data.
//
// The whole UI is built against this one interface. Today it is satisfied by
// an in-memory Demo source; later a Moodle source (HTTP + parsing) will
// implement exactly the same methods, and the UI will not change at all.
package source

import (
	"context"

	"github.com/ruyam/kamps/internal/model"
)

// Source is a read-only view of a person's courses, assignments and materials.
type Source interface {
	// Label is a short human description, e.g. "demo" or "kulon2.undip.ac.id".
	Label() string

	Courses(ctx context.Context) ([]model.Course, error)
	Assignments(ctx context.Context) ([]model.Assignment, error)
	Materials(ctx context.Context) ([]model.Material, error)
}
