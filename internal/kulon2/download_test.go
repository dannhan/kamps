package kulon2

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ruyam/kamps/internal/library"
	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/session"
)

func TestLiveSourceDownloadWritesFile(t *testing.T) {
	body := []byte("course material")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Cookie"), "MoodleSession=cookie") {
			t.Errorf("download request missing session cookie: %q", r.Header.Get("Cookie"))
		}
		if r.URL.Path != "/pluginfile.php/1/mod_resource/content/0/slides.pdf" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Length", fmt.Sprint(len(body)))
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	src := newTestSource(t, srv.URL, session.Session{Cookie: "cookie", SessKey: "saved-key"})
	dest := filepath.Join(t.TempDir(), "metnum", "week-1", "slides.pdf")
	material := model.Material{
		ID:   "M165401",
		Kind: "file",
		URL:  srv.URL + "/pluginfile.php/1/mod_resource/content/0/slides.pdf",
		Size: int64(len(body)),
	}

	got, err := src.Download(context.Background(), material, dest, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != dest {
		t.Fatalf("path = %q, want %q", got, dest)
	}
	saved, err := os.ReadFile(dest)
	if err != nil || string(saved) != string(body) {
		t.Fatalf("saved = %q, err = %v", saved, err)
	}
}

func TestLiveSourceDownloadRejectsExternalLink(t *testing.T) {
	src := newTestSource(t, "https://lms.test", session.Session{Cookie: "cookie"})
	_, err := src.Download(context.Background(), model.Material{ID: "M1", Kind: "url", URL: "https://example.test"}, filepath.Join(t.TempDir(), "f"), nil)
	if err == nil || !strings.Contains(err.Error(), "external link") {
		t.Fatalf("err = %v", err)
	}
}

func TestLiveSourceDownloadReportsExpiredSession(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>login</html>"))
	}))
	defer srv.Close()

	src := newTestSource(t, srv.URL, session.Session{Cookie: "cookie"})
	_, err := src.Download(context.Background(), model.Material{ID: "M1", Kind: "file", URL: srv.URL + "/pluginfile.php/1/x.pdf"}, filepath.Join(t.TempDir(), "x.pdf"), nil)
	if err == nil || !strings.Contains(err.Error(), "kamps login") {
		t.Fatalf("err = %v", err)
	}
}

func TestLiveSourceDownloadResolvesResourcePage(t *testing.T) {
	body := []byte("slides body")
	mux := http.NewServeMux()
	mux.HandleFunc("/mod/resource/view.php", func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Cookie"), "MoodleSession=cookie") {
			t.Errorf("resource page request missing session cookie: %q", r.Header.Get("Cookie"))
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, `<html><body><a href="/pluginfile.php/1/mod_resource/content/0/slides.pdf?forcedownload=1">Download</a></body></html>`)
	})
	mux.HandleFunc("/pluginfile.php/1/mod_resource/content/0/slides.pdf", func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Cookie"), "MoodleSession=cookie") {
			t.Errorf("file request missing session cookie: %q", r.Header.Get("Cookie"))
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Length", fmt.Sprint(len(body)))
		_, _ = w.Write(body)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	src := newTestSource(t, srv.URL, session.Session{Cookie: "cookie"})
	dest := filepath.Join(t.TempDir(), "metnum", "week-1", "slides.pdf")
	material := model.Material{ID: "M165401", Kind: "file", URL: srv.URL + "/mod/resource/view.php?id=165401"}

	got, err := src.Download(context.Background(), material, dest, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != dest {
		t.Fatalf("path = %q, want %q", got, dest)
	}
	saved, err := os.ReadFile(dest)
	if err != nil || string(saved) != string(body) {
		t.Fatalf("saved = %q, err = %v", saved, err)
	}
}

func TestLiveSourceDownloadRejectsFolderActivity(t *testing.T) {
	src := newTestSource(t, "https://lms.test", session.Session{Cookie: "cookie"})
	_, err := src.Download(context.Background(), model.Material{ID: "M1", Kind: "file", URL: "https://lms.test/mod/folder/view.php?id=2"}, filepath.Join(t.TempDir(), "f"), nil)
	if err == nil || !strings.Contains(err.Error(), "folder") {
		t.Fatalf("err = %v", err)
	}
}

func TestRetryableDownloadError(t *testing.T) {
	if !retryableDownloadError(library.ErrRateLimited) || !retryableDownloadError(library.ErrIncompleteDownload) {
		t.Fatal("transient download errors should be retried")
	}
	if retryableDownloadError(library.ErrFileNotFound) {
		t.Fatal("404 should not be retried")
	}
}
