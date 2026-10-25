package export

import (
	"regexp"
	"strings"

	"github.com/ruyam/kamps/internal/model"
)

var markdownIDPattern = regexp.MustCompile(`\((K[0-9A-Za-z]+)\)`)

// Markdown writes GitHub-flavoured task lines.
type Markdown struct{}

func (Markdown) Format() string { return "markdown" }

func (Markdown) Render(items []Item, existing []byte) (string, Result) {
	result := Result{}
	seen := map[string]bool{}
	for _, match := range markdownIDPattern.FindAllSubmatch(existing, -1) {
		seen[string(match[1])] = true
	}
	var lines []string
	for _, item := range items {
		if seen[item.ID] {
			result.Skipped++
			continue
		}
		seen[item.ID] = true
		result.Added++
		box := "[ ]"
		suffix := ""
		switch item.Action {
		case model.Missed:
			box = "[x]"
			suffix = " #missed"
		case model.Ignored:
			box = "[x]"
			suffix = " #ignored"
		}
		lines = append(lines, "- "+box+" ["+item.Course+"] "+item.Name+" ("+item.ID+")"+dueSuffix(item)+suffix)
	}
	return strings.Join(lines, "\n"), result
}

func (w Markdown) Write(path string, items []Item) (Result, error) {
	return writeWithLock(path, func(existing []byte) (string, Result) {
		return w.Render(items, existing)
	})
}
