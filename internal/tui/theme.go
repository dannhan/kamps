package tui

import "github.com/charmbracelet/lipgloss"

// The default Charm palette.
//
// These are the colours the Bubble Tea / Lip Gloss / Bubbles examples ship with:
// ANSI-256 greys for structure plus Charm's purple and pink for accents. They are
// adaptive, so the whole app stays readable whether your terminal is light or
// dark — nothing here is a bespoke brand colour.
//
//	greys    252 body · 249 secondary · 243 muted · 240 structure  (dark terminal)
//	accents  62 purple (frame) · 212 pink (cursor/keys) · 170 plum (codes)
//	meaning  42 green (tracked) · 214 orange (due soon) · 203 red (missed)
var (
	colFg      = lipgloss.AdaptiveColor{Light: "234", Dark: "252"} // body text
	colDim     = lipgloss.AdaptiveColor{Light: "238", Dark: "248"} // secondary text
	colMuted   = lipgloss.AdaptiveColor{Light: "243", Dark: "243"} // tertiary / structure
	colBorder  = lipgloss.AdaptiveColor{Light: "250", Dark: "240"} // inner panel borders
	colFrame   = lipgloss.AdaptiveColor{Light: "62", Dark: "62"}   // outer frame — Charm purple
	colAccent  = lipgloss.AdaptiveColor{Light: "162", Dark: "212"} // pink — cursor, keys, emphasis
	colAccent2 = lipgloss.AdaptiveColor{Light: "128", Dark: "170"} // plum — course codes, tags
	colGreen   = lipgloss.AdaptiveColor{Light: "28", Dark: "42"}   // tracked / ok
	colYellow  = lipgloss.AdaptiveColor{Light: "166", Dark: "214"} // due soon / warning
	colRed     = lipgloss.AdaptiveColor{Light: "160", Dark: "203"} // missed / overdue
	colSurface = lipgloss.AdaptiveColor{Light: "254", Dark: "236"}
	colBarBG   = lipgloss.AdaptiveColor{Light: "252", Dark: "235"}
)

// All text styles live here. Views compose these; they never call
// lipgloss.NewStyle() themselves (see AGENTS.md).
var (
	// chrome / header
	stHeader    = lipgloss.NewStyle().Foreground(colAccent2).Bold(true)
	stHeaderDim = lipgloss.NewStyle().Foreground(colMuted).Bold(true)
	stEmpty     = lipgloss.NewStyle().Foreground(colMuted).Italic(true)
	stTitle     = lipgloss.NewStyle().Foreground(colAccent).Bold(true)
	stToast     = lipgloss.NewStyle().Foreground(colBarBG).Background(colGreen).Bold(true).Padding(0, 1)
	stHelpBox   = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colFrame).
			Padding(1, 2)

	// tabs
	stTabOn  = lipgloss.NewStyle().Foreground(colAccent).Bold(true)
	stTabOff = lipgloss.NewStyle().Foreground(colMuted)

	// list rows
	stRow          = lipgloss.NewStyle().Foreground(colFg)
	stRowDim       = lipgloss.NewStyle().Foreground(colMuted)
	stRowCursor    = lipgloss.NewStyle().Foreground(colBarBG).Background(colAccent).Bold(true)
	stCursorGutter = lipgloss.NewStyle().Foreground(colAccent).Bold(true)

	// meaning
	stDone    = lipgloss.NewStyle().Foreground(colGreen)
	stIgnored = lipgloss.NewStyle().Foreground(colMuted)
	stOverdue = lipgloss.NewStyle().Foreground(colRed).Bold(true)
	stSoon    = lipgloss.NewStyle().Foreground(colYellow)
	stCode    = lipgloss.NewStyle().Foreground(colAccent2)

	// boxes / frames
	stFrame   = lipgloss.NewStyle().Foreground(colFrame)  // outer frame
	stBorder  = lipgloss.NewStyle().Foreground(colBorder) // inner panels
	stBoxTtl  = lipgloss.NewStyle().Foreground(colAccent).Bold(true)
	stLabel   = lipgloss.NewStyle().Foreground(colMuted)
	stValue   = lipgloss.NewStyle().Foreground(colFg)
	stKey     = lipgloss.NewStyle().Foreground(colAccent)
	stSection = lipgloss.NewStyle().Foreground(colAccent).Bold(true) // ◆ active step
	stStep    = lipgloss.NewStyle().Foreground(colMuted)             // ◇ pending step
	stChipOn  = lipgloss.NewStyle().Foreground(colBarBG).Background(colAccent).Bold(true).Padding(0, 1)
	stChipOff = lipgloss.NewStyle().Foreground(colMuted)

	// triage flags
	stFlagT = lipgloss.NewStyle().Foreground(colGreen).Bold(true) // track
	stFlagM = lipgloss.NewStyle().Foreground(colRed).Bold(true)   // missed
	stFlagI = lipgloss.NewStyle().Foreground(colMuted).Bold(true) // ignore

	// selection markers
	stCheckOn  = lipgloss.NewStyle().Foreground(colAccent)
	stCheckOff = lipgloss.NewStyle().Foreground(colMuted)
)
