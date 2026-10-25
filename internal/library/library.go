package library

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	ErrInsufficientDiskSpace = errors.New("insufficient disk space")
	ErrEmptyFile             = errors.New("remote file is empty (0 bytes)")
	ErrIncompleteDownload    = errors.New("download incomplete: byte count mismatch")
	ErrSessionExpiredHTML    = errors.New("session expired: server returned an HTML page instead of the requested file")
	ErrFileNotFound          = errors.New("file not found on server (HTTP 404/410)")
	ErrUnauthorized          = errors.New("unauthorized: session rejected by server")
	ErrRateLimited           = errors.New("rate limited by server (HTTP 429)")
)

// DestinationAction indicates the decision made when resolving a destination path.
type DestinationAction int

const (
	ActionDownload DestinationAction = iota
	ActionSkip
	ActionVersion
)

// SanitizeFilename cleans a filename to prevent path traversal, invalid characters,
// and excessive length while preserving file extensions.
func SanitizeFilename(name string) string {
	cleaned := strings.TrimSpace(name)
	if cleaned == "" {
		return "unnamed"
	}

	// Remove path traversal and directory separators.
	cleaned = filepath.Base(cleaned)
	if cleaned == "." || cleaned == "/" || cleaned == "\\" {
		return "unnamed"
	}

	// Replace illegal filesystem characters: \ / : * ? " < > | and control characters.
	var sb strings.Builder
	for _, r := range cleaned {
		if r < 32 || r == 127 || strings.ContainsRune(`\/:*?"<>|`, r) {
			sb.WriteRune('_')
		} else {
			sb.WriteRune(r)
		}
	}
	result := strings.Trim(sb.String(), ". ")
	if result == "" {
		return "unnamed"
	}

	// Truncate length if over 200 bytes, preserving extension.
	const maxLen = 200
	if len(result) > maxLen {
		ext := filepath.Ext(result)
		stem := strings.TrimSuffix(result, ext)
		if len(ext) >= maxLen {
			result = result[:maxLen]
		} else {
			availableStemLen := maxLen - len(ext)
			if availableStemLen > 0 && len(stem) > availableStemLen {
				stem = stem[:availableStemLen]
			}
			result = stem + ext
		}
	}
	return result
}

// SectionPath builds a structured relative or absolute path:
// <baseDir>/<course>/<section>/<filename>
func SectionPath(baseDir, courseCodeOrName, sectionName, filename string) string {
	courseClean := SanitizeFilename(courseCodeOrName)
	if courseClean == "unnamed" {
		courseClean = "general-course"
	}
	secClean := SanitizeFilename(sectionName)
	if secClean == "unnamed" || secClean == "" {
		secClean = "general"
	}
	fileClean := SanitizeFilename(filename)
	return filepath.Join(baseDir, courseClean, secClean, fileClean)
}

// ResolveDestination checks an existing target file and determines whether
// to download, skip (identical size), or version (different size / user-modified).
func ResolveDestination(targetPath string, remoteSize int64) (string, DestinationAction) {
	fi, err := os.Stat(targetPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return targetPath, ActionDownload
		}
		// On permission or other stat error, return targetPath.
		return targetPath, ActionDownload
	}

	// File exists. If expected size is known (> 0) and matches exactly:
	if remoteSize > 0 && fi.Size() == remoteSize {
		return targetPath, ActionSkip
	}

	// Size differs (or remoteSize unknown). Do NOT overwrite user's file.
	// Version suffix: <stem>_v2.<ext>, <stem>_v3.<ext>, etc.
	dir := filepath.Dir(targetPath)
	ext := filepath.Ext(targetPath)
	stem := strings.TrimSuffix(filepath.Base(targetPath), ext)

	for version := 2; version < 1000; version++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s_v%d%s", stem, version, ext))
		candFi, candErr := os.Stat(candidate)
		if candErr != nil && errors.Is(candErr, os.ErrNotExist) {
			return candidate, ActionVersion
		}
		if candErr == nil && remoteSize > 0 && candFi.Size() == remoteSize {
			return candidate, ActionSkip
		}
	}

	return targetPath, ActionDownload
}

// availableBytes returns filesystem bytes available to the current user.
func availableBytes(path string) (int64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	return int64(stat.Bavail) * int64(stat.Bsize), nil
}

// CheckDiskSpace checks whether the filesystem containing dir has enough free space.
func CheckDiskSpace(dir string, needed int64) error {
	if needed <= 0 {
		return nil
	}
	available, err := availableBytes(dir)
	if err != nil {
		return fmt.Errorf("check available disk space: %w", err)
	}
	if available < needed {
		return fmt.Errorf("%w: need %d bytes, have %d bytes", ErrInsufficientDiskSpace, needed, available)
	}
	return nil
}

// Coordinator tracks active downloads to avoid duplicate concurrent transfers.
type Coordinator struct {
	mu     sync.Mutex
	active map[string]bool
}

// NewCoordinator creates a thread-safe download coordinator.
func NewCoordinator() *Coordinator {
	return &Coordinator{active: make(map[string]bool)}
}

// TryStart attempts to mark an ID as downloading. Returns false if already active.
func (c *Coordinator) TryStart(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.active[id] {
		return false
	}
	c.active[id] = true
	return true
}

