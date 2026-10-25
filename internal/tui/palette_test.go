package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func typeRunes(app *App, text string) {
	for _, r := range text {
		app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

func TestPaletteOpensFiltersAndExecutes(t *testing.T) {
	app := demoApp(t)
	app.tab = TabTasks
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	if !app.paletteOn {
		t.Fatal(": did not open the palette")
	}
	typeRunes(app, "overdue")
	app.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if app.paletteOn {
		t.Fatal("enter did not close the palette")
	}
	if app.filter != "overdue" {
		t.Fatalf("filter = %q", app.filter)
	}
}

func TestPaletteRunsHelp(t *testing.T) {
	app := demoApp(t)
	app.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	typeRunes(app, "help")
	app.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !app.helpOn {
		t.Fatal("palette did not open help")
	}
}

func TestPaletteRendersAsACommandScreen(t *testing.T) {
	app := demoApp(t)
	app.paletteOn = true
	view := stripANSI(app.View())
	if !strings.Contains(view, "commands") || !strings.Contains(view, "type to filter") || !strings.Contains(view, "sync now") {
		t.Fatalf("palette view = %q", view)
	}
}
