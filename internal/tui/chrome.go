package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

type sizeClass int

const (
	sizeTiny sizeClass = iota
	sizeCompact
	sizeStandard
	sizeWide
)

func sizeClassFor(w, h int) sizeClass {
	switch {
	case w < 56 || h < 16:
		return sizeTiny
	case w < 80 || h < 24:
		return sizeCompact
	case w >= 110 && h >= 28:
		return sizeWide
	default:
		return sizeStandard
	}
}

func (m *App) sizeClass() sizeClass { return sizeClassFor(m.width, m.height) }

// bodyHeightFor is the single source of truth for the body height. Tiny omits
// the bottom rule; the other classes use one or two footer rows.
func bodyHeightFor(w, h int) int {
	class := sizeClassFor(w, h)
	if class == sizeTiny {
		return maxInt(0, h-4)
	}
	footerRows := 1
	if class >= sizeStandard {
		footerRows = 2
	}
	return maxInt(0, h-4-footerRows)
}

func (m *App) View() string {
	w, h := m.width, m.height
	if w <= 0 {
		w = 100
	}
	if h <= 0 {
		h = 30
	}
	w, h = maxInt(1, w), maxInt(1, h)

	if w < 40 || h < 10 {
		if m.helpOn {
			return m.tinyHelp(w, h)
		}
		return "terminal too small · q quit · ? help"
	}

	class := sizeClassFor(w, h)
	bodyHeight := bodyHeightFor(w, h)

	title, right, _, rows := m.tabContent(w, bodyHeight)
	if m.loadError != "" {
		rows = []string{stOverdue.Render("error · " + m.loadError)}
	} else if !m.loaded {
		rows = []string{stDim.Render("loading… fetching courses, assignments, and materials")}
	}
	if m.paletteOn {
		title, right = "commands", ""
		rows = m.paletteLines(w)
	} else if m.helpOn {
		title, right = "help", ""
		clipped := false
		rows, clipped = m.helpLines(w, bodyHeight)
		if clipped {
			title = "help · j/k scroll"
		}
	}

	tabBar := m.tabBar(w)
	if class == sizeTiny {
		tabBar = m.tabBarTiny(w)
	}
	lines := []string{
		tabBar,
		titleLine(title, right, w),
		rule(w),
	}
	lines = append(lines, bodyLines(rows, w, bodyHeight)...)
	if class != sizeTiny {
		lines = append(lines, rule(w))
	}
	lines = append(lines, m.footerLines(w, class)...)
	return strings.Join(lines, "\n")
}

// tabBar is the top line: the tabs and a right-aligned status. No box — the top
// and bottom bars sit flush against the screen.
func (m *App) tabBar(w int) string {
	labels := []string{"1 Tasks", "2 Materials", "3 Sync", "4 Commit"}
	const cellWidth = 13
	var cells []string
	for i, label := range labels {
		cells = append(cells, fixedTabCell(label, m.tab == Tab(i), cellWidth))
	}
	left := strings.Join(cells, "")
	return barLine(left, m.tabBarStatus(w-lipgloss.Width(left)-3), w)
}

// tabBarStatus degrades the right-hand status instead of dropping it whole:
// full summary, then per-state counts, then just overdue, then nothing.
func (m *App) tabBarStatus(available int) string {
	if available <= 0 {
		return ""
	}
	if m.tab == TabMaterials {
		total, local := len(m.materials), m.downloadedMaterials()
		if total == 0 {
			return ""
		}
		if full := m.tabSummary(); lipgloss.Width(full) <= available {
			return full
		}
		compact := stFaint.Render(fmt.Sprintf("%d/%d local", local, total))
		if lipgloss.Width(compact) <= available {
			return compact
		}
		return ""
	}
	if summary := m.tabSummary(); lipgloss.Width(summary) <= available {
		return summary
	}
	if totals := m.taskTotals(available); totals != "" {
		return totals
	}
	if overdue := m.overdueCount(); overdue > 0 {
		label := stOverdue.Render(fmt.Sprintf("%d overdue", overdue))
		if lipgloss.Width(label) <= available {
			return label
		}
	}
	return ""
}

// tabBarTiny keeps only the numeric tab cells plus the active tab name.
func (m *App) tabBarTiny(w int) string {
	labels := []string{"1", "2", "3", "4"}
	var cells []string
	for i, label := range labels {
		cells = append(cells, fixedTabCell(label, m.tab == Tab(i), 3))
	}
	return barLine(strings.Join(cells, ""), m.tab.String(), w)
}

