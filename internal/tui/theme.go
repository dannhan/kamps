package tui

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type themeColors struct {
	Bg       lipgloss.AdaptiveColor
	Surface  lipgloss.AdaptiveColor
	Marked   lipgloss.AdaptiveColor
	Overlay  lipgloss.AdaptiveColor
	Border   lipgloss.AdaptiveColor
	Bright   lipgloss.AdaptiveColor
	Primary  lipgloss.AdaptiveColor
	Dim      lipgloss.AdaptiveColor
	Subtle   lipgloss.AdaptiveColor
	TreeLine lipgloss.AdaptiveColor
	Accent   lipgloss.AdaptiveColor
	Accent2  lipgloss.AdaptiveColor
	New      lipgloss.AdaptiveColor
	Tracked  lipgloss.AdaptiveColor
	Done     lipgloss.AdaptiveColor
	Ignored  lipgloss.AdaptiveColor
	Missed   lipgloss.AdaptiveColor
}

var themeTokens = themeColors{
	Bg:      lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#1B1B26"},
	Surface: lipgloss.AdaptiveColor{Light: "#E4E4E4", Dark: "#333355"},
	Marked:  lipgloss.AdaptiveColor{Light: "#E2E2F4", Dark: "#413A66"},
	Overlay: lipgloss.AdaptiveColor{Light: "#F2F2F2", Dark: "#292940"},
	Border:  lipgloss.AdaptiveColor{Light: "#C8C8D0", Dark: "#3B3E5E"},
	Bright:  lipgloss.AdaptiveColor{Light: "#1B1B26", Dark: "#EDEDED"},
	Primary: lipgloss.AdaptiveColor{Light: "#1B1B26", Dark: "#EDEDED"},
	Dim:     lipgloss.AdaptiveColor{Light: "#5C5C5C", Dark: "#C6C6C6"},
	Subtle:  lipgloss.AdaptiveColor{Light: "#8A8A8A", Dark: "#8A8A8A"},
	TreeLine: lipgloss.AdaptiveColor{
		Light: quietHex("#1B1B26", "#E4E4E4"),
		Dark:  quietHex("#EDEDED", "#333355"),
	},
	Accent:  lipgloss.AdaptiveColor{Light: "#4B4FA0", Dark: "#8288B8"},
	Accent2: lipgloss.AdaptiveColor{Light: "#4B4FA0", Dark: "#8288B8"},
	New:     lipgloss.AdaptiveColor{Light: "#8A8A8A", Dark: "#8A8A8A"},
	Tracked: lipgloss.AdaptiveColor{Light: "#A5680F", Dark: "#E0AF68"},
	Done:    lipgloss.AdaptiveColor{Light: "#1F8A4C", Dark: "#6FCF97"},
	Ignored: lipgloss.AdaptiveColor{Light: "#7651A8", Dark: "#B59AE0"},
	Missed:  lipgloss.AdaptiveColor{Light: "#C0392B", Dark: "#F7768E"},
}

var courseAccents = []lipgloss.AdaptiveColor{
	{Light: "#3F64A8", Dark: "#8DB4E2"},
	{Light: "#1F7A6A", Dark: "#6FD0BE"},
	{Light: "#A5680F", Dark: "#E0AF68"},
	{Light: "#7651A8", Dark: "#B59AE0"},
	{Light: "#A63D5C", Dark: "#F08AA8"},
	{Light: "#3E7D3E", Dark: "#8FD08F"},
}

func quietHex(foreground, background string) string {
	parse := func(value string) ([3]float64, bool) {
		var rgb [3]float64
		value = strings.TrimPrefix(value, "#")
		if len(value) != 6 {
			return rgb, false
		}
		for i := range rgb {
			component, err := strconv.ParseUint(value[i*2:i*2+2], 16, 8)
			if err != nil {
				return rgb, false
			}
			v := float64(component) / 255
			if v <= 0.04045 {
				rgb[i] = v / 12.92
			} else {
				rgb[i] = math.Pow((v+0.055)/1.055, 2.4)
			}
		}
		return rgb, true
	}
	fg, okFG := parse(foreground)
	bg, okBG := parse(background)
	if !okFG || !okBG {
		return "#8A8A8A"
	}
	var value strings.Builder
	value.WriteByte('#')
	for i := range fg {
		linear := bg[i] + (fg[i]-bg[i])*0.45
		var srgb float64
		if linear <= 0.0031308 {
			srgb = linear * 12.92
		} else {
			srgb = 1.055*math.Pow(linear, 1/2.4) - 0.055
		}
		component := int(srgb*255 + 0.5)
		value.WriteString(fmt.Sprintf("%02X", component))
	}
	return value.String()
}

var (
	stPrimary  = lipgloss.NewStyle().Foreground(themeTokens.Primary)
	stDim      = lipgloss.NewStyle().Foreground(themeTokens.Dim)
	stTreeLine = lipgloss.NewStyle().Foreground(themeTokens.TreeLine)
	stFaint    = lipgloss.NewStyle().Foreground(themeTokens.Subtle)
	stEmpty    = lipgloss.NewStyle().Foreground(themeTokens.Subtle).Italic(true)
	stHelpKey  = lipgloss.NewStyle().Foreground(themeTokens.Bright)
	stHelpTxt  = lipgloss.NewStyle().Foreground(themeTokens.Subtle)
	stHelpSep  = lipgloss.NewStyle().Foreground(themeTokens.Border)

	stRow     = stPrimary
	stRowDim  = lipgloss.NewStyle().Foreground(themeTokens.Subtle)
	stCursor  = lipgloss.NewStyle().Foreground(themeTokens.Accent)
	stSelMark = lipgloss.NewStyle().Foreground(themeTokens.Accent)

	stDone    = lipgloss.NewStyle().Foreground(themeTokens.Done)
	stSoon    = lipgloss.NewStyle().Foreground(themeTokens.Dim)
	stOverdue = lipgloss.NewStyle().Foreground(themeTokens.Missed)
	stCode    = lipgloss.NewStyle().Foreground(themeTokens.Accent)

	stPaneTitle = lipgloss.NewStyle().Foreground(themeTokens.Bright)
	stRule      = lipgloss.NewStyle().Foreground(themeTokens.Border)
	stKey       = lipgloss.NewStyle().Foreground(themeTokens.Accent)

	stFlagNew = lipgloss.NewStyle().Foreground(themeTokens.New)
	stFlagT   = lipgloss.NewStyle().Foreground(themeTokens.Tracked)
	stFlagM   = lipgloss.NewStyle().Foreground(themeTokens.Missed)
	stFlagI   = lipgloss.NewStyle().Foreground(themeTokens.Ignored)
)
