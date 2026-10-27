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
	// Canvas is the terminal background the palette was designed against. It is
	// never painted; it exists so contrast checks use the same tokens.
	Canvas lipgloss.Color

	Surface  lipgloss.Color
	Marked   lipgloss.Color
	Border   lipgloss.Color
	Bright   lipgloss.Color
	Primary  lipgloss.Color
	Dim      lipgloss.Color
	Subtle   lipgloss.Color
	TreeLine lipgloss.Color
	Accent   lipgloss.Color
	Heading  lipgloss.Color

	// TreeConnector is deliberately darker/dimmer than Primary so the rails
	// recede behind titles.
	TreeConnector lipgloss.Color

	// Due-date urgency slots, from hottest to faintest.
	DueOverdue lipgloss.Color
	DueSoon    lipgloss.Color
	DueNormal  lipgloss.Color
	DueFar     lipgloss.Color
	DueNone    lipgloss.Color

	New     lipgloss.Color
	Tracked lipgloss.Color
	Done    lipgloss.Color
	Ignored lipgloss.Color
	Missed  lipgloss.Color

	Mono bool
}

func darkColors() themeColors {
	return themeColors{
		Canvas:        "#1B1B26",
		Surface:       "#334252",
		Marked:        "#30394A",
		Border:        "#3B3E5E",
		Bright:        "#EDEDED",
		Primary:       "#EDEDED",
		Dim:           "#C6C6C6",
		Subtle:        "#9A9AA4",
		TreeLine:      lipgloss.Color(quietHex("#EDEDED", "#1B1B26")),
		Accent:        "#89B4FA",
		Heading:       "#E2CCA4",
		TreeConnector: "#6A6A74",
		DueOverdue:    "#F7768E",
		DueSoon:       "#F0A868",
		DueNormal:     "#EDEDED",
		DueFar:        "#9A9AA4",
		DueNone:       "#8A8A8A",
		New:           "#9A9AA4",
		Tracked:       "#B6C1CB",
		Done:          "#6FCF97",
		Ignored:       "#9A9AA4",
		Missed:        "#F7768E",
	}
}

func lightColors() themeColors {
	return themeColors{
		Canvas:        "#FFFFFF",
		Surface:       "#D4D4DE",
		Marked:        "#CFCFE8",
		Border:        "#C8C8D0",
		Bright:        "#1B1B26",
		Primary:       "#1B1B26",
		Dim:           "#55555E",
		Subtle:        "#6B6B75",
		TreeLine:      lipgloss.Color(quietHex("#1B1B26", "#FFFFFF")),
		Accent:        "#3F4390",
		Heading:       "#71521F",
		TreeConnector: "#4B4B54",
		DueOverdue:    "#C0392B",
		DueSoon:       "#A65D00",
		DueNormal:     "#1B1B26",
		DueFar:        "#6B6B75",
		DueNone:       "#8A8A8A",
		New:           "#6B6B75",
		Tracked:       "#96590B",
		Done:          "#177A43",
		Ignored:       "#7651A8",
		Missed:        "#C0392B",
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
	return []lipgloss.Color{"#91B9EB", "#83C9C0", "#B9A5E8", "#A5C7A0", "#E1A4CA", "#91CFDF", "#C0B7F0", "#9CC3B8"}
}

func lightCourseAccents() []lipgloss.Color {
	return []lipgloss.Color{"#3F64A8", "#1F7A6A", "#7651A8", "#3E7D3E", "#97466E", "#24758A", "#64549C", "#416E64"}
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

	stDueOverdue lipgloss.Style
	stDueSoon    lipgloss.Style
	stDueNormal  lipgloss.Style
	stDueFar     lipgloss.Style
	stDueNone    lipgloss.Style

	stSectionTitle lipgloss.Style
	stPaneTitle    lipgloss.Style
	stRule         lipgloss.Style
	stKey          lipgloss.Style

	stFlagNew lipgloss.Style
	stFlagT   lipgloss.Style
	stFlagM   lipgloss.Style
	stFlagI   lipgloss.Style
)

func buildStyles() {
	stPrimary = fgStyle(themeTokens.Primary)
	stDim = fgStyle(themeTokens.Dim)
	stGuide = fgStyle(themeTokens.TreeConnector)
	stTreeLine = fgStyle(themeTokens.TreeLine)
	stFaint = fgStyle(themeTokens.Subtle)
	stEmpty = fgStyle(themeTokens.Subtle).Italic(true)
	stHelpKey = fgStyle(themeTokens.Bright)
	stHelpTxt = fgStyle(themeTokens.Subtle)
	stHelpSep = fgStyle(themeTokens.Subtle)

	stRow = stPrimary
	stRowDim = fgStyle(themeTokens.Subtle)
	stCursor = fgStyle(themeTokens.Accent)
	stSelMark = fgStyle(themeTokens.Accent)

	stDone = fgStyle(themeTokens.Done)
	stCode = fgStyle(themeTokens.Accent)

	stDueOverdue = fgStyle(themeTokens.DueOverdue).Bold(true)
	stDueSoon = fgStyle(themeTokens.DueSoon)
	stDueNormal = fgStyle(themeTokens.DueNormal)
	stDueFar = fgStyle(themeTokens.DueFar)
	stDueNone = fgStyle(themeTokens.DueNone)

	stSoon = stDueSoon
	stOverdue = stDueOverdue

	stSectionTitle = fgStyle(themeTokens.Heading).Bold(true)
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

// Allocate accents by the complete course list, rather than ID modulo palette
// size: unrelated LMS IDs can otherwise collide even with only two courses.
func (m *App) courseStyle(courseID int) lipgloss.Style {
	if len(courseAccents) == 0 {
		return stDim
	}
	index := 0
	for i, course := range m.sidebarCourses() {
		if course.ID == courseID {
			index = i
			break
		}
	}
	return fgStyle(courseAccent(index))
}

func courseAccent(index int) lipgloss.Color {
	if index < len(courseAccents) {
		return courseAccents[index]
	}
	// Additional courses get new hues instead of wrapping the fixed palette.
	// HSV keeps their brightness and saturation consistent with the theme.
	hue := math.Mod(float64(index-len(courseAccents))*137.508+17, 360) / 60
	value, saturation := 0.88, 0.30
	if themeTokens.Canvas == lightColors().Canvas {
		value, saturation = 0.48, 0.58
	}
	chroma := value * saturation
	x := chroma * (1 - math.Abs(math.Mod(hue, 2)-1))
	var red, green, blue float64
	switch int(hue) {
	case 0:
		red, green = chroma, x
	case 1:
		red, green = x, chroma
	case 2:
		green, blue = chroma, x
	case 3:
		green, blue = x, chroma
	case 4:
		red, blue = x, chroma
	default:
		red, blue = chroma, x
	}
	base := value - chroma
	return lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", int((red+base)*255), int((green+base)*255), int((blue+base)*255)))
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