// Finish unmarks an active download ID.
func (c *Coordinator) Finish(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.active, id)
}

// IsActive reports whether a download is in-flight.
func (c *Coordinator) IsActive(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.active[id]
}

// dispositionFilename returns the filename advertised by Content-Disposition.
func dispositionFilename(resp *http.Response) string {
	header := resp.Header.Get("Content-Disposition")
	if header == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(header)
	if err != nil {
		return ""
	}
	return params["filename"]
}

// DownloadFile streams a remote resource to destPath safely.
// It handles:
// - Destination directory creation
// - Pre-flight disk space validation
// - Temporary .part files with atomic rename
// - HTML login redirect detection
// - Clean cancellation cleanup on failure
func DownloadFile(ctx context.Context, client *http.Client, downloadURL, destPath string, expectedSize int64, progress func(downloaded, total int64)) (string, error) {
	if client == nil {
		client = http.DefaultClient
	}

	resolvedPath, action := ResolveDestination(destPath, expectedSize)
	if action == ActionSkip {
		if progress != nil && expectedSize > 0 {
			progress(expectedSize, expectedSize)
		}
		return resolvedPath, nil
	}

	destDir := filepath.Dir(resolvedPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("create download directory: %w", err)
	}

	if expectedSize > 0 {
		if err := CheckDiskSpace(destDir, expectedSize); err != nil {
			return "", err
		}
	}

	partPath := fmt.Sprintf("%s.kamps-%d-%d.part", resolvedPath, os.Getpid(), time.Now().UnixNano())
	partFile, err := os.OpenFile(partPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return "", fmt.Errorf("create temporary download file: %w", err)
	}

	partCleaned := false
	cleanup := func() {
		if !partCleaned {
			partCleaned = true
			_ = partFile.Close()
			_ = os.Remove(partPath)
		}
	}
	defer cleanup()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// Expected
	case http.StatusUnauthorized, http.StatusForbidden:
		return "", ErrUnauthorized
	case http.StatusNotFound, http.StatusGone:
		return "", ErrFileNotFound
	case http.StatusTooManyRequests:
		return "", ErrRateLimited
	default:
		if resp.StatusCode >= 400 {
			return "", fmt.Errorf("server error: HTTP %d", resp.StatusCode)
		}
	}

	// Detect silent HTML redirects (session expired)
	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(ct, "text/html") {
		return "", ErrSessionExpiredHTML
	}

	// Read and inspect initial bytes for magic HTML tags
	peekBuf := make([]byte, 512)
	nPeek, peekErr := io.ReadAtLeast(resp.Body, peekBuf, 1)
	if peekErr != nil && !errors.Is(peekErr, io.EOF) {
		return "", fmt.Errorf("read header bytes: %w", peekErr)
	}
	if nPeek == 0 {
		return "", ErrEmptyFile
	}

	peekSample := strings.ToLower(string(peekBuf[:nPeek]))
	if strings.Contains(peekSample, "<!doctype html") || strings.Contains(peekSample, "<html") {
		return "", ErrSessionExpiredHTML
	}

	// Prefer the server's filename when the caller only knows an activity name
	// such as "Slides" and has no extension to work with.
	if disposition := dispositionFilename(resp); disposition != "" && filepath.Ext(resolvedPath) == "" {
		candidate := filepath.Join(filepath.Dir(resolvedPath), SanitizeFilename(disposition))
		if candidate != resolvedPath {
			resolved, action := ResolveDestination(candidate, expectedSize)
			if action == ActionSkip {
				return resolved, nil
			}
			resolvedPath = resolved
		}
	}

	// Write peeked bytes to part file
	if _, err := partFile.Write(peekBuf[:nPeek]); err != nil {
		return "", fmt.Errorf("write part file: %w", err)
	}

	downloaded := int64(nPeek)
	total := resp.ContentLength
	if total <= 0 && expectedSize > 0 {
		total = expectedSize
	}
	if progress != nil {
		progress(downloaded, total)
	}

	// Stream rest of payload using 32 KB buffer
	buf := make([]byte, 32*1024)
	for {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		nr, er := resp.Body.Read(buf)
		if nr > 0 {
			nw, ew := partFile.Write(buf[:nr])
			if ew != nil {
				return "", fmt.Errorf("write part file: %w", ew)
			}
			downloaded += int64(nw)
			if progress != nil {
				progress(downloaded, total)
			}
		}
		if er != nil {
			if errors.Is(er, io.EOF) {
				break
			}
			if errors.Is(er, io.ErrUnexpectedEOF) {
				return "", ErrIncompleteDownload
			}
			return "", fmt.Errorf("read stream: %w", er)
		}
	}

	if resp.ContentLength > 0 && downloaded != resp.ContentLength {
		return "", ErrIncompleteDownload
	}
	if expectedSize > 0 && downloaded != expectedSize {
		return "", ErrIncompleteDownload
	}

	if err := partFile.Sync(); err != nil {
		return "", fmt.Errorf("flush part file: %w", err)
	}
	if err := partFile.Close(); err != nil {
		return "", fmt.Errorf("close part file: %w", err)
	}

	if err := os.Rename(partPath, resolvedPath); err != nil {
		return "", fmt.Errorf("rename to target destination: %w", err)
	}
	partCleaned = true

	return resolvedPath, nil
}
