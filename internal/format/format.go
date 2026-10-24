// Package format provides small formatting helpers shared by the interactive
// and plain-text UIs. It deliberately depends only on the domain model.
package format

import (
	"fmt"
	"sort"

	"github.com/ruyam/kamps/internal/model"
)

// HumanSize renders a byte count using the largest sensible binary unit.
func HumanSize(n int64) string {
	if n <= 0 {
		return "—"
	}
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	units := []string{"KB", "MB", "GB"}
	f := float64(n)
	i := -1
	for f >= unit && i < len(units)-1 {
		f /= unit
		i++
	}
	return fmt.Sprintf("%.1f %s", f, units[i])
}

// LessAssignment orders dated assignments first, oldest first, then by ID.
func LessAssignment(a, b model.Assignment) bool {
	if a.HasDue != b.HasDue {
		return a.HasDue
	}
	if a.HasDue && !a.Due.Equal(b.Due) {
		return a.Due.Before(b.Due)
	}
	return a.ID < b.ID
}

// SortAssignments sorts assignments in display order.
func SortAssignments(assignments []model.Assignment) {
	sort.Slice(assignments, func(i, j int) bool {
		return LessAssignment(assignments[i], assignments[j])
	})
}
