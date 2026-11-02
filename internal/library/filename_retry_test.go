package library

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type filenameTransport struct{ body string }

func (f filenameTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Disposition": {`attachment; filename="slides.pptx"`}}, ContentLength: int64(len(f.body)), Body: io.NopCloser(strings.NewReader(f.body)), Request: r}, nil
}

type extensionlessFilenameTransport struct{ body string }

func (f extensionlessFilenameTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Disposition": {`attachment; filename="0. Struktur Data File"`}}, ContentLength: int64(len(f.body)), Body: io.NopCloser(strings.NewReader(f.body)), Request: r}, nil
}

type rangeResetTransport struct{ requests int }

func (f *rangeResetTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	f.requests++
	if f.requests == 1 {
		return &http.Response{StatusCode: 416, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
	}
	return &http.Response{StatusCode: 200, Header: http.Header{}, ContentLength: 5, Body: io.NopCloser(strings.NewReader("hello")), Request: r}, nil
}

func TestRejectedRangeRestartsWithoutHoldingLock(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "file.pdf")
	if err := os.WriteFile(dest+".part", []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	transport := &rangeResetTransport{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result, err := DownloadFileWithOptions(ctx, &http.Client{Transport: transport}, "https://example.test/file", dest, DownloadOptions{Resume: true, LockDir: filepath.Join(dir, "locks")}, nil)
	if err != nil || result.Path != dest || transport.requests != 2 {
		t.Fatalf("result = %+v, requests %d, error %v", result, transport.requests, err)
	}
}

func TestServerFilenameAndRetryUseContents(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "0. Struktur Data File")
	client := &http.Client{Transport: filenameTransport{body: "first"}}
	opts := DownloadOptions{Resume: true, LockDir: filepath.Join(dir, "locks")}
	first, err := DownloadFileWithOptions(context.Background(), client, "https://example.test/file", dest, opts, nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.Path != filepath.Join(dir, "slides.pptx") {
		t.Fatalf("path = %q", first.Path)
	}
	again, err := DownloadFileWithOptions(context.Background(), client, "https://example.test/file", dest, opts, nil)
	if err != nil || again.Path != first.Path {
		t.Fatalf("retry = %+v, %v", again, err)
	}
	client.Transport = filenameTransport{body: "other"}
	changed, err := DownloadFileWithOptions(context.Background(), client, "https://example.test/file", dest, opts, nil)
	if err != nil || changed.Path != filepath.Join(dir, "slides_v2.pptx") {
		t.Fatalf("changed = %+v, %v", changed, err)
	}
	data, err := os.ReadFile(first.Path)
	if err != nil || string(data) != "first" {
		t.Fatalf("original = %q, %v", data, err)
	}
}

func TestExtensionlessServerFilenameKeepsPlannedExtension(t *testing.T) {
	body := "presentation"
	dir := t.TempDir()
	dest := filepath.Join(dir, "0. Struktur Data.pptx")
	client := &http.Client{Transport: extensionlessFilenameTransport{body: body}}
	result, err := DownloadFileWithOptions(context.Background(), client, "https://example.test/resource", dest, DownloadOptions{Resume: true, LockDir: filepath.Join(dir, "locks")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "0. Struktur Data File.pptx"); result.Path != want {
		t.Fatalf("path = %q, want %q", result.Path, want)
	}
	saved, err := os.ReadFile(result.Path)
	if err != nil || string(saved) != body {
		t.Fatalf("saved = %q, err = %v", saved, err)
	}
}
