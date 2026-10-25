package kulon2

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ruyam/kamps/internal/session"
)

func courseListHandler(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`[{"data":[{"id":18326,"shortname":"PAIK6304","fullname":"Metode Numerik A"}]}]`))
}

func newTestSource(t *testing.T, site string, saved session.Session) *LiveSource {
	t.Helper()
	path := filepath.Join(t.TempDir(), "session.json")
	saved.SiteURL = site
	if err := session.Save(path, saved); err != nil {
		t.Fatal(err)
	}
	src, err := NewSource(site, path, "")
	if err != nil {
		t.Fatal(err)
	}
	src.minRequestGap = 0
	return src
}

func TestLiveSourceUsesSavedSesskeyWithoutSessionCheck(t *testing.T) {
	var myRequests, listRequests atomic.Int32
	handler := http.NewServeMux()
	handler.HandleFunc("/my/", func(w http.ResponseWriter, r *http.Request) {
		myRequests.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
	})
	handler.HandleFunc("/lib/ajax/service.php", func(w http.ResponseWriter, r *http.Request) {
		listRequests.Add(1)
		if r.URL.Query().Get("sesskey") != "saved-key" {
			t.Errorf("sesskey = %q", r.URL.Query().Get("sesskey"))
		}
		courseListHandler(w)
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()
	src := newTestSource(t, srv.URL, session.Session{Cookie: "cookie", SessKey: "saved-key"})
	if _, err := src.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	if myRequests.Load() != 0 || listRequests.Load() != 1 {
		t.Fatalf("/my requests=%d list requests=%d", myRequests.Load(), listRequests.Load())
	}
}

func TestLiveSourceRefreshesRejectedSesskeyOnce(t *testing.T) {
	var myRequests, listRequests atomic.Int32
	handler := http.NewServeMux()
	handler.HandleFunc("/my/", func(w http.ResponseWriter, r *http.Request) {
		myRequests.Add(1)
		_, _ = w.Write([]byte(`<script>M.cfg = {"sesskey":"fresh-key","userId":42};</script>`))
	})
	handler.HandleFunc("/lib/ajax/service.php", func(w http.ResponseWriter, r *http.Request) {
		listRequests.Add(1)
		if r.URL.Query().Get("sesskey") == "stale-key" {
			_, _ = w.Write([]byte(`[{"error":true,"exception":"invalidsesskey","message":"invalid sesskey"}]`))
			return
		}
		if r.URL.Query().Get("sesskey") != "fresh-key" {
			t.Errorf("sesskey = %q", r.URL.Query().Get("sesskey"))
		}
		courseListHandler(w)
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()
	src := newTestSource(t, srv.URL, session.Session{Cookie: "cookie", SessKey: "stale-key"})
	if _, err := src.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	if myRequests.Load() != 1 || listRequests.Load() != 2 {
		t.Fatalf("/my requests=%d list requests=%d", myRequests.Load(), listRequests.Load())
	}
}

func TestLiveSourcePartialCourseFailureIsNotFatal(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/lib/ajax/service.php", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"data":[{"id":1,"shortname":"A","fullname":"Course A"},{"id":2,"shortname":"B","fullname":"Course B"}]}]`))
	})
	mux.HandleFunc("/course/view.php", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("id") == "1" {
			_, _ = w.Write([]byte(`<li id="section-0"><h3 class="sectionname">General</h3></li>`))
			return
		}
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	src := newTestSource(t, srv.URL, session.Session{Cookie: "cookie", SessKey: "saved-key"})
	snapshot, err := src.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Courses) != 2 || !strings.Contains(snapshot.Notice, "Course B") {
		t.Fatalf("partial snapshot/notice = %+v", snapshot)
	}
}

func TestFetchConcurrentRespectsWorkerLimit(t *testing.T) {
	src := &LiveSource{maxConcurrent: 3}
	var active, maximum atomic.Int32
	src.fetchConcurrent(context.Background(), 20, func(int) {
		current := active.Add(1)
		for {
			old := maximum.Load()
			if current <= old || maximum.CompareAndSwap(old, current) {
				break
			}
		}
		time.Sleep(time.Millisecond)
		active.Add(-1)
	})
	if got := maximum.Load(); got != 3 {
		t.Fatalf("max in flight = %d, want 3", got)
	}
}

func TestConcurrentRequestStartsRespectGap(t *testing.T) {
	src := &LiveSource{minRequestGap: 15 * time.Millisecond}
	var mu sync.Mutex
	starts := []time.Time{}
	const count = 4
	var group sync.WaitGroup
	for i := 0; i < count; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			if err := src.waitForRequestSlot(context.Background()); err != nil {
				t.Error(err)
				return
			}
			mu.Lock()
			starts = append(starts, time.Now())
			mu.Unlock()
		}()
	}
	group.Wait()
	for i := 1; i < len(starts); i++ {
		if starts[i].Sub(starts[i-1]) < 14*time.Millisecond {
			t.Fatalf("starts too close: %v", starts)
		}
	}
}

