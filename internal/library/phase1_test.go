package library

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ruyam/kamps/internal/model"
)

func TestDownloadFileResumesPartial(t *testing.T) {
	body := []byte("hello world")
	mux := http.NewServeMux()
	mux.HandleFunc("/file.bin", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Range"); got != "bytes=6-" {
			t.Errorf("Range = %q, want bytes=6-", got)
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes 6-%d/%d", len(body)-1, len(body)))
		w.Header().Set("Content-Length", fmt.Sprint(len(body)-6))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(body[6:])
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "file.bin")
	if err := os.WriteFile(dest+".part", body[:6], 0o644); err != nil {
		t.Fatal(err)
	}
	writePartMeta(dest+".part.json", partMeta{
		URL:       srv.URL + "/file.bin",
		Total:     int64(len(body)),
		Target:    dest,
		UpdatedAt: time.Now(),
	})

	result, err := DownloadFileWithOptions(context.Background(), srv.Client(), srv.URL+"/file.bin", dest, DownloadOptions{Resume: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Path != dest || !result.Resumed {
		t.Fatalf("result = %+v", result)
	}
	saved, err := os.ReadFile(dest)
	if err != nil || string(saved) != string(body) {
		t.Fatalf("saved = %q, err = %v", saved, err)
	}
	if _, err := os.Stat(dest + ".part"); !os.IsNotExist(err) {
		t.Fatalf("part file still exists: %v", err)
	}
}

func TestMaterialPathIncludesFolderAndStaysInside(t *testing.T) {
	base := t.TempDir()
	got, err := MaterialPath(base, "metnum", "Week 1", "Praktikum", "slides.pdf")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(base, "metnum", "Week 1", "Praktikum", "slides.pdf")
	if got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
	if _, err := MaterialPath(base, "../escape", "week", "", "file.pdf"); err != nil {
		t.Fatalf("sanitized traversal should stay inside: %v", err)
	}
}

func TestScanReconcilesFilesAndUnknowns(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "metnum", "week-1", "slides.pdf")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "stray.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	materials := []model.Material{
		{ID: "M1", Name: "slides.pdf", Downloaded: true},
		{ID: "M2", Name: "missing.pdf", Downloaded: true},
	}
	result := Scan(materials, func(m model.Material) string {
		return filepath.Join(dir, "metnum", "week-1", m.Name)
	}, []string{dir})
	if _, ok := result.Downloaded["M1"]; !ok {
		t.Fatal("existing file was not reconciled")
	}
	if len(result.Missing) != 1 || result.Missing[0] != "M2" {
		t.Fatalf("missing = %v", result.Missing)
	}
	if len(result.Unknown) != 1 {
		t.Fatalf("unknown = %v", result.Unknown)
	}
}

func TestNoSpaceDetection(t *testing.T) {
	if !isNoSpace(errors.New("write part file: no space left on device")) {
		t.Fatal("ENOSPC was not recognized")
	}
	if isNoSpace(errors.New("permission denied")) {
		t.Fatal("unrelated error was treated as ENOSPC")
	}
}
