package tui

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ruyam/kamps/internal/format"
	"github.com/ruyam/kamps/internal/library"
	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/source"
	"github.com/ruyam/kamps/internal/store"
)

type downloadProgressMsg struct {
	id          string
	done, total int64
}

type downloadDoneMsg struct {
	id   string
	path string
	err  error
}

type openURLMsg struct {
	url string
	err error
}

// openExternalURL launches the user's browser. Tests replace it.
var openExternalURL = func(rawURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}
	return cmd.Start()
}

// Attach binds the running program so background downloads can report progress.
func (m *App) Attach(p *tea.Program) { m.program = p }

// materialDestination is the on-disk target for a material:
// <library_dir>/<course>/<section>/<name>.
func (m *App) materialDestination(material model.Material) string {
	courseName := material.CourseCode
	if course, ok := m.courseByID(material.CourseID); ok {
		if course.Short != "" {
			courseName = course.Short
		} else if course.Code != "" {
			courseName = course.Code
		}
		if course.WorkspacePath != "" {
			if path, err := library.SubPath(course.WorkspacePath, material.Section, material.Folder, material.Name); err == nil {
				return path
			}
		}
	}
	path, err := library.MaterialPath(m.cfg.LibraryDir, courseName, material.Section, material.Folder, material.Name)
	if err != nil {
		return library.SectionPath(m.cfg.LibraryDir, courseName, material.Section, material.Name)
	}
	return path
}

func (m *App) materialByID(id string) *model.Material {
	for i := range m.materials {
		if m.materials[i].ID == id {
			return &m.materials[i]
		}
	}
	return nil
}

func (m *App) markDownloaded(id, path string) {
	material := m.materialByID(id)
	if material == nil {
		return
	}
	material.Downloaded = true
	m.updateState(func(s *store.State) {
		s.SetDownloadRecord(id, store.DownloadRecord{Path: path})
	})
}

// startDownload runs one material download off the UI goroutine.
func (m *App) startDownload(material model.Material) tea.Cmd {
	downloader, ok := m.src.(source.Downloader)
	if !ok {
		return nil
	}
	program := m.program
	id := material.ID
	dest := m.materialDestination(material)
	return func() tea.Msg {
		path, err := downloader.Download(context.Background(), material, dest, func(done, total int64) {
			if program != nil {
				program.Send(downloadProgressMsg{id: id, done: done, total: total})
			}
		})
		return downloadDoneMsg{id: id, path: path, err: err}
	}
}

// materialTargets resolves the Materials-tab selection: checked rows first,
// otherwise the focused file, bucket or course.
func (m *App) materialTargets() []model.Material {
	var targets []model.Material
	for _, material := range m.materials {
		if m.selectedM[material.ID] {
			targets = append(targets, material)
		}
	}
	if len(targets) > 0 {
		return targets
	}
	nodes := m.matNodes()
	if m.matCursor < 0 || m.matCursor >= len(nodes) {
		return nil
	}
	return m.materialNodeFiles(nodes[m.matCursor])
}

func (m *App) openExternal() tea.Cmd {
	if m.tab != TabMaterials {
		m.toast = "switch to Materials to open a link"
		m.toastTicks = 20
		return nil
	}
	for _, material := range m.materialTargets() {
		if material.URL == "" {
			continue
		}
		rawURL := material.URL
		m.selectedM = map[string]bool{}
		return func() tea.Msg {
			if err := openExternalURL(rawURL); err != nil {
				return openURLMsg{url: rawURL, err: err}
			}
			return openURLMsg{url: rawURL}
		}
	}
	m.toast = "no link on this row"
	m.toastTicks = 20
	return nil
}

func downloadErrorText(err error) string {
	switch {
	case errors.Is(err, library.ErrFileNotFound):
		return "file no longer available on the LMS"
	case errors.Is(err, library.ErrSessionExpiredHTML), errors.Is(err, library.ErrUnauthorized):
		return "session expired — run kamps login"
	case errors.Is(err, library.ErrInsufficientDiskSpace):
		return "not enough disk space"
	case errors.Is(err, library.ErrRateLimited):
		return "server is rate limiting; try again shortly"
	case errors.Is(err, library.ErrEmptyFile):
		return "remote file is empty"
	case errors.Is(err, library.ErrIncompleteDownload):
		return "download was cut short"
	case errors.Is(err, context.Canceled):
		return "cancelled"
	}
	return err.Error()
}

func downloadProgressText(material model.Material, done, total int64) string {
	if total > 0 {
		return fmt.Sprintf("%s: %d%%", material.Name, done*100/total)
	}
	return fmt.Sprintf("%s: %s", material.Name, format.HumanSize(done))
}

func savedFileLabel(path string) string {
	if path == "" {
		return "file"
	}
	return filepath.Base(path)
}
