package source

import (
	"context"
	"errors"

	"github.com/ruyam/kamps/internal/model"
)

// ErrIncompleteSnapshot signals that a source can identify some records but
// cannot yet provide all data required by the UI.
var ErrIncompleteSnapshot = errors.New("source cannot provide a complete coursework snapshot")

// UnavailableDataSource carries a verified course list while making it
// explicit that assignments and materials have not been loaded. It is useful
// for an early live-source milestone without presenting an empty UI as complete.
type UnavailableDataSource struct {
	LabelText string
	Courses   []model.Course
}

func (s *UnavailableDataSource) Label() string { return s.LabelText }

func (s *UnavailableDataSource) Load(context.Context) (Snapshot, error) {
	return Snapshot{
		Courses: append([]model.Course(nil), s.Courses...),
		Notice:  ErrIncompleteSnapshot.Error(),
	}, nil
}
