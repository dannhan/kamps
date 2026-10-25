package tui

// filters, cycled with f. `/` is reserved for search.
var filterNames = []string{"all", "undecided", "overdue", "tracked", "missed", "ignored"}

var materialFilterNames = []string{"all", "new", "downloaded", "link", "unknown"}

func (m *App) cycleFilter() {
	var names []string
	switch m.tab {
	case TabTasks:
		names = filterNames
	case TabMaterials:
		names = materialFilterNames
	default:
		return
	}
	cur := 0
	for i, f := range names {
		if f == m.filter {
			cur = i
		}
	}
	m.filter = names[(cur+1)%len(names)]
	m.tabFilters[m.tab] = m.filter
	if m.tab == TabTasks {
		m.taskCursor = 0
	} else {
		m.matCursor = 0
	}
	m.toast = "filter: " + m.filter
	m.toastTicks = 20
}

// clearFilter resets the active tab's filter to "all" and reports whether there
// was one to clear. Esc uses it before falling back to dropping selections, so
// a filter can be undone without cycling f all the way around.
func (m *App) clearFilter() bool {
	if m.tab != TabTasks && m.tab != TabMaterials {
		return false
	}
	if m.filter == "" || m.filter == "all" {
		return false
	}
	m.filter = "all"
	m.tabFilters[m.tab] = "all"
	if m.tab == TabTasks {
		m.taskCursor = 0
	} else {
		m.matCursor = 0
	}
	m.toast = "filter: all"
	m.toastTicks = 20
	return true
}
