package kulon2

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/ruyam/kamps/internal/library"
	"github.com/ruyam/kamps/internal/model"
)

// Download fetches a material's bytes through the authenticated session.
// External links are rejected: the caller opens those in a browser instead.
func (s *LiveSource) Download(ctx context.Context, material model.Material, destPath string, progress func(done, total int64)) (string, error) {
	if material.Kind == "url" {
		return "", errors.New("material is an external link, not a file")
	}
	if material.URL == "" {
		return "", fmt.Errorf("material %s has no download URL", material.ID)
	}

	if material.Kind == "book" {
		return "", errors.New("book activities are not downloadable")
	}

	downloadURL := material.URL
	if !isDirectFileURL(downloadURL) {
		if activityType(downloadURL) == "book" {
			return "", errors.New("book activities are not downloadable")
		}
		resolved, err := s.resolveFileURL(ctx, downloadURL)
		if err != nil {
			return "", err
		}
		downloadURL = resolved
	}

	base, err := parseSiteURL(s.siteURL)
	if err != nil {
		return "", err
	}
	client := cloneClientWithCookie(s.client, base, s.session.Cookie, 30*time.Second)

	if s.downloadSlots != nil {
		select {
		case s.downloadSlots <- struct{}{}:
			defer func() { <-s.downloadSlots }()
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	const maxAttempts = 3
	for attempt := 1; ; attempt++ {
		if err := s.waitForRequestSlot(ctx); err != nil {
			return "", err
		}
		result, err := library.DownloadFileWithOptions(ctx, client, downloadURL, destPath, library.DownloadOptions{
			ExpectedSize: material.Size,
			Resume:       true,
		}, progress)
		if errors.Is(err, library.ErrSessionExpiredHTML) || errors.Is(err, library.ErrUnauthorized) {
			return "", fmt.Errorf("%w: run `kamps login` again", err)
		}
		if err == nil || !retryableDownloadError(err) || attempt >= maxAttempts {
			return result.Path, err
		}
		if err := s.sleepContext(ctx, time.Duration(attempt)*500*time.Millisecond); err != nil {
			return result.Path, err
		}
	}
}

func retryableDownloadError(err error) bool {
	if errors.Is(err, library.ErrRateLimited) || errors.Is(err, library.ErrIncompleteDownload) {
		return true
	}
	message := err.Error()
	return strings.Contains(message, "HTTP 500") || strings.Contains(message, "HTTP 502") || strings.Contains(message, "HTTP 503") || strings.Contains(message, "HTTP 504")
}

// resolveFileURL turns a Moodle activity page such as
// /mod/resource/view.php?id=123 into the pluginfile.php link it serves.
func (s *LiveSource) resolveFileURL(ctx context.Context, pageURL string) (string, error) {
	page, err := s.getPage(ctx, pageURL)
	if errors.Is(err, ErrSessionExpired) {
		return "", fmt.Errorf("%w: run `kamps login` again", err)
	}
	if err != nil {
		return "", err
	}
	if link := findPluginFileURL(page, pageURL); link != "" {
		return link, nil
	}
	return "", errors.New("resource page has no downloadable file")
}

func isDirectFileURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	return err == nil && strings.Contains(parsed.Path, "pluginfile.php")
}

// activityType returns the <type> in /mod/<type>/view.php.
func activityType(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	for i := 0; i+1 < len(parts); i++ {
		if parts[i] == "mod" {
			return parts[i+1]
		}
	}
	return ""
}

// findPluginFileURL looks for the actual file behind a resource page. Moodle
// may expose it as a download anchor or embed it in an object/iframe.
func findPluginFileURL(page []byte, pageURL string) string {
	base, err := url.Parse(pageURL)
	if err != nil {
		return ""
	}
	targets := []struct{ tag, attr string }{
		{"a", "href"},
		{"object", "data"},
		{"embed", "src"},
		{"iframe", "src"},
		{"source", "src"},
		{"img", "src"},
	}
	fallback := ""
	for _, target := range targets {
		for _, node := range findElements(page, target.tag) {
			raw := strings.ReplaceAll(attribute(node, target.attr), "&amp;", "&")
			if !strings.Contains(raw, "pluginfile.php") {
				continue
			}
			ref, err := url.Parse(raw)
			if err != nil {
				continue
			}
			resolved := base.ResolveReference(ref).String()
			if strings.Contains(resolved, "/mod_resource/content/") {
				return resolved
			}
			if fallback == "" {
				fallback = resolved
			}
		}
	}
	return fallback
}
