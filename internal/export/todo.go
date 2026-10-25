package export

import (
	"regexp"
	"strings"

	"github.com/ruyam/kamps/internal/model"
)

var todoIDPattern = regexp.MustCompile(`kamps:(K[0-9A-Za-z]+)`)

// Todo writes todo.txt-compatible lines.
type Todo struct{}

func (Todo) Format() string { return "todo" }

func (Todo) Render(items []Item, existing []byte) (string, Result) {
	result := Result{}
	seen := map[string]bool{}
	for _, match := range todoIDPattern.FindAllSubmatch(existing, -1) {
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
		line := "[" + item.Course + "] " + item.Name + " (" + item.ID + ")" + dueSuffix(item) + " kamps:" + item.ID
		switch item.Action {
		case model.Missed:
			line = "x " + item.Due.Format("2006-01-02") + " " + line + " #missed"
		case model.Ignored:
			line = "x " + item.Due.Format("2006-01-02") + " " + line + " #ignored"
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n"), result
}

func (w Todo) Write(path string, items []Item) (Result, error) {
	return writeWithLock(path, func(existing []byte) (string, Result) {
		return w.Render(items, existing)
	})
}
