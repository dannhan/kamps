package tui

import "github.com/charmbracelet/lipgloss"

type themeColors struct {
	Bg      lipgloss.AdaptiveColor
	Surface lipgloss.AdaptiveColor
	Overlay lipgloss.AdaptiveColor
	Border  lipgloss.AdaptiveColor
	Bright  lipgloss.AdaptiveColor
	Text    lipgloss.AdaptiveColor
	Dim     lipgloss.AdaptiveColor
	Subtle  lipgloss.AdaptiveColor
	Accent  lipgloss.AdaptiveColor
	Accent2 lipgloss.AdaptiveColor
	New     lipgloss.AdaptiveColor
	Tracked lipgloss.AdaptiveColor
	Done    lipgloss.AdaptiveColor
	Ignored lipgloss.AdaptiveColor
	Missed  lipgloss.AdaptiveColor
}

var themeTokens = themeColors{
	Bg:      lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#1B1B26"},
	Surface: lipgloss.AdaptiveColor{Light: "#E4E4E4", Dark: "#333355"},
	Overlay: lipgloss.AdaptiveColor{Light: "#F2F2F2", Dark: "#292940"},
	Border:  lipgloss.AdaptiveColor{Light: "#C8C8D0", Dark: "#3B3E5E"},
	Bright:  lipgloss.AdaptiveColor{Light: "#1B1B26", Dark: "#EDEDED"},
	Text:    lipgloss.AdaptiveColor{Light: "#1B1B26", Dark: "#EDEDED"},
	Dim:     lipgloss.AdaptiveColor{Light: "#5C5C5C", Dark: "#C6C6C6"},
	Subtle:  lipgloss.AdaptiveColor{Light: "#8A8A8A", Dark: "#8A8A8A"},
	Accent:  lipgloss.AdaptiveColor{Light: "#4B4FA0", Dark: "#8288B8"},
	Accent2: lipgloss.AdaptiveColor{Light: "#4B4FA0", Dark: "#8288B8"},
	New:     lipgloss.AdaptiveColor{Light: "#8A8A8A", Dark: "#8A8A8A"},
	Tracked: lipgloss.AdaptiveColor{Light: "#A5680F", Dark: "#E0AF68"},
	Done:    lipgloss.AdaptiveColor{Light: "#1F8A4C", Dark: "#6FCF97"},
	Ignored: lipgloss.AdaptiveColor{Light: "#7651A8", Dark: "#B59AE0"},
	Missed:  lipgloss.AdaptiveColor{Light: "#C0392B", Dark: "#F7768E"},
}

var (
	stFaint   = lipgloss.NewStyle().Foreground(themeTokens.Subtle)
	stTitle   = lipgloss.NewStyle().Foreground(themeTokens.Bright).Bold(true)
	stEmpty   = lipgloss.NewStyle().Foreground(themeTokens.Subtle).Italic(true)
	stHelpBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(themeTokens.Border).
			Padding(1, 2)

	stTabOn  = lipgloss.NewStyle().Foreground(themeTokens.Bright).Background(themeTokens.Surface).Padding(0, 1)
	stTabOff = lipgloss.NewStyle().Foreground(themeTokens.Subtle).Padding(0, 1)

	stHelpKey = lipgloss.NewStyle().Foreground(themeTokens.Bright)
	stHelpTxt = lipgloss.NewStyle().Foreground(themeTokens.Subtle)
	stHelpSep = lipgloss.NewStyle().Foreground(themeTokens.Border)

	stRow     = lipgloss.NewStyle().Foreground(themeTokens.Text)
	stRowDim  = lipgloss.NewStyle().Foreground(themeTokens.Subtle)
	stCursor  = lipgloss.NewStyle().Foreground(themeTokens.Accent)
	stSelMark = lipgloss.NewStyle().Foreground(themeTokens.Accent)

	stDone    = lipgloss.NewStyle().Foreground(themeTokens.Done)
	stSoon    = lipgloss.NewStyle().Foreground(themeTokens.Dim)
	stOverdue = lipgloss.NewStyle().Foreground(themeTokens.Missed)
	stIgnored = lipgloss.NewStyle().Foreground(themeTokens.Ignored)
	stCode    = lipgloss.NewStyle().Foreground(themeTokens.Accent)

	stBorder    = lipgloss.NewStyle().Foreground(themeTokens.Border)
	stBorderHi  = lipgloss.NewStyle().Foreground(themeTokens.Accent)
	stPaneTitle = lipgloss.NewStyle().Foreground(themeTokens.Bright)
	stRule      = lipgloss.NewStyle().Foreground(themeTokens.Border)
	stLabel     = lipgloss.NewStyle().Foreground(themeTokens.Dim)
	stValue     = lipgloss.NewStyle().Foreground(themeTokens.Text)
	stKey       = lipgloss.NewStyle().Foreground(themeTokens.Accent)
	stSection   = lipgloss.NewStyle().Foreground(themeTokens.Bright)
	stStep      = lipgloss.NewStyle().Foreground(themeTokens.Subtle)
	stBarEmpty  = lipgloss.NewStyle().Foreground(themeTokens.Border)

	stFlagNew = lipgloss.NewStyle().Foreground(themeTokens.New)
	stFlagT   = lipgloss.NewStyle().Foreground(themeTokens.Tracked)
	stFlagM   = lipgloss.NewStyle().Foreground(themeTokens.Missed)
	stFlagI   = lipgloss.NewStyle().Foreground(themeTokens.Ignored)
)