func TestTransientHTTPRetryHonorsRetryAfter(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			http.Error(w, "busy", http.StatusTooManyRequests)
			return
		}
		_, _ = fmt.Fprint(w, `<li id="section-0"><h3 class="sectionname">General</h3></li>`)
	}))
	defer srv.Close()
	body, err := getPageWithRetry(context.Background(), srv.URL+"/course/view.php?id=1", "cookie", srv.Client(), func(context.Context) error { return nil }, func(context.Context, time.Duration) error { return nil })
	if err != nil || len(body) == 0 || attempts.Load() != 2 {
		t.Fatalf("body=%q attempts=%d err=%v", body, attempts.Load(), err)
	}
}

func TestLiveSourceLoadsCourseActivitiesAndAssignmentDeadline(t *testing.T) {
	requests := map[string]int{}
	mux := http.NewServeMux()
	mux.HandleFunc("/my/", func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.Path]++
		w.Write([]byte(`<script>M.cfg = {"sesskey":"fresh-key","userId":42};</script>`))
	})
	mux.HandleFunc("/lib/ajax/service.php", func(w http.ResponseWriter, r *http.Request) { requests[r.URL.Path]++; courseListHandler(w) })
	mux.HandleFunc("/course/view.php", func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.Path]++
		w.Write([]byte(`<li id="section-5"><h3 class="sectionname">Week 1</h3><ul><li class="activity assign modtype_assign"><a href="/mod/assign/view.php?id=155170"><span class="instancename">Submission Praktikum 1</span></a></li><li class="activity resource modtype_resource"><a href="/mod/resource/view.php?id=165401"><span class="instancename">Slides</span></a></li></ul></li>`))
	})
	mux.HandleFunc("/mod/assign/view.php", func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.Path]++
		w.Write(fixture(t, "assign_155170.html"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	src := newTestSource(t, srv.URL, session.Session{Cookie: "fixture-cookie"})
	snapshot, err := src.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Courses) != 1 || len(snapshot.Assignments) != 1 || len(snapshot.Materials) != 1 {
		t.Fatalf("counts: %+v", snapshot)
	}
	a := snapshot.Assignments[0]
	if a.ID != "K155170" || !a.HasDue || a.Due.Format("2006-01-02 15:04") != "2026-08-31 23:55" {
		t.Fatalf("assignment: %+v", a)
	}
	if requests["/mod/assign/view.php"] != 1 || requests["/course/view.php"] != 1 || requests["/lib/ajax/service.php"] != 1 || requests["/my/"] != 1 {
		t.Fatalf("requests=%v", requests)
	}
	if strings.Contains(snapshot.Notice, "not loaded yet") {
		t.Fatalf("stale notice=%q", snapshot.Notice)
	}
}

func TestNewSourceRejectsMissingOrWrongSiteSession(t *testing.T) {
	if _, err := NewSource("https://kulon2.undip.ac.id", filepath.Join(t.TempDir(), "absent"), ""); err == nil {
		t.Fatal("expected missing session error")
	}
	path := filepath.Join(t.TempDir(), "session.json")
	if err := session.Save(path, session.Session{SiteURL: "https://other.example", Cookie: "cookie"}); err != nil {
		t.Fatal(err)
	}
	if _, err := NewSource("https://kulon2.undip.ac.id", path, ""); err == nil {
		t.Fatal("expected mismatched site error")
	}
}
