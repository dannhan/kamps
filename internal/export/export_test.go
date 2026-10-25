package export

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ruyam/kamps/internal/model"
)

func sampleItems() []Item {
	return []Item{
		{ID: "K1", Course: "ppl", Name: "Report 1", Due: time.Date(2026, 8, 26, 0, 0, 0, 0, time.UTC), HasDue: true, Action: model.Tracked},
		{ID: "K2", Course: "metnum", Name: "Quiz", Due: time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC), HasDue: true, Action: model.Missed},
	}
}

func TestMarkdownRenderSkipsExistingAndMarksClosed(t *testing.T) {
	existing := []byte("- [ ] [ppl] Report 1 (K1) due:2026-08-26\n")
	text, result := (Markdown{}).Render(sampleItems(), existing)
	if result.Added != 1 || result.Skipped != 1 {
		t.Fatalf("result = %+v", result)
	}
	if !strings.Contains(text, "- [x] [metnum] Quiz (K2)") || !strings.Contains(text, "#missed") {
		t.Fatalf("text = %q", text)
	}
}

func TestMarkdownWriteIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.md")
	writer := Markdown{}
	first, err := writer.Write(path, sampleItems())
	if err != nil {
		t.Fatal(err)
	}
	if first.Added != 2 {
		t.Fatalf("first = %+v", first)
	}
	second, err := writer.Write(path, sampleItems())
	if err != nil {
		t.Fatal(err)
	}
	if second.Added != 0 || second.Skipped != 2 {
		t.Fatalf("second = %+v", second)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), "(K1)") != 1 {
		t.Fatalf("duplicate lines: %q", data)
	}
}

func TestTodoRenderUsesIdentityKey(t *testing.T) {
	text, result := (Todo{}).Render(sampleItems(), nil)
	if result.Added != 2 {
		t.Fatalf("result = %+v", result)
	}
	if !strings.Contains(text, "kamps:K1") || !strings.Contains(text, "x 2026-09-02") {
		t.Fatalf("text = %q", text)
	}
}

func TestTaskwarriorWriteUsesRunner(t *testing.T) {
	var calls [][]string
	oldLook, oldRun := lookTask, runner
	lookTask = func(string) (string, error) { return "/usr/bin/task", nil }
	runner = func(args ...string) ([]byte, error) {
		calls = append(calls, args)
		return nil, nil
	}
	t.Cleanup(func() { lookTask, runner = oldLook, oldRun })

	result, err := (Taskwarrior{}).Write("", sampleItems())
	if err != nil {
		t.Fatal(err)
	}
	if result.Added != 2 || len(calls) != 3 {
		t.Fatalf("result=%+v calls=%v", result, calls)
	}
	if calls[1][0] != "rc.confirmation=off" {
		t.Fatalf("add args = %v", calls[1])
	}
}

func TestTaskwarriorPreviewSkipsExisting(t *testing.T) {
	oldLook, oldRun := lookTask, runner
	lookTask = func(string) (string, error) { return "/usr/bin/task", nil }
	runner = func(args ...string) ([]byte, error) {
		return []byte(`[{"description":"Report 1 (K1)"}]`), nil
	}
	t.Cleanup(func() { lookTask, runner = oldLook, oldRun })

	text, result := (Taskwarrior{}).Preview(sampleItems())
	if result.Added != 1 || result.Skipped != 1 {
		t.Fatalf("result = %+v", result)
	}
	if strings.Contains(text, "(K1)") || !strings.Contains(text, "(K2)") {
		t.Fatalf("preview = %q", text)
	}
}
