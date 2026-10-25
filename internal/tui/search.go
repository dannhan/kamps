package tui

import (
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ruyam/kamps/internal/model"
)

// searchActive reports whether the search query is being edited or still
// narrows the tree.
func (m *App) searchActive() bool { return m.searching || m.search != "" }

// searchTitle is the left side of the title row while a search is on. The bar
// cursor only blinks in while the query is still being typed.
func (m *App) searchTitle() string {
	title := "search · " + m.search
	if m.searching {
		title += "▌"
	}
	return title
}

// resetTreeCursor puts the tree back on its first row after the result set
// changes, so typing never leaves the cursor past the last match.
func (m *App) resetTreeCursor() {
	if m.tab == TabTasks {
		m.taskCursor = 0
		return
	}
	m.matCursor = 0
}

// clearSearch drops the query and reports whether there was one. Esc clears
// search before it falls back to clearing the state filter.
func (m *App) clearSearch() bool {
	if m.search == "" {
		return false
	}
	m.search = ""
	m.searching = false
	m.resetTreeCursor()
	m.toast = "search cleared"
	m.toastTicks = 20
	return true
}

// searchMatches is the shared text predicate: an empty query matches
// everything, otherwise any field may contain it, case-insensitively.
func (m *App) searchMatches(fields ...string) bool {
	query := strings.ToLower(strings.TrimSpace(m.search))
	if query == "" {
		return true
	}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return false
}

func (m *App) matchesTaskSearch(assignment model.Assignment) bool {
	if strings.TrimSpace(m.search) == "" {
		return true
	}
	course, _ := m.courseByID(assignment.CourseID)
	return m.searchMatches(assignment.Name, assignment.ID, assignment.Goal, assignment.Section, assignment.SectionName,
		course.Name, course.Code, course.Short)
}

func (m *App) matchesMaterialSearch(material model.Material) bool {
	if strings.TrimSpace(m.search) == "" {
		return true
	}
	course, _ := m.courseByID(material.CourseID)
	return m.searchMatches(material.Name, material.ID, material.Section, material.SectionName, material.Kind,
		material.CourseCode, course.Name, course.Code, course.Short)
}

// deleteLastSearchWord mirrors readline's ctrl+w: drop trailing whitespace and
// the word before it, leaving the space that separated it from the previous
// word. A query that is only whitespace becomes empty.
func deleteLastSearchWord(query string) string {
	runes := []rune(query)
	i := len(runes)
	for i > 0 && unicode.IsSpace(runes[i-1]) {
		i--
	}
	for i > 0 && !unicode.IsSpace(runes[i-1]) {
		i--
	}
	return string(runes[:i])
}

// handleSearchKey owns every key while the query is being typed: printable
// runes append, arrows still move the result cursor, enter keeps the query and
// esc throws it away.
func (m *App) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.searching = false
		m.search = ""
		m.resetTreeCursor()
	case "enter":
		m.searching = false
	case "backspace", "ctrl+h":
		runes := []rune(m.search)
		if len(runes) > 0 {
			m.search = string(runes[:len(runes)-1])
			m.resetTreeCursor()
		}
	case "ctrl+w", "alt+backspace", "ctrl+backspace":
		m.search = deleteLastSearchWord(m.search)
		m.resetTreeCursor()
	case "ctrl+u":
		m.search = ""
		m.resetTreeCursor()
	case "up", "ctrl+p":
		m.move(-1)
	case "down", "ctrl+n":
		m.move(1)
	case "tab":
		m.switchTab((m.tab + 1) % tabCount)
	case "shift+tab":
		m.switchTab((m.tab + tabCount - 1) % tabCount)
	default:
		if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace {
			m.search += string(msg.Runes)
			m.resetTreeCursor()
		}
	}
	return m, nil
}
