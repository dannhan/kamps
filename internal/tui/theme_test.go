package tui

import (
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestQuietTreeLineColorTracksForegroundAndBackground(t *testing.T) {
	for _, pair := range [][2]string{{"#1B1B26", "#E4E4E4"}, {"#EDEDED", "#333355"}} {
		got := quietHex(pair[0], pair[1])
		if got == pair[0] || got == pair[1] || len(got) != 7 {
			t.Fatalf("quietHex(%q, %q) = %q", pair[0], pair[1], got)
		}
	}
	if got := quietHex("bad", "#000000"); got != "#8A8A8A" {
		t.Fatalf("invalid colors fallback = %q", got)
	}
}

func TestPaletteContrast(t *testing.T) {
	defer applyTheme("dark")

	cases := []struct {
		name  string
		theme themeColors
		min   float64
	}{
		{"dark", darkColors(), 4.5},
		{"light", lightColors(), 4.5},
	}
	for _, tc := range cases {
		bg := string(tc.theme.Canvas)
		t.Run(tc.name, func(t *testing.T) {
			text := map[string]lipgloss.Color{
				"primary":    tc.theme.Primary,
				"dim":        tc.theme.Dim,
				"subtle":     tc.theme.Subtle,
				"accent":     tc.theme.Accent,
				"new":        tc.theme.New,
				"tracked":    tc.theme.Tracked,
				"done":       tc.theme.Done,
				"ignored":    tc.theme.Ignored,
				"missed":     tc.theme.Missed,
				"due red":    tc.theme.DueOverdue,
				"due amber":  tc.theme.DueSoon,
				"due normal": tc.theme.DueNormal,
				"due far":    tc.theme.DueFar,
			}
			for name, color := range text {
				ratio := contrastRatio(string(color), bg)
				if ratio < tc.min {
					t.Errorf("%s/%s contrast = %.2f:1, want >= %.1f:1", tc.name, name, ratio, tc.min)
				}
			}

			for name, color := range map[string]lipgloss.Color{
				"tree connector": tc.theme.TreeConnector,
				"due none":       tc.theme.DueNone,
			} {
				if ratio := contrastRatio(string(color), bg); ratio < 3.0 {
					t.Errorf("%s/%s contrast = %.2f:1, want >= 3.0:1", tc.name, name, ratio)
				}
			}

			accents := darkCourseAccents()
			if tc.name == "light" {
				accents = lightCourseAccents()
			}
			for i, accent := range accents {
				if ratio := contrastRatio(string(accent), bg); ratio < tc.min {
					t.Errorf("%s/course accent %d contrast = %.2f:1, want >= %.1f:1", tc.name, i, ratio, tc.min)
				}
			}

			for name, band := range map[string]lipgloss.Color{"surface": tc.theme.Surface, "marked": tc.theme.Marked} {
				if ratio := contrastRatio(string(band), bg); ratio < 1.4 {
					t.Errorf("%s/%s band contrast = %.2f:1, want >= 1.4:1", tc.name, name, ratio)
				}
				if ratio := contrastRatio(string(tc.theme.Primary), string(band)); ratio < 4.5 {
					t.Errorf("%s/%s text contrast = %.2f:1, want >= 4.5:1", tc.name, name, ratio)
				}
			}
		})
	}
}

func TestDueToneBoundaries(t *testing.T) {
	for _, tc := range []struct {
		days   int
		hasDue bool
		want   dueTone
	}{
		{-1, true, dueToneOverdue},
		{0, true, dueToneSoon},
		{7, true, dueToneSoon},
		{8, true, dueToneNormal},
		{29, true, dueToneNormal},
		{30, true, dueToneFar},
		{0, false, dueToneNone},
	} {
		if got := dueToneForDays(tc.days, tc.hasDue); got != tc.want {
			t.Errorf("dueToneForDays(%d, %v) = %v, want %v", tc.days, tc.hasDue, got, tc.want)
		}
	}
}

func TestDueTonesUseDistinctColorTokens(t *testing.T) {
	applyTheme("dark")
	defer applyTheme("dark")

	tones := []lipgloss.Color{
		themeTokens.DueOverdue,
		themeTokens.DueSoon,
		themeTokens.DueNormal,
		themeTokens.DueFar,
		themeTokens.DueNone,
	}
	seen := map[lipgloss.Color]bool{}
	for _, color := range tones {
		if color == "" {
			t.Fatal("due tone has an empty color token")
		}
		if seen[color] {
			t.Fatalf("due tones reuse the color token %q", color)
		}
		seen[color] = true
	}
}

func TestMonoThemeHasNoColors(t *testing.T) {
	applyTheme("mono")
	defer applyTheme("dark")

	if !themeTokens.Mono {
		t.Fatal("mono theme not marked")
	}
	for name, color := range map[string]lipgloss.Color{
		"surface": themeTokens.Surface,
		"marked":  themeTokens.Marked,
		"accent":  themeTokens.Accent,
		"tracked": themeTokens.Tracked,
	} {
		if color != "" {
			t.Errorf("mono %s = %q, want empty", name, color)
		}
	}
	rendered := stCursor.Render("x") + stOverdue.Render("y")
	for _, seq := range []string{"38;", "48;", "38;5;", "48;5;"} {
		if strings.Contains(rendered, seq) {
			t.Errorf("mono rendering contains color sequence %q: %q", seq, rendered)
		}
	}
}

func TestNormalizeTheme(t *testing.T) {
	for _, pair := range [][2]string{
		{"", string(themeAuto)},
		{"AUTO", string(themeAuto)},
		{"dark", string(themeDark)},
		{" Light ", string(themeLight)},
		{"mono", string(themeMono)},
		{"solarized", string(themeAuto)},
	} {
		if got := normalizeTheme(pair[0]); got != themeMode(pair[1]) {
			t.Errorf("normalizeTheme(%q) = %q, want %q", pair[0], got, pair[1])
		}
	}
}

func contrastRatio(a, b string) float64 {
	la, lb := relativeLuminance(a), relativeLuminance(b)
	hi, lo := math.Max(la, lb), math.Min(la, lb)
	return (hi + 0.05) / (lo + 0.05)
}

func relativeLuminance(hex string) float64 {
	hex = strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(hex) != 6 {
		return 0
	}
	var linear [3]float64
	for i := range linear {
		component, err := strconv.ParseUint(hex[i*2:i*2+2], 16, 8)
		if err != nil {
			return 0
		}
		v := float64(component) / 255
		if v <= 0.04045 {
			linear[i] = v / 12.92
		} else {
			linear[i] = math.Pow((v+0.055)/1.055, 2.4)
		}
	}
	return 0.2126*linear[0] + 0.7152*linear[1] + 0.0722*linear[2]
}
