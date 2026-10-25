package export

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/ruyam/kamps/internal/model"
)

// Taskwarrior adds tasks through the task CLI. It never edits Taskwarrior's
// data files directly.
type Taskwarrior struct{}

func (Taskwarrior) Format() string { return "taskwarrior" }

// runner and lookTask are replaceable in tests.
var lookTask = exec.LookPath

var runner = func(args ...string) ([]byte, error) {
	cmd := exec.Command("task", args...)
	return cmd.CombinedOutput()
}

func (Taskwarrior) Render(items []Item, existing []byte) (string, Result) {
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
		lines = append(lines, taskCommand(item))
	}
	return strings.Join(lines, "\n"), result
}

func taskCommand(item Item) string {
	project := strings.ReplaceAll(strings.TrimSpace(item.Course), " ", "-")
	if project == "" {
		project = "kamps"
	}
	args := []string{"rc.confirmation=off", "add", "project:" + project, "+kamps", item.Name + " (" + item.ID + ")"}
	if item.Action == model.Missed || item.Action == model.Ignored {
		args = append(args, "+closed")
	}
	return "task " + strings.Join(args, " ")
}

// Preview queries existing tasks so the preview shows the same skip set as a
// real write.
func (w Taskwarrior) Preview(items []Item) (string, Result) {
	if _, err := lookTask("task"); err != nil {
		return "", Result{}
	}
	existing, _ := runner("export", "+kamps")
	return w.Render(items, existing)
}

func (w Taskwarrior) Write(path string, items []Item) (Result, error) {
	if _, err := lookTask("task"); err != nil {
		return Result{}, fmt.Errorf("taskwarrior is not installed")
	}
	existing, _ := runner("export", "+kamps")
	_, result := w.Render(items, existing)

	seen := map[string]bool{}
	for _, match := range markdownIDPattern.FindAllSubmatch(existing, -1) {
		seen[string(match[1])] = true
	}
	for _, item := range items {
		if seen[item.ID] {
			continue
		}
		seen[item.ID] = true
		project := strings.ReplaceAll(strings.TrimSpace(item.Course), " ", "-")
		if project == "" {
			project = "kamps"
		}
		args := []string{"rc.confirmation=off", "add", "project:" + project, "+kamps", item.Name + " (" + item.ID + ")"}
		if item.Action == model.Missed || item.Action == model.Ignored {
			args = append(args, "+closed")
		}
		if output, err := runner(args...); err != nil {
			return result, fmt.Errorf("task add: %v: %s", err, strings.TrimSpace(string(output)))
		}
	}
	return result, nil
}