// footerRowCount keeps the two-row footer on Standard/Wide terminals and
// collapses it to one row on Compact/Tiny.
func (m *App) footerRowCount() int {
	if m.sizeClass() >= sizeStandard {
		return 2
	}
	return 1
}

// footerHintItems is the single hint list for the footer. Search, palette and
// help own their own modes; the trees and queue use the tab's priority list.
func (m *App) footerHintItems() []string {
	switch {
	case m.helpOn:
		if m.helpMaxScroll() > 0 {
			return []string{"j/k scroll", keyHintLabel(m.keys.Help, "close")}
		}
		return []string{keyHintLabel(m.keys.Help, "close")}
	case m.paletteOn:
		return []string{"type to filter", "↑↓ move", keyHintLabel(m.keys.Confirm, "run"), keyHintLabel(m.keys.Back, "close")}
	case m.searching:
		return []string{"type search", "↑↓ move", keyHintLabel(m.keys.Confirm, "done"), keyHintLabel(m.keys.Back, "cancel")}
	}
	detail := keyHintLabel(m.keys.Confirm, "detail")
	switch m.detailMode() {
	case detailSplit:
		detail = keyHintLabel(m.keys.Confirm, "hide")
	case detailReplace:
		detail = keyHintLabel(m.keys.Back, "back")
	}
	if m.sizeClass() == sizeTiny {
		return []string{keyHint(m.keys.Tabs), keyHint(m.keys.Quit), keyHintLabel(m.keys.Help, "all keys")}
	}
	switch m.tab {
	case TabTasks:
		return []string{
			"↑↓ move", detail, keyHint(m.keys.Select),
			keyHintGroup("decide", m.keys.Track, m.keys.Missed, m.keys.Ignore),
			keyHintLabel(m.keys.Commit, "review"), keyHint(m.keys.Search), keyHint(m.keys.Filter),
			keyHintLabel(m.keys.Help, "all keys"),
		}
	case TabMaterials:
		return []string{
			"↑↓ move", detail, keyHint(m.keys.Select), keyHint(m.keys.Download), keyHint(m.keys.Open),
			keyHint(m.keys.Search), keyHint(m.keys.Filter), keyHintLabel(m.keys.Help, "all keys"),
		}
	case TabSync:
		return []string{keyHint(m.keys.Sync), keyHintLabel(m.keys.Help, "all keys")}
	case TabCommit:
		return []string{
			"↑↓ move", keyHintLabel(m.keys.Write, "write"), keyHintLabel(m.keys.Copy, "copy"),
			keyHintLabel(m.keys.Drop, "drop"), keyHintLabel(m.keys.Back, "back"),
			keyHintLabel(m.keys.Help, "all keys"),
		}
	}
	return []string{keyHintLabel(m.keys.Help, "all keys")}
}

func keyHint(binding key.Binding) string {
	help := binding.Help()
	return help.Key + " " + help.Desc
}

func keyHintLabel(binding key.Binding, label string) string {
	return binding.Help().Key + " " + label
}

func keyHintGroup(label string, bindings ...key.Binding) string {
	keys := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		keys = append(keys, binding.Help().Key)
	}
	return strings.Join(keys, "/") + " " + label
}

// renderHintItems keeps the last item (the safety hint) and drops lower-priority
// items from the right until the line fits.
func renderHintItems(items []string, width int) string {
	if len(items) == 0 {
		return ""
	}
	required := items[len(items)-1]
	kept := make([]string, 0, len(items))
	for _, item := range items[:len(items)-1] {
		candidate := append(append([]string{}, kept...), item, required)
		if lipgloss.Width(hints(candidate)) > width {
			break
		}
		kept = append(kept, item)
	}
	kept = append(kept, required)
	return hints(kept)
}

func (m *App) footerLines(w int, class sizeClass) []string {
	inner := maxInt(1, w-1)
	summary := m.footerSummary()
	if class <= sizeCompact {
		if m.statusErr != "" {
			return []string{" " + fitLine(stOverdue.Render("err: "+m.statusErr)+stFaint.Render("  esc dismiss"), inner)}
		}
		return []string{" " + barLine(renderHintItems(m.footerHintItems(), inner), summary, inner)}
	}
	return []string{
		" " + fitLine(renderHintItems(m.footerHintItems(), inner), inner),
		" " + m.footerStatusLine(inner, summary),
	}
}

