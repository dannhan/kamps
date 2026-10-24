package tui

import (
	"context"
	"errors"
	"testing"

	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/source"
)

type stubSource struct {
	snapshot source.Snapshot
	err      error
}

func (s stubSource) Label() string { return "stub" }

func (s stubSource) Load(context.Context) (source.Snapshot, error) {
	return s.snapshot, s.err
}

func TestLoadCmdReturnsSnapshot(t *testing.T) {
	want := source.Snapshot{
		Courses: []model.Course{{ID: 7, Name: "Test Course"}},
	}
	msg := loadCmd(stubSource{snapshot: want})()
	loaded, ok := msg.(loadedMsg)
	if !ok {
		t.Fatalf("loadCmd returned %T, want loadedMsg", msg)
	}
	if len(loaded.Courses) != 1 || loaded.Courses[0].ID != 7 {
		t.Fatalf("loadCmd lost the snapshot: %+v", loaded.Snapshot)
	}
}

func TestLoadCmdReturnsError(t *testing.T) {
	want := errors.New("boom")
	msg := loadCmd(stubSource{err: want})()
	failure, ok := msg.(errMsg)
	if !ok {
		t.Fatalf("loadCmd returned %T, want errMsg", msg)
	}
	if !errors.Is(failure.err, want) {
		t.Fatalf("loadCmd changed the error: %v", failure.err)
	}
}
