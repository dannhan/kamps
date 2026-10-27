package tui

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ruyam/kamps/internal/cache"
	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/kulon2"
	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/session"
	"github.com/ruyam/kamps/internal/source"
)

func TestExpiredSessionStartupSettlesBeforePublishingFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login/index.php" {
			_, _ = fmt.Fprint(w, "login")
			return
		}
		http.Redirect(w, r, "/login/index.php", http.StatusFound)
	}))
	defer server.Close()
	for _, cached := range []bool{true, false} {
		t.Run(fmt.Sprintf("cached=%v", cached), func(t *testing.T) {
			cfg := config.Config{StateDir: t.TempDir(), SiteURL: server.URL}
			if err := session.Save(cfg.SessionPath(), session.Session{SiteURL: server.URL, Cookie: "expired-fixture", SessKey: "old-key", ExpiresAt: time.Now().Add(-time.Hour)}); err != nil {
				t.Fatal(err)
			}
			src, err := kulon2.NewSource(server.URL, cfg.SessionPath(), cfg.StateDir)
			if err != nil {
				t.Fatal(err)
			}
			snapshot := source.Snapshot{
				Courses:     []model.Course{{ID: 1, Name: "Numerical Methods"}},
				Assignments: []model.Assignment{{ID: "K1", CourseID: 1, Name: "Cached project report", HasDue: true, Due: dueOn(workFirstNow(), 3)}},
			}
			if cached {
				if err := cache.Save(cfg.CachePath(), cfg.SiteURL, snapshot); err != nil {
					t.Fatal(err)
				}
			}
			app := New(src, cfg, "test")
			app.now = workFirstNow
			cmd := app.Init()
			for _, size := range [][2]int{{80, 24}, {120, 35}, {160, 45}} {
				app.width, app.height = size[0], size[1]
				view := stripANSI(app.View())
				if strings.Contains(view, "error ·") {
					t.Fatal("startup displayed an error before fallback completed")
				}
				if cached && !strings.Contains(view, "Cached project report") {
					t.Fatal("startup hid cached coursework")
				}
			}
			msg, ok := cmd().(errMsg)
			if !ok {
				t.Fatal("expired server failed to produce load error")
			}
			if (msg.cached != nil) != cached {
				t.Fatal("load command did not resolve cache fallback")
			}
			if !strings.Contains(msg.err.Error(), "kamps login") {
				t.Fatal("expired-session error lacks recovery instruction")
			}
			_, followup := app.Update(msg)
			if followup != nil {
				t.Fatal("failure published before cache fallback completed")
			}
			for _, size := range [][2]int{{80, 24}, {120, 35}, {160, 45}} {
				app.width, app.height = size[0], size[1]
				view := stripANSI(app.View())
				if !strings.Contains(view, "kamps login") {
					t.Fatal("settled state lost login instruction")
				}
				if cached {
					if strings.Contains(view, "error ·") || !strings.Contains(view, "Cached project report") || !app.offline || app.loadError != "" {
						t.Fatalf("cached startup replaced coursework with an error:\n%s", view)
					}
				} else if !strings.Contains(view, "error ·") || app.loadError == "" {
					t.Fatal("uncached startup hid its failure")
				}
				if dir := os.Getenv("KAMPS_RENDER_DIR"); dir != "" {
					if err := os.MkdirAll(dir, 0700); err != nil {
						t.Fatal(err)
					}
					name := fmt.Sprintf("expired-%dx%d-cached-%v.txt", size[0], size[1], cached)
					if err := os.WriteFile(filepath.Join(dir, name), []byte(view+"\n"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			for i := 0; i < 100; i++ {
				app.advance()
			}
			if !strings.Contains(stripANSI(app.View()), "kamps login") {
				t.Fatal("session warning expired like a toast")
			}
			if cached {
				sync := stripANSI(strings.Join(app.syncBody(100, 20), "\n"))
				if !strings.Contains(sync, "last error") || !strings.Contains(sync, "kamps login") {
					t.Fatal("Sync lost failure details")
				}
			}
		})
	}
}

func TestSuccessfulRetryClearsOfflineSessionFailure(t *testing.T) {
	cfg := config.Config{StateDir: t.TempDir(), SiteURL: "https://lms.example.test"}
	snapshot := source.Snapshot{Courses: []model.Course{{ID: 1, Name: "Cached course"}}}
	if err := cache.Save(cfg.CachePath(), cfg.SiteURL, snapshot); err != nil {
		t.Fatal(err)
	}
	src := &stubSource{snapshot: snapshot, err: errors.New("session expired; run kamps login")}
	app := New(src, cfg, "test")
	app.Update(app.Init()())
	if !app.offline || app.syncError == "" {
		t.Fatal("fixture did not enter cached offline state")
	}
	src.err = nil
	app.Update(loadCmd(app.service)())
	if app.offline || app.syncError != "" || app.loadError != "" || app.cacheNotice != "" {
		t.Fatal("successful retry retained offline/session failure")
	}
}
