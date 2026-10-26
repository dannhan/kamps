package tui

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ruyam/kamps/internal/model"
)

// pendingCounts returns how many uncommitted decisions of each kind are queued.
func (m *App) pendingCounts() (tracked, missed, ignored int) {
	return m.queue.Counts()
}

// clipboardMsg reports the result of handing the preview to the terminal.
type clipboardMsg struct {
	lines int
	err   error
}

// writeClipboard copies text through OSC 52, which most terminals accept
// without an extra dependency. The sequence moves no cursor and draws nothing.
func writeClipboard(text string) clipboardMsg {
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	_, err := os.Stdout.WriteString("\x1b]52;c;" + encoded + "\a")
	return clipboardMsg{lines: len(strings.Split(text, "\n")), err: err}
}

// commitPreviewText is exactly what the queue would hand to a todo list.
func (m *App) commitPreviewText() string {
	return m.commitPreviewTextForFormat()
}

// copyCommitPreview puts the preview lines on the clipboard. The preview may
// read the existing commit file, so it is built inside the command rather than
// in Update.
func (m *App) copyCommitPreview() tea.Cmd {
	if m.queue.Len() == 0 {
		m.toast = "nothing to copy · decide in Tasks first"
		m.toastTicks = 20
		return nil
	}
	writer, err := m.exportWriter()
	if err != nil {
		m.toast = "copy failed"
		m.statusErr = err.Error()
		m.toastTicks = 30
		return nil
	}
	items := m.exportItems()
	commitFile := m.cfg.CommitFile
	return func() tea.Msg {
		text := previewText(writer, items, commitFile)
		if text == "" {
			return clipboardMsg{err: errors.New("nothing to copy")}
		}
		return writeClipboard(text)
	}
}

// dropCommitItem removes the focused decision from the queue. The decision
// itself stays applied; only the export is dropped.
func (m *App) dropCommitItem() {
	items := m.commitItems()
	if m.commitCursor < 0 || m.commitCursor >= len(items) {
		return
	}
	item := items[m.commitCursor]
	m.queue.Remove(item.assignment.ID)
	m.clampCursor()
	m.toast = "dropped " + item.assignment.ID + " from the queue"
	m.toastTicks = 20
}

func (m *App) openCommit() {
	m.switchTab(TabCommit)
}

func (m *App) closeCommit() {
	if m.tab == TabCommit {
		m.switchTab(m.returnTab)
	}
}

type commitItem struct {
	assignment model.Assignment
	action     model.Triage
}

func (m *App) commitItems() []commitItem {
	items := m.queue.Items(m.assignments)
	out := make([]commitItem, 0, len(items))
	for _, item := range items {
		out = append(out, commitItem{assignment: item.Assignment, action: item.Action})
	}
	return out
}

func (m *App) commitBody(width, height int) []string {
	contentWidth := maxInt(1, width-1) // the shared row renderer reserves a gutter
	items := m.commitItems()
	type displayRow struct {
		text    string
		focused bool
	}
	var all []displayRow
	lastAction := model.Triage(-1)
	focusIndex := 0
	cursorRow := 0
	for _, item := range items {
		if item.action != lastAction {
			lastAction = item.action
			label, style := commitActionLabel(item.action)
			all = append(all, displayRow{text: style.Render("▾ " + label)})
		}
		a := item.assignment
		course := m.shortCode(a.CourseID)
		if focusIndex == m.commitCursor {
			cursorRow = len(all)
		}
		lead := "  " + a.ID + " · " + course + " · "
		nameWidth := maxInt(1, contentWidth-lipgloss.Width(lead))
		all = append(all, displayRow{text: stRow.Render(lead + truncate(taskDisplayName(a.Name), nameWidth)), focused: focusIndex == m.commitCursor})
		focusIndex++
		if item.action == model.Tracked {
			all = append(all, displayRow{text: "    " + stDim.Render(truncate(m.taskLine(a), maxInt(1, contentWidth-4)))})
		}
		if a.HasDue && a.Due.Before(time.Now()) {
			all = append(all, displayRow{text: "    " + stOverdue.Render("will show as OVERDUE")})
		}
		if similar := m.similarManualTask(a); similar != "" {
			line := fmt.Sprintf("similar manual task: %q", similar)
			all = append(all, displayRow{text: "    " + stDim.Render(truncate(line, maxInt(1, contentWidth-4)))})
		}
		all = append(all, displayRow{text: ""})
	}
	if len(all) == 0 {
		all = append(all,
			displayRow{text: stEmpty.Render("no decisions queued")},
			displayRow{text: stFaint.Render("  decide with t/m/i in Tasks; y copies the lines")},
		)
	}
	start, end := windowRange(len(all), cursorRow, height)
	rows := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		rows = append(rows, paneRow(all[i].text, width, rowHighlight{cursor: all[i].focused}))
	}
	return rows
}

func commitActionLabel(action model.Triage) (string, lipgloss.Style) {
	switch action {
	case model.Tracked:
		return "done (track)", stFlagT
	case model.Missed:
		return "missed", stFlagM
	case model.Ignored:
		return "ignore", stFlagI
	default:
		return "undecided", stFlagNew
	}
}

func (m *App) commitContent(width, height int) (string, string, []string, []string) {
	return "review & commit · " + m.exportStatusLine(), "", []string{"↑↓ move", "w write", "y copy", "x drop", "esc back", "? more"}, m.commitBody(width, height)
}
