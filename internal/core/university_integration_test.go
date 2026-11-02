package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/library"
	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/workspace"
)

func TestUniversityCourseMaterialDestination(t *testing.T) {
	root := t.TempDir()
	courseDir := filepath.Join(root, "basdat")
	weekDir := filepath.Join(courseDir, "materi", "week-05")
	if err := os.MkdirAll(weekDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(courseDir, "course.json"), []byte(`{"code":"MIK1624303","lms_course_id":17964,"source":"lms"}`), 0600); err != nil {
		t.Fatal(err)
	}
	courses, notices := workspace.Resolve([]model.Course{{ID: 17964, Code: "MIK1624303"}}, []string{root})
	if len(notices) != 0 {
		t.Fatalf("notices: %v", notices)
	}
	got := Destination(config.Config{LibraryDir: root}, courses, model.Material{CourseID: 17964, Section: "week-5", Folder: "Slides", Name: "intro.pdf"})
	want := filepath.Join(weekDir, "Slides", "intro.pdf")
	if got != want {
		t.Fatalf("destination = %q, want %q", got, want)
	}
	got = Destination(config.Config{LibraryDir: root}, courses, model.Material{CourseID: 17964, Section: "week-6", Folder: "praktikum", Name: "work.pdf"})
	want = filepath.Join(courseDir, "materi", "week-06", "praktikum", "work.pdf")
	if got != want {
		t.Fatalf("new week destination = %q, want %q", got, want)
	}
}

func TestUniversityDownloadPreservesExistingCoursework(t *testing.T) {
	root := t.TempDir()
	course := filepath.Join(root, "metnum")
	week := filepath.Join(course, "materi", "week-03")
	if err := os.MkdirAll(week, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(course, "course.json"), []byte(`{"lms_course_id":18326,"code":"PAIK6304","source":"lms"}`), 0600); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(week, "notes.pdf")
	if err := os.WriteFile(old, []byte("my coursework"), 0600); err != nil {
		t.Fatal(err)
	}
	courses, _ := workspace.Resolve([]model.Course{{ID: 18326}}, []string{root})
	dest := Destination(config.Config{LibraryDir: root}, courses, model.Material{CourseID: 18326, Section: "week-3", Name: "notes.pdf"})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("downloaded material")) }))
	defer server.Close()
	path, err := library.DownloadFile(context.Background(), server.Client(), server.URL, dest, int64(len("downloaded material")), nil)
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(week, "notes_v2.pdf") {
		t.Fatalf("downloaded to %q", path)
	}
	oldContent, err := os.ReadFile(old)
	if err != nil || string(oldContent) != "my coursework" {
		t.Fatalf("original = %q, %v", oldContent, err)
	}
	newContent, err := os.ReadFile(path)
	if err != nil || string(newContent) != "downloaded material" {
		t.Fatalf("download = %q, %v", newContent, err)
	}
}
