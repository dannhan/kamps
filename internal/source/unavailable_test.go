package source

import (
	"context"
	"testing"

	"github.com/ruyam/kamps/internal/model"
)

func TestUnavailableDataSourceLabelsPartialSnapshot(t *testing.T) {
	src := &UnavailableDataSource{
		LabelText: "Kulon2",
		Courses:   []model.Course{{ID: 7, Name: "Test"}},
	}
	snapshot, err := src.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Courses) != 1 || len(snapshot.Assignments) != 0 || len(snapshot.Materials) != 0 || snapshot.Notice == "" {
		t.Fatalf("snapshot does not make partial data explicit: %+v", snapshot)
	}
	snapshot.Courses[0].Name = "mutated"
	again, err := src.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if again.Courses[0].Name != "Test" {
		t.Fatal("source exposed its course slice")
	}
}
