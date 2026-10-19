package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap is the single source of truth for keybindings. The footer hints are
// built by hand (see hints in frame.go); this drives the "?" overlay.
type KeyMap struct {
	Up, Down    key.Binding
	Left, Right key.Binding
	NextTab     key.Binding
	PrevTab     key.Binding
	Tabs        key.Binding
	Select      key.Binding
	Track       key.Binding
	Missed      key.Binding
	Ignore      key.Binding
	Undecide    key.Binding
	All         key.Binding
	Commit      key.Binding
	Layout      key.Binding
	Filter      key.Binding
	Sync        key.Binding
	Download    key.Binding
	Confirm     key.Binding
	Back        key.Binding
	Help        key.Binding
	Quit        key.Binding
}

// DefaultKeys returns the default bindings.
func DefaultKeys() KeyMap {
	return KeyMap{
		Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Left:     key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "left · fold")),
		Right:    key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "right · open")),
		NextTab:  key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next tab")),
		PrevTab:  key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("⇧tab", "prev tab")),
		Tabs:     key.NewBinding(key.WithKeys("1", "2", "3"), key.WithHelp("1-3", "tabs")),
		Select:   key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "select")),
		Track:    key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "track")),
		Missed:   key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "missed")),
		Ignore:   key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "ignore")),
		Undecide: key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "undecided")),
		All:      key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "select all")),
		Commit:   key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "review & commit")),
		Layout:   key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "view")),
		Filter:   key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Sync:     key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sync")),
		Download: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "download")),
		Confirm:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
		Back:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "more")),
		Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

// ShortHelp satisfies help.KeyMap.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.NextTab, k.Select, k.Track, k.Missed, k.Ignore, k.Commit, k.Layout, k.Help, k.Quit}
}

// FullHelp satisfies help.KeyMap (the "?" overlay).
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right, k.NextTab, k.PrevTab, k.Tabs, k.Layout},
		{k.Select, k.All, k.Track, k.Missed, k.Ignore, k.Undecide},
		{k.Commit, k.Filter, k.Sync, k.Download, k.Confirm, k.Back},
		{k.Help, k.Quit},
	}
}
