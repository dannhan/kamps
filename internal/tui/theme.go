package tui

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/ruyam/kamps/internal/model"
)

type themeMode string

const (
	themeAuto  themeMode = "auto"
	themeDark  themeMode = "dark"
	themeLight themeMode = "light"
	themeMono  themeMode = "mono"
)

// themeColors holds one resolved palette. Every field is a concrete color so a
// theme override and monochrome mode behave the same on every terminal. An
// empty color means "leave the terminal default alone" (mono).
type themeColors struct {
	Surface  lipgloss.Color
	Marked   lipgloss.Color
	Border   lipgloss.Color
	Bright   lipgloss.Color
	Primary  lipgloss.Color
	Dim      lipgloss.Color
	Subtle   lipgloss.Color
	Guide    lipgloss.Color
	TreeLine lipgloss.Color
	Accent   lipgloss.Color

	New     lipgloss.Color
	Tracked lipgloss.Color
	Done    lipgloss.Color
	Ignored lipgloss.Color
	Missed  lipgloss.Color

	Mono bool
}

func darkColors() themeColors {
	return themeColors{
		Surface:  "#3A3A5C",
		Marked:   "#4D4478",
		Border:   "#3B3E5E",
		Bright:   "#EDEDED",
		Primary:  "#EDEDED",
		Dim:      "#C6C6C6",
		Subtle:   "#9A9AA4",
		Guide:    "#8A8A8A",
		TreeLine: lipgloss.Color(quietHex("#EDEDED", "#1B1B26")),
		Accent:   "#9AA0D0",
		New:      "#9A9AA4",
		Tracked:  "#E0AF68",
		Done:     "#6FCF97",
		Ignored:  "#B59AE0",
		Missed:   "#F7768E",
	}
}

func lightColors() themeColors {
	return themeColors{
		Surface:  "#D4D4DE",
		Marked:   "#CFCFE8",
		Border:   "#C8C8D0",
		Bright:   "#1B1B26",
		Primary:  "#1B1B26",
		Dim:      "#55555E",
		Subtle:   "#6B6B75",
		Guide:    "#8A8A8A",
		TreeLine: lipgloss.Color(quietHex("#1B1B26", "#FFFFFF")),
		Accent:   "#3F4390",
		New:      "#6B6B75",
		Tracked:  "#96590B",
		Done:     "#177A43",
		Ignored:  "#7651A8",
		Missed:   "#C0392B",
	}
}

func monoColors() themeColors {
	return themeColors{Mono: true}
}

// themeTokens is the active palette. applyTheme replaces it once before the
// first render; widgets read styles, never hex values, after that.
var themeTokens = darkColors()

var courseAccents = darkCourseAccents()

func darkCourseAccents() []lipgloss.Color {
	return []lipgloss.Color{"#8DB4E2", "#6FD0BE", "#E0AF68", "#B59AE0", "#F08AA8", "#8FD08F"}
}

func lightCourseAccents() []lipgloss.Color {
	return []lipgloss.Color{"#3F64A8", "#1F7A6A", "#A63D5C", "#7651A8", "#A5680F", "#3E7D3E"}
}

func normalizeTheme(value string) themeMode {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "dark":
		return themeDark
	case "light":
		return themeLight
	case "mono":
		return themeMono
	default:
		return themeAuto
	}
}

// resolveTheme turns a configured value into the palette actually used. Auto
// follows the terminal background; NO_COLOR always wins.
func resolveTheme(value string) themeMode {
	mode := normalizeTheme(value)
	if termenv.EnvNoColor() {
		return themeMono
	}
	if mode != themeAuto {
		return mode
	}
	if lipgloss.HasDarkBackground() {
		return themeDark
	}
	return themeLight
}

// applyTheme resolves and installs the palette. It is called once from New so
// tests and the running app see the same styles.
func applyTheme(value string) {
	mode := resolveTheme(value)
	themeTokens = colorsFor(mode)
	switch {
	case themeTokens.Mono:
		courseAccents = nil
	case mode == themeLight:
		courseAccents = lightCourseAccents()
	default:
		courseAccents = darkCourseAccents()
	}
	buildStyles()
}

