package library

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// DownloadOptions controls one transfer.
type DownloadOptions struct {
	ExpectedSize int64
	// Resume keeps a stable .part file across attempts and sends Range requests.
	Resume bool
	// Headers are added to the request (for example, an authenticated cookie is
	// already handled by the client's jar).
	Headers http.Header
}

// DownloadResult reports where the bytes landed.
type DownloadResult struct {
	Path    string
	Resumed bool
	Bytes   int64
}

type partMeta struct {
	URL          string    `json:"url"`
	ETag         string    `json:"etag,omitempty"`
	LastModified string    `json:"last_modified,omitempty"`
	Total        int64     `json:"total,omitempty"`
	Target       string    `json:"target"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// DownloadFileWithOptions streams a remote resource to destPath. With
// Resume=true it keeps a stable .part file (and .part.json metadata) across
// cancellations and process restarts, and continues with a Range request when
// the server supports it.
func DownloadFileWithOptions(ctx context.Context, client *http.Client, downloadURL, destPath string, opts DownloadOptions, progress func(downloaded, total int64)) (DownloadResult, error) {
	if client == nil {
		client = http.DefaultClient
	}

	resolvedPath, action := ResolveDestination(destPath, opts.ExpectedSize)
	if action == ActionSkip {
		if progress != nil && opts.ExpectedSize > 0 {
			progress(opts.ExpectedSize, opts.ExpectedSize)
		}
		return DownloadResult{Path: resolvedPath, Bytes: opts.ExpectedSize}, nil
	}

	destDir := filepath.Dir(resolvedPath)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return DownloadResult{}, fmt.Errorf("create download directory: %w", err)
	}
	if opts.ExpectedSize > 0 {
		if err := CheckDiskSpace(destDir, opts.ExpectedSize); err != nil {
			return DownloadResult{}, err
		}
	}

	partPath := resolvedPath + ".part"
	metaPath := partPath + ".json"
	offset, meta := loadPart(partPath, metaPath, downloadURL, opts.Resume)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return DownloadResult{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept-Encoding", "identity")
	for key, values := range opts.Headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	if opts.Resume && offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
		if meta.ETag != "" {
			req.Header.Set("If-Range", meta.ETag)
		} else if meta.LastModified != "" {
			req.Header.Set("If-Range", meta.LastModified)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return DownloadResult{Path: resolvedPath, Bytes: offset}, err
	}
	defer resp.Body.Close()

	resumed := false
	switch resp.StatusCode {
	case http.StatusOK:
		offset = 0
	case http.StatusPartialContent:
		resumed = offset > 0
	case http.StatusRequestedRangeNotSatisfiable:
		// The server no longer has the range. Restart from zero.
		offset = 0
		_ = os.Remove(partPath)
		_ = os.Remove(metaPath)
		return DownloadFileWithOptions(ctx, client, downloadURL, destPath, DownloadOptions{ExpectedSize: opts.ExpectedSize, Headers: opts.Headers}, progress)
	case http.StatusUnauthorized, http.StatusForbidden:
		cleanupPart(partPath, metaPath, opts.Resume)
		return DownloadResult{}, ErrUnauthorized
	case http.StatusNotFound, http.StatusGone:
		cleanupPart(partPath, metaPath, opts.Resume)
		return DownloadResult{}, ErrFileNotFound
	case http.StatusTooManyRequests:
		cleanupPart(partPath, metaPath, opts.Resume)
		return DownloadResult{}, ErrRateLimited
	default:
		if resp.StatusCode >= 400 {
			cleanupPart(partPath, metaPath, opts.Resume)
			return DownloadResult{}, fmt.Errorf("server error: HTTP %d", resp.StatusCode)
		}
	}

	// Detect silent HTML redirects (session expired).
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(contentType, "text/html") {
		cleanupPart(partPath, metaPath, opts.Resume)
		return DownloadResult{}, ErrSessionExpiredHTML
	}

	total := resp.ContentLength
	if total > 0 {
		total += offset
	}
	if resp.StatusCode == http.StatusPartialContent {
		if start, size, ok := parseContentRange(resp.Header.Get("Content-Range")); ok {
			total = size
			if start != offset {
				cleanupPart(partPath, metaPath, opts.Resume)
				return DownloadResult{}, ErrIncompleteDownload
			}
		}
	}
	if total <= 0 && opts.ExpectedSize > 0 {
		total = opts.ExpectedSize
	}
	if total > offset {
		if err := CheckDiskSpace(destDir, total-offset); err != nil {
			cleanupPart(partPath, metaPath, opts.Resume)
			return DownloadResult{}, err
		}
	}

	// Prefer the server's filename when the caller only knows an activity name
	// such as "Slides" and has no extension to work with.
	if disposition := dispositionFilename(resp); disposition != "" && filepath.Ext(resolvedPath) == "" {
		candidate := filepath.Join(filepath.Dir(resolvedPath), SanitizeFilename(disposition))
		if candidate != resolvedPath {
			resolved, action := ResolveDestination(candidate, opts.ExpectedSize)
			if action == ActionSkip {
				cleanupPart(partPath, metaPath, opts.Resume)
				return DownloadResult{Path: resolved}, nil
			}
			if offset > 0 {
				newPart := resolved + ".part"
				newMeta := newPart + ".json"
				if err := os.Rename(partPath, newPart); err == nil {
					partPath, metaPath = newPart, newMeta
				}
			}
			resolvedPath = resolved
		}
	}

	flags := os.O_CREATE | os.O_WRONLY
	if offset > 0 {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	partFile, err := os.OpenFile(partPath, flags, 0o644)
	if err != nil {
		return DownloadResult{}, fmt.Errorf("create temporary download file: %w", err)
	}

	success := false
	keep := false
	defer func() {
		_ = partFile.Close()
		if success {
			_ = os.Remove(metaPath)
			return
		}
		if !keep {
			_ = os.Remove(partPath)
			_ = os.Remove(metaPath)
		}
	}()

	written := offset
	if offset == 0 {
		peekBuf := make([]byte, 512)
		nPeek, peekErr := io.ReadAtLeast(resp.Body, peekBuf, 1)
		if peekErr != nil && !errors.Is(peekErr, io.EOF) {
			return DownloadResult{Path: resolvedPath, Bytes: written}, fmt.Errorf("read header bytes: %w", peekErr)
		}
		if nPeek == 0 {
			return DownloadResult{}, ErrEmptyFile
		}
		sample := strings.ToLower(string(peekBuf[:nPeek]))
		if strings.Contains(sample, "<!doctype html") || strings.Contains(sample, "<html") {
			return DownloadResult{}, ErrSessionExpiredHTML
		}
		if _, err := partFile.Write(peekBuf[:nPeek]); err != nil {
			return DownloadResult{Path: resolvedPath, Bytes: written}, fmt.Errorf("write part file: %w", err)
		}
		written += int64(nPeek)
		if progress != nil {
			progress(written, total)
		}
	}

	if opts.Resume {
		writePartMeta(metaPath, partMeta{
			URL:          downloadURL,
			ETag:         resp.Header.Get("ETag"),
			LastModified: resp.Header.Get("Last-Modified"),
			Total:        total,
			Target:       resolvedPath,
			UpdatedAt:    time.Now(),
		})
	}

	buf := make([]byte, 32*1024)
	for {
		if ctx.Err() != nil {
			keep = opts.Resume
			return DownloadResult{Path: resolvedPath, Bytes: written, Resumed: resumed}, ctx.Err()
		}
		nr, er := resp.Body.Read(buf)
		if nr > 0 {
			nw, ew := partFile.Write(buf[:nr])
			if ew != nil {
				keep = opts.Resume && (errors.Is(ew, context.Canceled) || errors.Is(ew, context.DeadlineExceeded))
				if isNoSpace(ew) {
					return DownloadResult{Path: resolvedPath, Bytes: written}, fmt.Errorf("%w: %v", ErrInsufficientDiskSpace, ew)
				}
				return DownloadResult{Path: resolvedPath, Bytes: written}, fmt.Errorf("write part file: %w", ew)
			}
			written += int64(nw)
			if progress != nil {
				progress(written, total)
			}
		}
		if er != nil {
			if errors.Is(er, io.EOF) {
				break
			}
			keep = opts.Resume && (errors.Is(er, io.ErrUnexpectedEOF) || errors.Is(er, context.Canceled) || errors.Is(er, context.DeadlineExceeded) || isNetworkError(er))
			if errors.Is(er, io.ErrUnexpectedEOF) {
				return DownloadResult{Path: resolvedPath, Bytes: written, Resumed: resumed}, ErrIncompleteDownload
			}
			return DownloadResult{Path: resolvedPath, Bytes: written, Resumed: resumed}, fmt.Errorf("read stream: %w", er)
		}
	}

	expectedTotal := total
	if expectedTotal <= 0 {
		expectedTotal = opts.ExpectedSize
	}
	if expectedTotal > 0 && written != expectedTotal {
		keep = opts.Resume
		return DownloadResult{Path: resolvedPath, Bytes: written, Resumed: resumed}, ErrIncompleteDownload
	}
	if opts.ExpectedSize > 0 && written != opts.ExpectedSize {
		keep = opts.Resume
		return DownloadResult{Path: resolvedPath, Bytes: written, Resumed: resumed}, ErrIncompleteDownload
	}

	if err := partFile.Sync(); err != nil {
		return DownloadResult{Path: resolvedPath, Bytes: written}, fmt.Errorf("flush part file: %w", err)
	}
	if err := partFile.Close(); err != nil {
		return DownloadResult{Path: resolvedPath, Bytes: written}, fmt.Errorf("close part file: %w", err)
	}
	if err := os.Rename(partPath, resolvedPath); err != nil {
		return DownloadResult{Path: resolvedPath, Bytes: written}, fmt.Errorf("rename to target destination: %w", err)
	}
	success = true
	return DownloadResult{Path: resolvedPath, Bytes: written, Resumed: resumed}, nil
}

func loadPart(partPath, metaPath, downloadURL string, resume bool) (int64, partMeta) {
	if !resume {
		return 0, partMeta{}
	}
	info, err := os.Stat(partPath)
	if err != nil || info.IsDir() {
		return 0, partMeta{}
	}
	var meta partMeta
	if data, err := os.ReadFile(metaPath); err == nil {
		_ = json.Unmarshal(data, &meta)
	}
	if meta.URL != "" && meta.URL != downloadURL {
		return 0, partMeta{}
	}
	return info.Size(), meta
}

func writePartMeta(path string, meta partMeta) {
	data, err := json.Marshal(meta)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o600)
}

func cleanupPart(partPath, metaPath string, _ bool) {
	_ = os.Remove(partPath)
	_ = os.Remove(metaPath)
}

func parseContentRange(value string) (start, total int64, ok bool) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "bytes ") {
		return 0, 0, false
	}
	value = strings.TrimPrefix(value, "bytes ")
	parts := strings.SplitN(value, "/", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	rangePart := strings.SplitN(parts[0], "-", 2)
	if len(rangePart) != 2 {
		return 0, 0, false
	}
	start, err := strconv.ParseInt(rangePart[0], 10, 64)
	if err != nil {
		return 0, 0, false
	}
	if parts[1] == "*" {
		return start, 0, true
	}
	total, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, 0, false
	}
	return start, total, true
}

func isNoSpace(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "no space left") || strings.Contains(message, "enospc")
}

func isNetworkError(err error) bool {
	var netErr interface{ Timeout() bool }
	if errors.As(err, &netErr) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "connection reset") || strings.Contains(message, "unexpected eof") || strings.Contains(message, "broken pipe") || strings.Contains(message, "timeout")
}
