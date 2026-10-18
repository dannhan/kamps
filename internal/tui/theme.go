package tui

import "github.com/charmbracelet/lipgloss"

// A clean, dark palette with one warm accent. Roughly Tokyo-Night flavoured:
// quiet background, readable foreground, a few cheerful highlight colours.
var (
	colAccent  = lipgloss.Color("#7dcfff") // cyan — primary accent
	colPink    = lipgloss.Color("#f5c2e7") // cheerful secondary
	colGreen   = lipgloss.Color("#9ece6a") // done / success
	colAmber   = lipgloss.Color("#e0af68") // due soon / warning
	colRed     = lipgloss.Color("#f7768e") // overdue
	colPurple  = lipgloss.Color("#bb9af7")
	colMuted   = lipgloss.Color("#565f89")
	colFg      = lipgloss.Color("#c0caf5")
	colDim     = lipgloss.Color("#9aa5ce")
	colBarBG   = lipgloss.Color("#1a1b26")
	colSurface = lipgloss.Color("#292e42")
)

var (
	// header bar
	stHeader = lipgloss.NewStyle().
			Foreground(colAccent).Bold(true)
	stHeaderDim = lipgloss.NewStyle().Foreground(colMuted)

	// tab bar
	stTabActive = lipgloss.NewStyle().
			Foreground(colBarBG).Background(colAccent).Bold(true).Padding(0, 1)
	stTabIdle = lipgloss.NewStyle().
			Foreground(colDim).Padding(0, 1)

	// list rows
	stRow       = lipgloss.NewStyle().Foreground(colFg)
	stRowDim    = lipgloss.NewStyle().Foreground(colMuted)
	stRowCursor = lipgloss.NewStyle().
			Foreground(colBarBG).Background(colPink).Bold(true)
	stCursorGutter = lipgloss.NewStyle().Foreground(colPink).Bold(true)

	// semantic
	stDone    = lipgloss.NewStyle().Foreground(colGreen)
	stDoing   = lipgloss.NewStyle().Foreground(colAccent)
	stTodo    = lipgloss.NewStyle().Foreground(colDim)
	stIgnored = lipgloss.NewStyle().Foreground(colMuted)
	stOverdue = lipgloss.NewStyle().Foreground(colRed).Bold(true)
	stSoon    = lipgloss.NewStyle().Foreground(colAmber)
	stCode    = lipgloss.NewStyle().Foreground(colPurple)
	stTag     = lipgloss.NewStyle().Foreground(colMuted).Italic(true)

	// chrome
	stRule   = lipgloss.NewStyle().Foreground(colSurface)
	stFooter = lipgloss.NewStyle().Foreground(colMuted)
	stToast  = lipgloss.NewStyle().
			Foreground(colBarBG).Background(colGreen).Bold(true).Padding(0, 1)
	stHelpBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colAccent).
			Padding(1, 2)
	stEmpty = lipgloss.NewStyle().Foreground(colMuted).Italic(true)
	stTitle = lipgloss.NewStyle().Foreground(colPink).Bold(true)
)