func colorsFor(mode themeMode) themeColors {
	switch mode {
	case themeLight:
		return lightColors()
	case themeMono:
		return monoColors()
	default:
		return darkColors()
	}
}

// Styles. These are the only places a color is applied to a widget.
var (
	stPrimary  lipgloss.Style
	stDim      lipgloss.Style
	stGuide    lipgloss.Style
	stTreeLine lipgloss.Style
	stFaint    lipgloss.Style
	stEmpty    lipgloss.Style
	stHelpKey  lipgloss.Style
	stHelpTxt  lipgloss.Style
	stHelpSep  lipgloss.Style

	stRow     lipgloss.Style
	stRowDim  lipgloss.Style
	stCursor  lipgloss.Style
	stSelMark lipgloss.Style

	stDone    lipgloss.Style
	stSoon    lipgloss.Style
	stOverdue lipgloss.Style
	stCode    lipgloss.Style

	stPaneTitle lipgloss.Style
	stRule      lipgloss.Style
	stKey       lipgloss.Style

	stFlagNew lipgloss.Style
	stFlagT   lipgloss.Style
	stFlagM   lipgloss.Style
	stFlagI   lipgloss.Style
)

func buildStyles() {
	stPrimary = fgStyle(themeTokens.Primary)
	stDim = fgStyle(themeTokens.Dim)
	stGuide = fgStyle(themeTokens.Guide)
	stTreeLine = fgStyle(themeTokens.TreeLine)
	stFaint = fgStyle(themeTokens.Subtle)
	stEmpty = fgStyle(themeTokens.Subtle).Italic(true)
	stHelpKey = fgStyle(themeTokens.Bright)
	stHelpTxt = fgStyle(themeTokens.Subtle)
	stHelpSep = fgStyle(themeTokens.Border)

	stRow = stPrimary
	stRowDim = fgStyle(themeTokens.Subtle)
	stCursor = fgStyle(themeTokens.Accent)
	stSelMark = fgStyle(themeTokens.Accent)

	stDone = fgStyle(themeTokens.Done)
	stSoon = fgStyle(themeTokens.Dim)
	stOverdue = fgStyle(themeTokens.Missed).Bold(true)
	stCode = fgStyle(themeTokens.Accent)

	stPaneTitle = fgStyle(themeTokens.Bright)
	stRule = fgStyle(themeTokens.Border)
	stKey = fgStyle(themeTokens.Accent)

	stFlagNew = fgStyle(themeTokens.New)
	stFlagT = fgStyle(themeTokens.Tracked)
	stFlagM = fgStyle(themeTokens.Missed)
	stFlagI = fgStyle(themeTokens.Ignored)
}

// fgStyle leaves the color alone in mono mode.
func fgStyle(color lipgloss.Color) lipgloss.Style {
	if color == "" {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(color)
}

// bandStyle paints a background band, except in mono mode where the band would
// be noise and the gutter/mark glyph carries the state.
func bandStyle(style lipgloss.Style, color lipgloss.Color) lipgloss.Style {
	if color == "" {
		return style
	}
	return style.Background(color)
}

// tabCellStyle is the tab strip: the active tab is bold and underlined, not a
// filled block, so it cannot mismatch the user's terminal background.
func tabCellStyle(active bool) lipgloss.Style {
	if active {
		return fgStyle(themeTokens.Accent).Bold(true).Underline(true)
	}
	return fgStyle(themeTokens.Subtle)
}

func courseStyle(courseID int) lipgloss.Style {
	if len(courseAccents) == 0 {
		return lipgloss.NewStyle()
	}
	index := courseID % len(courseAccents)
	if index < 0 {
		index = -index
	}
	return fgStyle(courseAccents[index])
}

func triageStyle(t model.Triage) lipgloss.Style {
	switch t {
	case model.Tracked:
		return stFlagT
	case model.Missed:
		return stFlagM
	case model.Ignored:
		return stFlagI
	}
	return stFlagNew
}

func materialStateStyle(downloaded bool) lipgloss.Style {
	if downloaded {
		return stDone
	}
	return stFlagNew
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
