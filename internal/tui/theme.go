package tui

import "github.com/charmbracelet/lipgloss"

// The palette is gh-dash's.
//
// gh-dash keeps its chrome quiet: near-white primary text, two greys for
// de-emphasis, and borders sitting a shade off the background so content is the
// only thing that stands out. Real colour is reserved for status. Everything is
// adaptive, so it holds up on a light or dark terminal.
//
// The greys are deliberately neutral. A tinted near-white reads better on
// paper, but a 16-colour terminal quantises it to magenta, and it is not worth
// the app turning purple for anyone on a basic terminal.
var (
	colText     = lipgloss.AdaptiveColor{Light: "#1B1B26", Dark: "#EDEDED"} // primary text
	colTextDim  = lipgloss.AdaptiveColor{Light: "#5C5C5C", Dark: "#C6C6C6"} // secondary
	colFaint    = lipgloss.AdaptiveColor{Light: "#8A8A8A", Dark: "#8A8A8A"} // faint
	colSelBg    = lipgloss.AdaptiveColor{Light: "#E4E4E4", Dark: "#333355"} // cursor band
	colBorder   = lipgloss.AdaptiveColor{Light: "#C8C8D0", Dark: "#3B3E5E"} // unfocused pane
	colBorderHi = lipgloss.AdaptiveColor{Light: "#5B60A0", Dark: "#5B60A0"} // focused pane
	colAccent   = lipgloss.AdaptiveColor{Light: "#4B4FA0", Dark: "#8288B8"} // markers, ids
	colSuccess  = lipgloss.AdaptiveColor{Light: "#1F8A4C", Dark: "#6FCF97"}
	colWarn     = lipgloss.AdaptiveColor{Light: "#A5680F", Dark: "#E0AF68"}
	colError    = lipgloss.AdaptiveColor{Light: "#C0392B", Dark: "#F7768E"}
)

// All text styles live here. Views compose these; they never call
// lipgloss.NewStyle() themselves (see AGENTS.md).
var (
	// text hierarchy
	stFaint   = lipgloss.NewStyle().Foreground(colFaint)
	stTitle   = lipgloss.NewStyle().Foreground(colText).Bold(true)
	stEmpty   = lipgloss.NewStyle().Foreground(colFaint).Italic(true)
	stHelpBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colBorder).
			Padding(1, 2)

	// the top bar: the active tab is a filled pill, the way gh-dash marks the
	// view you are in
	stTabOn  = lipgloss.NewStyle().Foreground(colText).Background(colSelBg).Padding(0, 1)
	stTabOff = lipgloss.NewStyle().Foreground(colFaint).Padding(0, 1)

	// the bottom help bar
	stHelpKey = lipgloss.NewStyle().Foreground(colText)
	stHelpTxt = lipgloss.NewStyle().Foreground(colFaint)
	stHelpSep = lipgloss.NewStyle().Foreground(colBorder)

	// rows. The cursor is a background band; a space-selected row gets a marker,
	// so "where the keyboard is" and "what I queued" stay distinguishable.
	stRow     = lipgloss.NewStyle().Foreground(colText)
	stRowDim  = lipgloss.NewStyle().Foreground(colFaint)
	stCursor  = lipgloss.NewStyle().Foreground(colAccent)
	stSelMark = lipgloss.NewStyle().Foreground(colAccent)

	// meaning
	stDone    = lipgloss.NewStyle().Foreground(colSuccess)
	stSoon    = lipgloss.NewStyle().Foreground(colWarn)
	stOverdue = lipgloss.NewStyle().Foreground(colError)
	stIgnored = lipgloss.NewStyle().Foreground(colFaint)
	stCode    = lipgloss.NewStyle().Foreground(colAccent)

	// panes
	stBorder    = lipgloss.NewStyle().Foreground(colBorder)
	stBorderHi  = lipgloss.NewStyle().Foreground(colBorderHi)
	stPaneTitle = lipgloss.NewStyle().Foreground(colText)
	stRule      = lipgloss.NewStyle().Foreground(colBorder)
	stLabel     = lipgloss.NewStyle().Foreground(colTextDim)
	stValue     = lipgloss.NewStyle().Foreground(colText)
	stKey       = lipgloss.NewStyle().Foreground(colAccent)
	stSection   = lipgloss.NewStyle().Foreground(colText) // ◆ active step
	stStep      = lipgloss.NewStyle().Foreground(colFaint)
	stBarEmpty  = lipgloss.NewStyle().Foreground(colBorder)

	// triage flags
	stFlagT = lipgloss.NewStyle().Foreground(colSuccess)
	stFlagM = lipgloss.NewStyle().Foreground(colError)
	stFlagI = lipgloss.NewStyle().Foreground(colFaint)
)
