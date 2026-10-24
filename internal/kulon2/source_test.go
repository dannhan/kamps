package kulon2

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/ruyam/kamps/internal/session"
)

func TestLiveSourceLoadReturnsExplicitPartialSnapshot(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/my/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<script>M.cfg = {"sesskey":"fresh-key","userId":42};</script>`))
	})
	mux.HandleFunc("/course/view.php", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<li id="section-0"><h3 class="sectionname">General</h3></li>`))
	})
	mux.HandleFunc("/lib/ajax/service.php", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("sesskey") != "fresh-key" {
			t.Errorf("sesskey query = %q", r.URL.Query().Get("sesskey"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]any{map[string]any{"data": []any{
			map[string]any{"id": 18326, "shortname": "PAIK6304", "fullname": "Metode Numerik A"},
		}}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	path := filepath.Join(t.TempDir(), "session.json")
	if err := session.Save(path, session.Session{SiteURL: srv.URL, Cookie: "fixture-cookie"}); err != nil {
		t.Fatal(err)
	}
	src, err := NewSource(srv.URL, path)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := src.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Courses) != 1 || snapshot.Courses[0].ID != 18326 {
		t.Fatalf("courses = %+v", snapshot.Courses)
	}
	if len(snapshot.Assignments) != 0 || len(snapshot.Materials) != 0 || snapshot.Notice == "" {
		t.Fatalf("partial data not explicit: %+v", snapshot)
	}

	started := time.Now()
	if _, err := src.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed < time.Second {
		t.Fatalf("consecutive source loads were not politely spaced: %v", elapsed)
	}
}

func TestNewSourceRejectsMissingOrWrongSiteSession(t *testing.T) {
	if _, err := NewSource("https://kulon2.undip.ac.id", filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Fatal("expected missing session error")
	}
	path := filepath.Join(t.TempDir(), "session.json")
	if err := session.Save(path, session.Session{SiteURL: "https://other.example", Cookie: "cookie"}); err != nil {
		t.Fatal(err)
	}
	if _, err := NewSource("https://kulon2.undip.ac.id", path); err == nil {
		t.Fatal("expected mismatched site error")
	}
}
