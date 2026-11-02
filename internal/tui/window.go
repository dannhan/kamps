package tui

import (
	"fmt"
	"strings"
)

func windowRange(total, cursor, height int) (int, int) {
	if height < 1 {
		height = 1
	}
	if total <= height {
		return 0, total
	}
	start := cursor - height/2
	if start < 0 {
		start = 0
	}
	if start+height > total {
		start = total - height
	}
	return start, start + height
}

func cursorWindow(total, cursor, height, scrolloff int, offset *int) (int, int) {
	if height < 1 {
		height = 1
	}
	if total <= height {
		*offset = 0
		return 0, total
	}
	cursor = clampInt(cursor, 0, total-1)
	margin := minInt(maxInt(0, scrolloff), (height-1)/2)
	start := clampInt(*offset, 0, total-height)
	if cursor < start+margin {
		start = maxInt(0, cursor-margin)
	}
	if cursor >= start+height-margin {
		start = minInt(total-height, cursor-height+margin+1)
	}
	*offset = start
	return start, start + height
}

// scrollHint reports how many rows the viewport hides above and below the
// window windowRange would pick. It returns "" when everything fits, so the
// caller can append it to a title row without special-casing the full case.
func scrollHint(total, cursor, height int) string {
	start, end := windowRange(total, cursor, height)
	var parts []string
	if start > 0 {
		parts = append(parts, fmt.Sprintf("▲ %d", start))
	}
	if end < total {
		parts = append(parts, fmt.Sprintf("▼ %d", total-end))
	}
	return stFaint.Render(strings.Join(parts, " "))
}
