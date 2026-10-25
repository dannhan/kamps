package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ruyam/kamps/internal/model"
)

func TestResolveMapsByIDAndShortname(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "basdat")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"version":1,"courses":[{"lms_id":17964,"shortname":"MIK1624303","path":"."}]}`
	if err := os.WriteFile(filepath.Join(repo, "course.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	courses := []model.Course{
		{ID: 17964, Short: "basdat", Code: "MIK1624303"},
		{ID: 1, Short: "other"},
	}
	resolved, notices := Resolve(courses, []string{root})
	if len(notices) != 0 {
		t.Fatalf("notices = %v", notices)
	}
	if resolved[0].WorkspacePath != repo {
		t.Fatalf("path = %q, want %q", resolved[0].WorkspacePath, repo)
	}
	if resolved[1].WorkspacePath != "" {
		t.Fatalf("unmapped course got %q", resolved[1].WorkspacePath)
	}
}

func TestResolveReportsMissingPath(t *testing.T) {
	root := t.TempDir()
	body := `{"version":1,"courses":[{"lms_id":1,"path":"./missing"}]}`
	if err := os.WriteFile(filepath.Join(root, "course.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	_, notices := Resolve([]model.Course{{ID: 1}}, []string{root})
	if len(notices) == 0 {
		t.Fatal("missing mapped path should produce a notice")
	}
}

func TestValidateDownloadPath(t *testing.T) {
	base := t.TempDir()
	if err := ValidateDownloadPath(base, filepath.Join(base, "week", "file.pdf")); err != nil {
		t.Fatal(err)
	}
	if err := ValidateDownloadPath(base, filepath.Join(base, "..", "escape.pdf")); err == nil {
		t.Fatal("expected traversal rejection")
	}
}