func (m *App) footerStatusLine(width int, summary string) string {
	left := m.modeChip()
	if m.statusErr != "" {
		left += " " + stOverdue.Render("err: "+m.statusErr) + stFaint.Render("  esc dismiss")
	} else if m.toast != "" {
		left += " " + stFaint.Render("✻ "+m.toast+" ")
	}
	return barLine(left, summary, width)
}

func (m *App) modeChip() string {
	label := strings.ToUpper(m.tab.String())
	switch {
	case m.helpOn:
		label = "HELP"
	case m.paletteOn:
		label = "COMMANDS"
	case m.searching:
		label = "SEARCH"
	}
	return stKey.Render(label)
}

// footerSummary is the persistent right side of the footer: selection count,
// queue totals, active filter, and offline/cache state.
func (m *App) footerSummary() string {
	var parts []string
	if sel := m.selectionLabel(); sel != "" {
		parts = append(parts, sel)
	}
	if queue := m.queueSummary(); queue != "" {
		parts = append(parts, queue)
	}
	if (m.tab == TabTasks || m.tab == TabMaterials) && m.filter != "" && m.filter != "all" {
		parts = append(parts, stFaint.Render("filter: "+m.filter))
	}
	switch {
	case m.cacheNotice != "":
		label := "cached · "
		if m.offline {
			label = "offline · "
		}
		parts = append(parts, stDim.Render(label+m.cacheNotice))
	case m.loadNotice != "":
		parts = append(parts, stDim.Render("partial · "+m.loadNotice))
	}
	return strings.Join(parts, stFaint.Render("  "))
}

