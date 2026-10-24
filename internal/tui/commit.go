package tui

import (
	"encoding/base64"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ruyam/kamps/internal/model"
)

// pendingCounts returns how many uncommitted decisions of each kind are queued.
func (m *App) pendingCounts() (tracked, missed, ignored int) {
	for _, t := range m.pending {
		switch t {
		case model.Tracked:
			tracked++
		case model.Missed:
			missed++
		case model.Ignored:
			ignored++
		}
	}
	return
}

// clipboardMsg reports the result of handing the preview to the terminal.
type clipboardMsg struct {
	lines int
	err   error
}

// clipboardCmd copies text through OSC 52, which most terminals accept without
// an extra dependency. The sequence moves no cursor and draws nothing.
func clipboardCmd(text string) tea.Cmd {
	return func() tea.Msg {
		encoded := base64.StdEncoding.EncodeToString([]byte(text))
		_, err := os.Stdout.WriteString("\x1b]52;c;" + encoded + "\a")
		return clipboardMsg{lines: len(strings.Split(text, "\n")), err: err}
	}
}

// commitPreviewText is exactly what the queue would hand to a todo list.
func (m *App) commitPreviewText() string {
	items := m.commitItems()
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, m.taskLine(item.assignment))
	}
	return strings.Join(lines, "\n")
}

// copyCommitPreview puts the preview lines on the clipboard, or explains that
// there is nothing queued yet.
func (m *App) copyCommitPreview() tea.Cmd {
	text := m.commitPreviewText()
	if text == "" {
		m.toast = "nothing to copy · decide in Tasks first"
		m.toastTicks = 20
		return nil
	}
	return clipboardCmd(text)
}

// dropCommitItem removes the focused decision from the queue. The decision
// itself stays applied; only the export is dropped.
func (m *App) dropCommitItem() {
	items := m.commitItems()
	if m.commitCursor < 0 || m.commitCursor >= len(items) {
		return
	}
	item := items[m.commitCursor]
	delete(m.pending, item.assignment.ID)
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

// queueLabel summarises what `c` would commit.
func (m *App) queueLabel() string {
	t, mi, ig := m.pendingCounts()
	if t == 0 && mi == 0 && ig == 0 {
		return stFaint.Render("queue empty ")
	}
	parts := []string{}
	if t > 0 {
		parts = append(parts, stFlagT.Render(fmt.Sprintf("%d track", t)))
	}
	if mi > 0 {
		parts = append(parts, stFlagM.Render(fmt.Sprintf("%d missed", mi)))
	}
	if ig > 0 {
		parts = append(parts, stFlagI.Render(fmt.Sprintf("%d ignore", ig)))
	}
	return strings.Join(parts, stFaint.Render(" · ")) + stFaint.Render(" to commit ")
}

type commitItem struct {
	assignment model.Assignment
	action     model.Triage
}

func (m *App) commitItems() []commitItem {
	items := make([]commitItem, 0, len(m.pending))
	for id, action := range m.pending {
		if assignment := m.byID(id); assignment != nil {
			items = append(items, commitItem{assignment: *assignment, action: action})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].action != items[j].action {
			return items[i].action < items[j].action
		}
		return items[i].assignment.ID < items[j].assignment.ID
	})
	return items
}

func (m *App) commitBody(width, height int) []string {
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
		nameWidth := maxInt(1, width-lipgloss.Width(lead))
		all = append(all, displayRow{text: stRow.Render(lead + truncate(a.Name, nameWidth)), focused: focusIndex == m.commitCursor})
		focusIndex++
		if item.action == model.Tracked {
			all = append(all, displayRow{text: "    " + stDim.Render(truncate(m.taskLine(a), maxInt(1, width-4)))})
		}
		if a.HasDue && a.Due.Before(time.Now()) {
			all = append(all, displayRow{text: "    " + stOverdue.Render("will show as OVERDUE")})
		}
		if similar := m.similarManualTask(a); similar != "" {
			line := fmt.Sprintf("similar manual task: %q", similar)
			all = append(all, displayRow{text: "    " + stDim.Render(truncate(line, maxInt(1, width-4)))})
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
		rows = append(rows, sharedFocusRow(all[i].text, width, all[i].focused))
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
	return "review & commit · y copies the preview", "", []string{"↑↓ move", "y copy", "x drop", "esc back"}, m.commitBody(width, height)
}
