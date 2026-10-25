package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *App) View() string {
	w, h := m.width, m.height
	if w <= 0 {
		w = 100
	}
	if h <= 0 {
		h = 30
	}
	if w < 80 {
		w = maxInt(1, w)
	}
	if h < 24 {
		h = maxInt(1, h)
	}

	title, right, hints, rows := m.tabContent(maxInt(1, w), maxInt(0, h-5))
	if m.loadError != "" {
		rows = []string{stOverdue.Render("error · " + m.loadError)}
	} else if !m.loaded {
		rows = []string{stDim.Render("loading… fetching courses, assignments, and materials")}
	}
	if m.helpOn {
		title, right, hints, rows = "help", "", []string{"↑↓ move", "tab switch", "? close"}, m.helpLines(maxInt(1, w))
	}
	if m.cacheNotice != "" && !m.helpOn {
		label := "cached · "
		if m.offline {
			label = "offline · "
		}
		right = rightAligned(right, stDim.Render(label+m.cacheNotice))
	} else if m.loadNotice != "" && !m.helpOn {
		right = rightAligned(right, stDim.Render("partial · "+m.loadNotice))
	}
	if w < 80 || h < 24 {
		title, right, rows = "terminal too small", "", []string{"terminal too small"}
		hints = []string{"? more"}
	}
	bodyHeight := maxInt(0, h-5)
	lines := []string{
		m.tabBar(w),
		titleLine(title, right, w),
		rule(w),
	}
	lines = append(lines, bodyLines(rows, w, bodyHeight)...)
	lines = append(lines, rule(w), m.helpBar(w, hints))
	return strings.Join(lines, "\n")
}

// tabBar is the top line: the app name, the tabs, and a right-aligned status.
// No box — gh-dash lets the top and bottom bars sit flush against the screen.
func (m *App) tabBar(w int) string {
	labels := []string{"Tasks", "Materials", "Sync", "Commit"}
	const cellWidth = 14
	var cells []string
	for i, label := range labels {
		count := ""
		switch Tab(i) {
		case TabTasks:
			count = fmt.Sprint(len(m.assignments))
		case TabMaterials:
			count = fmt.Sprint(len(m.materials))
		case TabCommit:
			count = fmt.Sprint(len(m.commitItems()))
		}
		text := pad(label, 9, false) + " " + pad(count, 2, true)
		cells = append(cells, fixedTabCell(text, m.tab == Tab(i), cellWidth))
	}
	left := strings.Join(cells, "")
	status := stFaint.Render("backup · sync · cookie")
	statusWidth := lipgloss.Width(status)
	if w <= len(cells)*cellWidth+statusWidth {
		return fitLine(left+status, w)
	}
	return fitLine(left+strings.Repeat(" ", w-len(cells)*cellWidth-statusWidth)+status, w)
}

func (m *App) helpBar(w int, items []string) string {
	inner := w - 1
	if m.searching {
		left := hints([]string{"type to search", "↑↓ move", "enter done", "esc cancel"})
		return " " + barLine(left, m.taskQueueLabel(), inner)
	}
	if m.tab == TabTasks {
		return " " + m.taskFooter(inner)
	}
	right := m.queueLabel()
	if m.tab == TabCommit {
		right = stKey.Render(fmt.Sprintf("queue: %d", len(m.pending)))
	}
	if f := m.footer(); f != "" {
		right = f
	}
	if len(items) > 6 {
		items = items[:6]
	}
	if len(items) == 0 || items[len(items)-1] != "? more" {
		items = append(items, "? more")
	}
	available := maxInt(0, inner-lipgloss.Width(right)-3)
	for lipgloss.Width(hints(items)) > available && len(items) > 1 {
		items = append(items[:len(items)-2], items[len(items)-1])
	}
	return " " + barLine(hints(items), right, inner)
}

// barLine lays a left and a right segment on one line of exactly w columns,
// dropping the right segment when the two would collide.
func barLine(left, right string, w int) string {
	if right == "" || lipgloss.Width(left)+3+lipgloss.Width(right) > w {
		return fitLine(left, w)
	}
	gap := maxInt(1, w-lipgloss.Width(left)-lipgloss.Width(right))
	return fitLine(left+strings.Repeat(" ", gap)+right, w)
}

func (m *App) tabContent(width, height int) (string, string, []string, []string) {
	var title, right string
	var hints []string
	var rows []string
	switch m.tab {
	case TabTasks:
		title, right, hints, rows = m.tasksContent(width, height)
	case TabMaterials:
		title, right, hints, rows = m.materialsContent(width, height)
	case TabSync:
		title, right, hints, rows = m.syncContent(width, height)
	case TabCommit:
		title, right, hints, rows = m.commitContent(width, height)
	}
	if m.searching {
		hints = []string{"type to search", "↑↓ move", "enter done", "esc cancel"}
	}
	return title, right, hints, rows
}

// rightAligned joins the non-empty pieces of a title's right side with the
// usual two-space gap.
func rightAligned(parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, "  ")
}

func (m *App) helpLines(w int) []string {
	full := strings.Split(m.help.FullHelpView(m.keys.FullHelp()), "\n")
	legend := stFlagNew.Render("○") + " " + stDim.Render("new") + "  " +
		stFlagT.Render("◐") + " " + stDim.Render("tracked") + "  " +
		stDone.Render("●") + " " + stDim.Render("done") + "  " +
		stFlagI.Render("◆") + " " + stDim.Render("ignored") + "  " +
		stFlagM.Render("×") + " " + stDim.Render("missed")
	return append(append(full, legend), stFaint.Render("mouse off: drag to select text"))
}

// footer shows the last thing the app did. It replaces the commit-queue readout
// while it is on screen, the way gh-dash surfaces notifications in its status
// bar rather than in a modal.
func (m *App) footer() string {
	if m.toast != "" {
		return stKey.Render("✻ ") + stFaint.Render(m.toast+" ")
	}
	return ""
}
