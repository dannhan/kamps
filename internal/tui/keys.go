package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap is the single source of truth for keybindings. It also feeds the
// help footer and the "?" overlay via the bubbles/help package.
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	NextTab  key.Binding
	PrevTab  key.Binding
	Select   key.Binding
	Toggle   key.Binding
	Download key.Binding
	Sync     key.Binding
	Back     key.Binding
	Help     key.Binding
	Quit     key.Binding
}

// DefaultKeys returns the default bindings.
func DefaultKeys() KeyMap {
	return KeyMap{
		Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		NextTab:  key.NewBinding(key.WithKeys("tab", "l"), key.WithHelp("tab", "section")),
		PrevTab:  key.NewBinding(key.WithKeys("shift+tab", "h"), key.WithHelp("⇧tab", "prev section")),
		Select:   key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "select")),
		Toggle:   key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "done")),
		Download: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "download")),
		Sync:     key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sync")),
		Back:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear/back")),
		Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

// ShortHelp satisfies help.KeyMap (footer).
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.NextTab, k.Select, k.Toggle, k.Download, k.Sync, k.Help, k.Quit}
}

// FullHelp satisfies help.KeyMap (the "?" overlay).
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.NextTab, k.PrevTab, k.Back},
		{k.Select, k.Toggle, k.Download, k.Sync, k.Help, k.Quit},
	}
}
