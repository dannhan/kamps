package source

import (
	"context"
	"testing"
)

func TestDemoLoadReturnsCopies(t *testing.T) {
	demo := NewDemo()
	first, err := demo.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	first.Courses[0].Name = "changed"
	first.Assignments[0].Name = "changed"
	first.Materials[0].Name = "changed"

	second, err := demo.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if second.Courses[0].Name == "changed" || second.Assignments[0].Name == "changed" || second.Materials[0].Name == "changed" {
		t.Fatal("Load exposed the demo's internal slices")
	}
}