// queueSummary is the one queue readout used by the footer.
func (m *App) queueSummary() string {
	tracked, missed, ignored := m.pendingCounts()
	if tracked+missed+ignored == 0 {
		return ""
	}
	var parts []string
	if tracked > 0 {
		parts = append(parts, stFlagT.Render(fmt.Sprintf("%d track", tracked)))
	}
	if missed > 0 {
		parts = append(parts, stFlagM.Render(fmt.Sprintf("%d missed", missed)))
	}
	if ignored > 0 {
		parts = append(parts, stFlagI.Render(fmt.Sprintf("%d ignore", ignored)))
	}
	return stFaint.Render("queue: ") + strings.Join(parts, stFaint.Render(" · "))
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

type helpSection struct {
	title string
	items []key.Binding
}

// helpSections is tab-aware: only bindings that do something on the current
// tab are listed. The bindings themselves are the single source of truth.
func (m *App) helpSections() []helpSection {
	navigation := []key.Binding{
		m.keys.Up, m.keys.Down, m.keys.Top, m.keys.Bottom,
		m.keys.PageUp, m.keys.PageDown, m.keys.Left, m.keys.Right,
		m.keys.NextTab, m.keys.PrevTab, m.keys.Tabs, m.keys.Confirm, m.keys.Back,
	}
	modes := []key.Binding{m.keys.Search, m.keys.Filter, m.keys.Palette, m.keys.Help, m.keys.Quit}
	sections := []helpSection{
		{title: "Navigation", items: navigation},
		{title: "Modes", items: modes},
	}
	switch m.tab {
	case TabTasks:
		sections = append(sections,
			helpSection{title: "Selection & bulk", items: []key.Binding{m.keys.Select, m.keys.All, m.keys.Range, m.keys.Invert, m.keys.Undo}},
			helpSection{title: "Decide", items: []key.Binding{m.keys.Track, m.keys.Missed, m.keys.Ignore, m.keys.Undecide}},
			helpSection{title: "Queue", items: []key.Binding{m.keys.Commit, m.keys.Write, m.keys.Copy, m.keys.Drop, m.keys.DropAll}},
		)
	case TabMaterials:
		sections = append(sections,
			helpSection{title: "Materials", items: []key.Binding{m.keys.Select, m.keys.Download, m.keys.Open, m.keys.Filter}},
			helpSection{title: "Queue", items: []key.Binding{m.keys.Commit, m.keys.Write, m.keys.Copy}},
		)
	case TabCommit:
		sections = append(sections,
			helpSection{title: "Queue", items: []key.Binding{m.keys.Write, m.keys.Copy, m.keys.Drop, m.keys.DropAll}},
		)
	}
	return sections
}

func (m *App) helpContent(width int) []string {
	sections := m.helpSections()
	if width < 100 {
		var lines []string
		for _, section := range sections {
			lines = append(lines, renderHelpSection(section, width)...)
		}
		return append(lines, m.helpLegendLines(width)...)
	}

	leftWidth := maxInt(1, (width-1)/2)
	rightWidth := maxInt(1, width-leftWidth-1)
	split := splitHelpSections(sections)
	left := flattenHelp(sections[:split], leftWidth)
	right := flattenHelp(sections[split:], rightWidth)
	right = append(right, m.helpLegendLines(rightWidth)...)
	height := maxInt(len(left), len(right))
	rows := make([]string, 0, height)
	for i := 0; i < height; i++ {
		l, r := "", ""
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		rows = append(rows, fitLine(l, leftWidth)+" "+fitLine(r, rightWidth))
	}
	return rows
}

func splitHelpSections(sections []helpSection) int {
	total := 0
	for _, section := range sections {
		total += 1 + len(section.items)
	}
	half := total / 2
	acc, split := 0, 0
	for i, section := range sections {
		if acc >= half {
			break
		}
		acc += 1 + len(section.items)
		split = i + 1
	}
	if split == 0 {
		split = 1
	}
	if split > len(sections) {
		split = len(sections)
	}
	return split
}

func flattenHelp(sections []helpSection, width int) []string {
	var lines []string
	for _, section := range sections {
		lines = append(lines, renderHelpSection(section, width)...)
	}
	return lines
}

func renderHelpSection(section helpSection, width int) []string {
	keyWidth := 0
	for _, binding := range section.items {
		help := binding.Help()
		if w := lipgloss.Width(help.Key); w > keyWidth {
			keyWidth = w
		}
	}
	keyWidth = minInt(keyWidth, 12)
	lines := []string{stFaint.Render(strings.ToUpper(section.title))}
	for _, binding := range section.items {
		help := binding.Help()
		line := "  " + stHelpKey.Render(pad(help.Key, keyWidth, false)) + "  " + stHelpTxt.Render(help.Desc)
		lines = append(lines, fitLine(line, width))
	}
	return append(lines, "")
}

func (m *App) helpLegendLines(width int) []string {
	entries := []struct {
		glyph string
		label string
		style lipgloss.Style
	}{
		{"○", "new", stFlagNew},
		{"◐", "tracked", stFlagT},
		{"◆", "ignored", stFlagI},
		{"×", "missed", stFlagM},
		{"!", "overdue", stOverdue},
		{"▪", "selected", stSelMark},
		{"▫", "partial", stSelMark},
	}
	lines := []string{stFaint.Render("LEGEND")}
	line := "  "
	for _, entry := range entries {
		cell := entry.style.Render(entry.glyph) + " " + stDim.Render(entry.label)
		if lipgloss.Width(line)+lipgloss.Width(cell)+2 > width && strings.TrimSpace(line) != "" {
			lines = append(lines, line)
			line = "  "
		}
		if strings.TrimSpace(line) != "" {
			line += "  "
		}
		line += cell
	}
	if strings.TrimSpace(line) != "" {
		lines = append(lines, line)
	}
	return append(lines, "")
}

// helpViewport is the body area the overlay can use at the current window size.
func (m *App) helpViewport() (int, int) {
	w, h := m.width, m.height
	if w <= 0 {
		w = 100
	}
	if h <= 0 {
		h = 30
	}
	if w < 40 || h < 10 {
		return maxInt(1, w), maxInt(1, h-1)
	}
	return maxInt(1, w), bodyHeightFor(w, h)
}

// tinyHelp keeps the help overlay reachable below the normal minimum. It uses
// the whole screen: one title line plus the windowed sections.
func (m *App) tinyHelp(w, h int) string {
	height := maxInt(1, h-1)
	rows, _ := m.helpLines(maxInt(1, w), height)
	lines := []string{" " + fitLine("help · ? close", maxInt(1, w-1))}
	for _, row := range rows {
		lines = append(lines, fitLine(row, w))
	}
	return strings.Join(lines, "\n")
}

func (m *App) helpMaxScroll() int {
	width, height := m.helpViewport()
	return maxInt(0, len(m.helpContent(width))-height)
}

// helpLines windows the help content; clipped reports whether scrolling is
// useful so the title can advertise it.
func (m *App) helpLines(width, height int) ([]string, bool) {
	content := m.helpContent(width)
	scroll := clampInt(m.helpScroll, 0, maxInt(0, len(content)-height))
	clipped := len(content) > height
	if height <= 0 {
		return nil, clipped
	}
	end := minInt(len(content), scroll+height)
	return content[scroll:end], clipped
}
