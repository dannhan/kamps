package tui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/library"
	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/source"
)

type fakeDownloadSource struct {
	snapshot source.Snapshot
	files    map[string]string
	errs     map[string]error
	mu       sync.Mutex
	requests []string
}

func (f *fakeDownloadSource) Label() string { return "fake" }

func (f *fakeDownloadSource) Load(context.Context) (source.Snapshot, error) {
	return f.snapshot, nil
}

func (f *fakeDownloadSource) Download(_ context.Context, material model.Material, destPath string, progress func(done, total int64)) (string, error) {
	f.mu.Lock()
	f.requests = append(f.requests, material.ID)
	f.mu.Unlock()
	if err := f.errs[material.ID]; err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return "", err
	}
	body := f.files[material.ID]
	if err := os.WriteFile(destPath, []byte(body), 0644); err != nil {
		return "", err
	}
	if progress != nil {
		progress(int64(len(body)), int64(len(body)))
	}
	return destPath, nil
}

func (f *fakeDownloadSource) requestCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.requests)
}

func downloadTestSource() *fakeDownloadSource {
	return &fakeDownloadSource{
		snapshot: source.Snapshot{
			Courses: []model.Course{{ID: 1, Code: "MIK1", Short: "metnum", Name: "Metode Numerik"}},
			Materials: []model.Material{
				{ID: "M1", CourseID: 1, CourseCode: "metnum", Name: "slides.pdf", Section: "week-1", Kind: "file", URL: "https://lms/mod/resource/view.php?id=1", Size: 5},
				{ID: "M2", CourseID: 1, CourseCode: "metnum", Name: "Reading", Section: "week-1", Kind: "url", URL: "https://example.test/reading"},
			},
		},
		files: map[string]string{"M1": "hello"},
		errs:  map[string]error{},
	}
}

func downloadTestApp(t *testing.T, src *fakeDownloadSource) *App {
	t.Helper()
	cfg := config.Config{StateDir: t.TempDir(), LibraryDir: t.TempDir()}
	app := New(src, cfg, "test")
	app.Update(app.Init()())
	app.tab = TabMaterials
	return app
}

func focusMaterial(t *testing.T, app *App, id string) {
	t.Helper()
	for i, n := range app.matNodes() {
		if n.kind == nodeFile && n.mat.ID == id {
			app.matCursor = i
			return
		}
	}
	t.Fatalf("material %s is not in the tree", id)
}

func TestDownloadFetchesFileAndMarksDownloaded(t *testing.T) {
	src := downloadTestSource()
	app := downloadTestApp(t, src)
	focusMaterial(t, app, "M1")

	cmd := app.download()
	if cmd == nil {
		t.Fatal("expected a download command")
	}
	app.Update(cmd())

	target := filepath.Join(app.cfg.LibraryDir, "metnum", "week-1", "slides.pdf")
	body, err := os.ReadFile(target)
	if err != nil || string(body) != "hello" {
		t.Fatalf("downloaded file = %q, err = %v", body, err)
	}
	if !app.materialByID("M1").Downloaded {
		t.Fatal("material was not marked downloaded")
	}
	if _, ok := app.state.Downloads["M1"]; !ok {
		t.Fatal("download was not persisted to state")
	}
}

func TestDownloadFailureLeavesFlagUnset(t *testing.T) {
	src := downloadTestSource()
	src.errs["M1"] = library.ErrFileNotFound
	app := downloadTestApp(t, src)
	focusMaterial(t, app, "M1")

	app.Update(app.download()())

	if app.materialByID("M1").Downloaded {
		t.Fatal("failed download must not be marked downloaded")
	}
	if _, ok := app.state.Downloads["M1"]; ok {
		t.Fatal("failed download must not be persisted")
	}
	if got := app.toast; got == "" || !strings.Contains(got, "no longer available") {
		t.Fatalf("toast = %q", got)
	}
}

func TestDownloadSkipsExternalLinks(t *testing.T) {
	src := downloadTestSource()
	app := downloadTestApp(t, src)
	focusMaterial(t, app, "M2")

	if cmd := app.download(); cmd != nil {
		t.Fatal("external links must not start a download")
	}
	if src.requestCount() != 0 {
		t.Fatalf("external link caused %d downloads", src.requestCount())
	}
	if !strings.Contains(app.toast, "external link") {
		t.Fatalf("toast = %q", app.toast)
	}
}

func TestOpenExternalLaunchesBrowser(t *testing.T) {
	src := downloadTestSource()
	app := downloadTestApp(t, src)
	focusMaterial(t, app, "M2")

	opened := ""
	oldOpen := openExternalURL
	openExternalURL = func(rawURL string) error {
		opened = rawURL
		return nil
	}
	t.Cleanup(func() { openExternalURL = oldOpen })

	cmd := app.openExternal()
	if cmd == nil {
		t.Fatal("expected an open command")
	}
	app.Update(cmd())
	if opened != "https://example.test/reading" {
		t.Fatalf("opened = %q", opened)
	}
}

func TestOpenExternalReportsBrowserFailure(t *testing.T) {
	src := downloadTestSource()
	app := downloadTestApp(t, src)
	focusMaterial(t, app, "M2")

	oldOpen := openExternalURL
	openExternalURL = func(string) error { return errors.New("no browser") }
	t.Cleanup(func() { openExternalURL = oldOpen })

	app.Update(app.openExternal()())
	if !strings.Contains(app.toast, "could not open browser") {
		t.Fatalf("toast = %q", app.toast)
	}
}

func TestDownloadInFlightIsNotStartedTwice(t *testing.T) {
	src := downloadTestSource()
	app := downloadTestApp(t, src)
	focusMaterial(t, app, "M1")
	app.downloads["M1"] = true

	if cmd := app.download(); cmd != nil {
		t.Fatal("an in-flight download must not start again")
	}
	if src.requestCount() != 0 {
		t.Fatalf("duplicate download request count = %d", src.requestCount())
	}
}

func TestMaterialDestinationUsesLibraryLayout(t *testing.T) {
	src := downloadTestSource()
	app := downloadTestApp(t, src)
	material := *app.materialByID("M1")
	want := filepath.Join(app.cfg.LibraryDir, "metnum", "week-1", "slides.pdf")
	if got := app.materialDestination(material); got != want {
		t.Fatalf("destination = %q, want %q", got, want)
	}
}

func TestDownloadErrorText(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want string
	}{
		{library.ErrSessionExpiredHTML, "session expired"},
		{library.ErrInsufficientDiskSpace, "disk space"},
		{library.ErrRateLimited, "rate limiting"},
		{library.ErrEmptyFile, "empty"},
		{library.ErrIncompleteDownload, "cut short"},
	} {
		if got := downloadErrorText(tc.err); !strings.Contains(got, tc.want) {
			t.Errorf("downloadErrorText(%v) = %q, want %q", tc.err, got, tc.want)
		}
	}
}
