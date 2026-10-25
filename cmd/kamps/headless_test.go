package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ruyam/kamps/internal/api"
	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/model"
	"github.com/ruyam/kamps/internal/source"
	"github.com/ruyam/kamps/internal/store"
)

func headlessConfig(t *testing.T) config.Config {
	t.Helper()
	return config.Config{
		StateDir:   t.TempDir(),
		LibraryDir: t.TempDir(),
		SiteURL:    "https://demo.test",
	}
}

func runTestHeadless(t *testing.T, cfg config.Config, args ...string) (string, string, int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	handled, code := runHeadless(args, cfg, source.NewDemo(), &stdout, &stderr)
	if !handled {
		t.Fatalf("%v was not handled", args)
	}
	return stdout.String(), stderr.String(), code
}

func TestHeadlessTasksJSON(t *testing.T) {
	stdout, stderr, code := runTestHeadless(t, headlessConfig(t), "tasks", "--json", "--filter", "undecided")
	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	var envelope struct {
		Version int        `json:"version"`
		Command string     `json:"command"`
		Items   []api.Task `json:"items"`
	}
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Version != api.Version || envelope.Command != "tasks" {
		t.Fatalf("envelope = %+v", envelope)
	}
	if len(envelope.Items) == 0 {
		t.Fatal("no tasks returned")
	}
	for _, task := range envelope.Items {
		if task.Triage != "undecided" {
			t.Fatalf("filter leaked %+v", task)
		}
	}
}

func TestHeadlessTriagePersists(t *testing.T) {
	cfg := headlessConfig(t)
	stdout, stderr, code := runTestHeadless(t, cfg, "triage", "K901017", "--tracked", "--json")
	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	if !strings.Contains(stdout, `"triage":"tracked"`) {
		t.Fatalf("output = %s", stdout)
	}
	state, err := store.Load(cfg.StatePath())
	if err != nil {
		t.Fatal(err)
	}
	if state.Decision("K901017") != model.Tracked {
		t.Fatalf("decision = %v", state.Decision("K901017"))
	}
}

func TestHeadlessDownloadDryRun(t *testing.T) {
	stdout, stderr, code := runTestHeadless(t, headlessConfig(t), "download", "--all-new", "--dry-run", "--json")
	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	var envelope struct {
		Items []struct {
			ID   string `json:"id"`
			Path string `json:"path"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Items) == 0 {
		t.Fatal("no download targets")
	}
	for _, item := range envelope.Items {
		if item.ID == "" || item.Path == "" {
			t.Fatalf("incomplete target %+v", item)
		}
	}
}

func TestHeadlessExportWritesFile(t *testing.T) {
	cfg := headlessConfig(t)
	out := filepath.Join(t.TempDir(), "tasks.md")
	_, stderr, code := runTestHeadless(t, cfg, "export", "--format", "markdown", "--out", out, "--ids", "K901017", "--json")
	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "K901017") {
		t.Fatalf("exported file = %q", data)
	}
}

func TestHeadlessUnknownCommandFails(t *testing.T) {
	_, stderr, code := runTestHeadless(t, headlessConfig(t), "bogus")
	if code != 2 || !strings.Contains(stderr, "unknown command") {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
}
