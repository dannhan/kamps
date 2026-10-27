package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
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

type downloadTestSource struct {
	materials []model.Material
	calls     []string
}

func (s *downloadTestSource) Label() string { return "test" }
func (s *downloadTestSource) Load(context.Context) (source.Snapshot, error) {
	return source.Snapshot{Courses: []model.Course{{ID: 1, Code: "C1"}}, Materials: s.materials}, nil
}
func (s *downloadTestSource) Download(_ context.Context, material model.Material, path string, _ func(int64, int64)) (string, error) {
	s.calls = append(s.calls, material.ID)
	if material.ID == "bad" {
		return "", errors.New("transfer failed")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}
	return path, os.WriteFile(path, []byte("content"), 0600)
}

func TestHeadlessDownloadSelectionAndPartialResults(t *testing.T) {
	materials := []model.Material{{ID: "ok", CourseID: 1, Name: "ok", Kind: "file", URL: "https://demo.test/ok"}, {ID: "bad", CourseID: 1, Name: "bad", Kind: "file", URL: "https://demo.test/bad"}, {ID: "link", CourseID: 1, Kind: "url"}, {ID: "book", CourseID: 1, Kind: "book"}, {ID: "page", CourseID: 1, Kind: "page", URL: "https://demo.test/page"}, {ID: "lab", CourseID: 1, Kind: "praktikum", URL: "https://demo.test/lab"}, {ID: "missing-url", CourseID: 1, Kind: "file"}}
	for _, selector := range [][]string{{"--all-new"}, {"--course", "C1"}} {
		t.Run(strings.Join(selector, "_"), func(t *testing.T) {
			src := &downloadTestSource{materials: materials}
			var stdout, stderr bytes.Buffer
			args := append([]string{"download", "--json"}, selector...)
			_, code := runHeadless(args, headlessConfig(t), src, &stdout, &stderr)
			if code != 5 || !strings.Contains(stdout.String(), `"id":"ok"`) || !strings.Contains(stderr.String(), "bad: transfer failed") {
				t.Fatalf("code %d stdout %s stderr %s", code, stdout.String(), stderr.String())
			}
			sort.Strings(src.calls)
			if strings.Join(src.calls, ",") != "bad,ok" {
				t.Fatalf("calls = %v", src.calls)
			}
		})
	}
}

func TestHeadlessDownloadRejectsUnknownIDsBeforeTransfer(t *testing.T) {
	src := &downloadTestSource{materials: []model.Material{{ID: "ok", CourseID: 1, Kind: "file"}}}
	var stdout, stderr bytes.Buffer
	_, code := runHeadless([]string{"download", "ok", "missing", "--json"}, headlessConfig(t), src, &stdout, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "missing") || len(src.calls) != 0 {
		t.Fatalf("code %d calls %v stderr %s", code, src.calls, stderr.String())
	}
}

func TestHeadlessDownloadEmptyBulkSelection(t *testing.T) {
	src := &downloadTestSource{materials: []model.Material{{ID: "link", CourseID: 1, Kind: "url"}}}
	var stdout, stderr bytes.Buffer
	_, code := runHeadless([]string{"download", "--all-new", "--json"}, headlessConfig(t), src, &stdout, &stderr)
	if code != 0 || len(src.calls) != 0 || !strings.Contains(stdout.String(), `"items":[]`) {
		t.Fatalf("code %d stdout %s stderr %s", code, stdout.String(), stderr.String())
	}
}

func TestHeadlessDownloadRejectsIDsOutsideCourse(t *testing.T) {
	src := &downloadTestSource{materials: []model.Material{{ID: "other", CourseID: 2, Kind: "file"}}}
	var stdout, stderr bytes.Buffer
	_, code := runHeadless([]string{"download", "other", "--course", "C1"}, headlessConfig(t), src, &stdout, &stderr)
	if code != 2 || len(src.calls) != 0 || !strings.Contains(stderr.String(), "does not belong") {
		t.Fatalf("code %d calls %v stderr %s", code, src.calls, stderr.String())
	}
}
