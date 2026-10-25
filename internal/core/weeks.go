package core

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var weekSectionPattern = regexp.MustCompile(`(?i)^week[-_ ]?0*([0-9]+)$`)

// SectionNumber parses a week label such as week-2.
func SectionNumber(section string) (int, bool) {
	match := weekSectionPattern.FindStringSubmatch(strings.TrimSpace(section))
	if len(match) < 2 {
		return 0, false
	}
	n, err := strconv.Atoi(match[1])
	return n, err == nil
}

// SectionLess orders week-1 before week-2 before week-10.
func SectionLess(a, b string) bool {
	na, oka := SectionNumber(a)
	nb, okb := SectionNumber(b)
	if oka && okb {
		return na < nb
	}
	if oka != okb {
		return oka
	}
	return a < b
}

// SectionLabel normalizes week-01 to week-1 and empty to week-0.
func SectionLabel(section string) string {
	section = strings.TrimSpace(section)
	if section == "" {
		return "week-0"
	}
	if match := weekSectionPattern.FindStringSubmatch(section); len(match) > 1 {
		if n, err := strconv.Atoi(match[1]); err == nil {
			return fmt.Sprintf("week-%d", n)
		}
	}
	return section
}
