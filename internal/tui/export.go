package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ruyam/kamps/internal/export"
)

type exportMsg struct {
	result export.Result
	err    error
}

func (m *App) exportWriter() (export.Writer, error) {
	return export.For(m.cfg.ExportFormat)
}

func (m *App) exportItems() []export.Item {
	items := m.commitItems()
	out := make([]export.Item, 0, len(items))
	for _, item := range items {
		a := item.assignment
		out = append(out, export.Item{
			ID:     a.ID,
			Course: m.shortCode(a.CourseID),
			Name:   a.Name,
			Due:    a.Due,
			HasDue: a.HasDue,
			Action: item.action,
		})
	}
	return out
}

// commitPreviewText is exactly what the selected exporter would append.
func (m *App) commitPreviewTextForFormat() string {
	writer, err := m.exportWriter()
	if err != nil {
		return ""
	}
	items := m.exportItems()
	if previewer, ok := writer.(export.Previewer); ok {
		text, _ := previewer.Preview(items)
		return text
	}
	var existing []byte
	if m.cfg.CommitFile != "" {
		existing, _ = os.ReadFile(m.cfg.CommitFile)
	}
	text, _ := writer.Render(items, existing)
	return text
}

func (m *App) writeCommitPreview() tea.Cmd {
	writer, err := m.exportWriter()
	if err != nil {
		m.toast = err.Error()
		m.toastTicks = 30
		return nil
	}
	items := m.exportItems()
	if len(items) == 0 {
		m.toast = "nothing to commit · decide in Tasks first"
		m.toastTicks = 20
		return nil
	}
	if writer.Format() != "taskwarrior" && m.cfg.CommitFile == "" {
		m.toast = "set commit_file or pass --commit-file"
		m.toastTicks = 30
		return nil
	}
	path := m.cfg.CommitFile
	if writer.Format() == "taskwarrior" {
		path = ""
	}
	return func() tea.Msg {
		result, writeErr := writer.Write(path, items)
		return exportMsg{result: result, err: writeErr}
	}
}

func (m *App) exportStatusLine() string {
	writer, err := m.exportWriter()
	if err != nil {
		return "export: " + err.Error()
	}
	target := m.cfg.CommitFile
	if writer.Format() == "taskwarrior" {
		target = "task CLI"
	}
	if target == "" {
		target = "(no commit_file)"
	}
	return fmt.Sprintf("%s → %s", writer.Format(), target)
}
