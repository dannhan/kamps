package tui

import "github.com/charmbracelet/lipgloss"

// "Sunset amber": warm amber accents over warm neutrals.
// Deliberately low-saturation borders so the amber reads as the one loud thing.
var (
	colAccent  = lipgloss.Color("#ffb454") // amber — primary accent
	colAccent2 = lipgloss.Color("#ff7b72") // coral — secondary
	colGreen   = lipgloss.Color("#8ec07c") // muted green
	colYellow  = lipgloss.Color("#fabd2f") // warm yellow
	colRed     = lipgloss.Color("#ea6962") // soft red
	colMuted   = lipgloss.Color("#928374") // warm grey
	colFg      = lipgloss.Color("#ebdbb2") // warm cream
	colDim     = lipgloss.Color("#bdae93")
	colBorder  = lipgloss.Color("#5b5148")
	colSurface = lipgloss.Color("#3c3836")
	colBarBG   = lipgloss.Color("#282828")
)

var (
	// header / chrome
	stHeader    = lipgloss.NewStyle().Foreground(colAccent).Bold(true)
	stHeaderDim = lipgloss.NewStyle().Foreground(colMuted)
	stRule      = lipgloss.NewStyle().Foreground(colBorder)
	stFooter    = lipgloss.NewStyle().Foreground(colMuted)
	stEmpty     = lipgloss.NewStyle().Foreground(colMuted).Italic(true)
	stTitle     = lipgloss.NewStyle().Foreground(colAccent).Bold(true)
	stToast     = lipgloss.NewStyle().Foreground(colBarBG).Background(colGreen).Bold(true).Padding(0, 1)
	stHelpBox   = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colAccent).
			Padding(1, 2)

	// tabs
	stTabActive = lipgloss.NewStyle().Foreground(colBarBG).Background(colAccent).Bold(true).Padding(0, 1)
	stTabIdle   = lipgloss.NewStyle().Foreground(colDim).Padding(0, 1)
	stTabOn     = lipgloss.NewStyle().Foreground(colAccent).Bold(true)
	stTabOff    = lipgloss.NewStyle().Foreground(colMuted)

	// list rows
	stRow          = lipgloss.NewStyle().Foreground(colFg)
	stRowDim       = lipgloss.NewStyle().Foreground(colMuted)
	stRowCursor    = lipgloss.NewStyle().Foreground(colBarBG).Background(colAccent).Bold(true)
	stCursorGutter = lipgloss.NewStyle().Foreground(colAccent).Bold(true)

	// semantic
	stDone    = lipgloss.NewStyle().Foreground(colGreen)
	stDoing   = lipgloss.NewStyle().Foreground(colAccent)
	stTodo    = lipgloss.NewStyle().Foreground(colDim)
	stIgnored = lipgloss.NewStyle().Foreground(colMuted)
	stOverdue = lipgloss.NewStyle().Foreground(colRed).Bold(true)
	stSoon    = lipgloss.NewStyle().Foreground(colYellow)
	stCode    = lipgloss.NewStyle().Foreground(colAccent2)
	stTag     = lipgloss.NewStyle().Foreground(colMuted).Italic(true)

	// boxes / frames
	stBorder  = lipgloss.NewStyle().Foreground(colBorder)
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
