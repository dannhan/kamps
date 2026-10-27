package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ruyam/kamps/internal/store"
)

func TestPendingDownloadPersistenceChecksFile(t *testing.T) {
	for _, exists := range []bool{false, true} {
		t.Run(map[bool]string{false: "deleted before save", true: "download recreated before save"}[exists], func(t *testing.T) {
			app := downloadTestApp(t, downloadTestSource())
			path := filepath.Join(t.TempDir(), "slides.pdf")
			if exists {
				if err := os.WriteFile(path, []byte("new"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			msg := persistStateCmd(app.service, nil, map[string]store.DownloadRecord{"M1": {Path: path, Size: 3}})().(stateSavedMsg)
			if msg.err != nil {
				t.Fatal(msg.err)
			}
			state, err := store.Load(app.cfg.StatePath())
			if err != nil {
				t.Fatal(err)
			}
			if state.Downloaded("M1") != exists {
				t.Fatalf("download recorded = %v, want %v", state.Downloaded("M1"), exists)
			}
		})
	}
}

func TestMissingPendingDownloadKeepsNewerDifferentPath(t *testing.T) {
	app := downloadTestApp(t, downloadTestSource())
	oldPath := filepath.Join(t.TempDir(), "old.pdf")
	newPath := filepath.Join(t.TempDir(), "new.pdf")
	if err := os.WriteFile(newPath, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := app.service.Persist(func(state *store.State) {
		state.SetDownloadRecord("M1", store.DownloadRecord{Path: newPath, Size: 3})
	}); err != nil {
		t.Fatal(err)
	}
	msg := persistStateCmd(app.service, nil, map[string]store.DownloadRecord{"M1": {Path: oldPath}})().(stateSavedMsg)
	if msg.err != nil {
		t.Fatal(msg.err)
	}
	state, err := store.Load(app.cfg.StatePath())
	if err != nil {
		t.Fatal(err)
	}
	if state.Downloads["M1"].Path != newPath {
		t.Fatalf("newer record lost: %+v", state.Downloads["M1"])
	}
}

func TestDeletedDownloadStaysClearedWhenPendingSaveRunsAfterRefresh(t *testing.T) {
	app := downloadTestApp(t, downloadTestSource())
	material := *app.materialByID("M1")
	download := app.startDownload(material)
	msg := download().(downloadDoneMsg)
	if msg.err != nil {
		t.Fatal(msg.err)
	}
	_, save := app.Update(msg)
	if save == nil {
		t.Fatal("download did not schedule a save")
	}
	if err := os.Remove(msg.path); err != nil {
		t.Fatal(err)
	}
	app.Update(loadCmd(app.service)())
	if app.materialByID("M1").Downloaded || app.state.Downloaded("M1") {
		t.Fatal("refresh did not clear deleted file from UI state")
	}
	app.Update(save())
	if app.materialByID("M1").Downloaded || app.state.Downloaded("M1") {
		t.Fatal("pending save restored deleted file in UI state")
	}
	state, err := store.Load(app.cfg.StatePath())
	if err != nil {
		t.Fatal(err)
	}
	if state.Downloaded("M1") {
		t.Fatal("pending save restored deleted file on disk")
	}
}
