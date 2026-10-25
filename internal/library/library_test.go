package library

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"../../etc/passwd", "passwd"},
		{"week:1/slides?.pdf", "slides_.pdf"},
		{"", "unnamed"},
		{"..", "unnamed"},
	} {
		if got := SanitizeFilename(tc.in); got != tc.want {
			t.Errorf("SanitizeFilename(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestSanitizeFilenameLength(t *testing.T) {
	got := SanitizeFilename(strings.Repeat("a", 300) + ".pdf")
	if len(got) > 200 || filepath.Ext(got) != ".pdf" {
		t.Fatalf("sanitized length=%d extension=%q", len(got), filepath.Ext(got))
	}
}

func TestSectionPathSanitizesEachComponent(t *testing.T) {
	path := SectionPath("/library", "../Course", "Week/1", "../../slides.pdf")
	if path != filepath.Join("/library", "Course", "1", "slides.pdf") {
		t.Fatalf("path = %q", path)
	}
}

func TestResolveDestinationVersionDoesNotClobber(t *testing.T) {
	dir := t.TempDir()
	original := filepath.Join(dir, "slides.pdf")
	if err := os.WriteFile(original, []byte("student notes"), 0600); err != nil {
		t.Fatal(err)
	}
	resolved, action := ResolveDestination(original, 100)
	if action != ActionVersion || resolved != filepath.Join(dir, "slides_v2.pdf") {
		t.Fatalf("resolved=%q action=%v", resolved, action)
	}
	if got, _ := os.ReadFile(original); string(got) != "student notes" {
		t.Fatalf("original overwritten: %q", got)
	}
}

func TestResolveDestinationSkipsSameSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.pdf")
	if err := os.WriteFile(path, []byte("1234"), 0600); err != nil {
		t.Fatal(err)
	}
	got, action := ResolveDestination(path, 4)
	if got != path || action != ActionSkip {
		t.Fatalf("got=%q action=%v", got, action)
	}
}

func TestDownloadFileSuccessAndNoClobber(t *testing.T) {
	body := []byte("download content")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Length", fmt.Sprint(len(body)))
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	target := filepath.Join(t.TempDir(), "course", "week", "slides.pdf")
	got, err := DownloadFile(context.Background(), srv.Client(), srv.URL, target, int64(len(body)), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != target {
		t.Fatalf("path=%q want %q", got, target)
	}
	contents, err := os.ReadFile(target)
	if err != nil || string(contents) != string(body) {
		t.Fatalf("contents=%q err=%v", contents, err)
	}
}

func TestDownloadFileRejectsHTMLAndCleansPart(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, "<html>login</html>")
	}))
	defer srv.Close()
	dir := t.TempDir()
	_, err := DownloadFile(context.Background(), srv.Client(), srv.URL, filepath.Join(dir, "file.pdf"), 0, nil)
	if !errors.Is(err, ErrSessionExpiredHTML) {
		t.Fatalf("err=%v", err)
	}
	assertNoParts(t, dir)
}

func TestDownloadFileNotFoundCleansPart(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()
	dir := t.TempDir()
	_, err := DownloadFile(context.Background(), srv.Client(), srv.URL, filepath.Join(dir, "file.pdf"), 0, nil)
	if !errors.Is(err, ErrFileNotFound) {
		t.Fatalf("err=%v", err)
	}
	assertNoParts(t, dir)
}

func TestDownloadFileEmptyAndTruncated(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		size int
		want error
	}{{"empty", "", 0, ErrEmptyFile}, {"truncated", "abc", 10, ErrIncompleteDownload}} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/octet-stream")
				if tc.size > 0 {
					w.Header().Set("Content-Length", fmt.Sprint(tc.size))
				}
				_, _ = io.WriteString(w, tc.body)
			}))
			defer srv.Close()
			dir := t.TempDir()
			_, err := DownloadFile(context.Background(), srv.Client(), srv.URL, filepath.Join(dir, "file.bin"), int64(tc.size), nil)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err=%v want %v", err, tc.want)
			}
			assertNoParts(t, dir)
		})
	}
}

func TestDownloadFileCancellationCleansPart(t *testing.T) {
	started := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		close(started)
		<-r.Context().Done()
	}))
	defer srv.Close()
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	var err error
	go func() {
		defer wg.Done()
		_, err = DownloadFile(ctx, srv.Client(), srv.URL, filepath.Join(dir, "file.bin"), 0, nil)
	}()
	<-started
	cancel()
	wg.Wait()
	if err == nil {
		t.Fatal("expected cancellation error")
	}
	assertNoParts(t, dir)
}

func TestDownloadFileUsesContentDispositionName(t *testing.T) {
	body := []byte("pdf bytes")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", `attachment; filename="Week 1 Slides.pdf"`)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	dir := t.TempDir()
	got, err := DownloadFile(context.Background(), srv.Client(), srv.URL, filepath.Join(dir, "Slides"), int64(len(body)), nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "Week 1 Slides.pdf"); got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
}

func TestCoordinatorDeduplicates(t *testing.T) {
	c := NewCoordinator()
	if !c.TryStart("M1") || c.TryStart("M1") {
		t.Fatal("expected second start to be rejected")
	}
	c.Finish("M1")
	if !c.TryStart("M1") {
		t.Fatal("expected start after finish to succeed")
	}
}

func assertNoParts(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".part") {
			t.Fatalf("temporary file left behind: %s", entry.Name())
		}
	}
}
