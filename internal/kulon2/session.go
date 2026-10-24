// Package kulon2 talks to the Kulon2 Moodle instance. For now it only verifies
// a manually supplied session cookie; the read-only source will grow here.
package kulon2

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ErrSessionExpired means the LMS redirected to a login page or otherwise did
// not accept the cookie.
var ErrSessionExpired = errors.New("session expired")

var (
	sessKeyPattern = regexp.MustCompile(`"sesskey"\s*:\s*"([^"]+)"`)
	userIDPattern  = regexp.MustCompile(`"userid"\s*:\s*"?([0-9]+)"?`)
	loginPattern   = regexp.MustCompile(`(?i)(/login/index\.php|/auth/oidc)`)
)

// Identity is the small set of values Kamps needs from an authenticated page.
type Identity struct {
	UserID  int
	SessKey string
}

// VerifySession checks a MoodleSession cookie against /my/. It returns
// ErrSessionExpired when the site treats the cookie as logged out.
func VerifySession(ctx context.Context, siteURL, cookie string) (Identity, error) {
	if strings.TrimSpace(cookie) == "" {
		return Identity{}, errors.New("empty MoodleSession cookie")
	}
	base, err := parseSiteURL(siteURL)
	if err != nil {
		return Identity{}, err
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return Identity{}, err
	}
	jar.SetCookies(base, []*http.Cookie{{
		Name:     "MoodleSession",
		Value:    cookie,
		Path:     "/",
		HttpOnly: true,
	}})

	client := &http.Client{Jar: jar, Timeout: 20 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base.String(), "/")+"/my/", nil)
	if err != nil {
		return Identity{}, err
	}
	req.Header.Set("User-Agent", "kamps/0.1")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := client.Do(req)
	if err != nil {
		return Identity{}, fmt.Errorf("verify session: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return Identity{}, fmt.Errorf("read /my/: %w", err)
	}
	if isLoginPage(resp.Request.URL) {
		return Identity{}, ErrSessionExpired
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Identity{}, fmt.Errorf("GET /my/: %s", resp.Status)
	}
	if resp.Request.URL.Host != base.Host {
		return Identity{}, fmt.Errorf("session verification redirected to unexpected host %q", resp.Request.URL.Host)
	}

	sessKey := firstMatch(sessKeyPattern, body)
	if sessKey == "" {
		if loginPattern.Match(body) {
			return Identity{}, ErrSessionExpired
		}
		return Identity{}, errors.New("authenticated page has no sesskey")
	}

	identity := Identity{SessKey: sessKey}
	if raw := firstMatch(userIDPattern, body); raw != "" {
		identity.UserID, _ = strconv.Atoi(raw)
	}
	return identity, nil
}

func parseSiteURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("site_url is not configured")
	}
	base, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("site_url: %w", err)
	}
	if base.Scheme != "http" && base.Scheme != "https" {
		return nil, fmt.Errorf("site_url must start with http:// or https://")
	}
	if base.Host == "" {
		return nil, errors.New("site_url has no host")
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/"
	base.RawQuery = ""
	base.Fragment = ""
	return base, nil
}

func isLoginPage(u *url.URL) bool {
	if u == nil {
		return false
	}
	path := strings.ToLower(u.Path)
	return strings.Contains(path, "/login/") || strings.Contains(path, "/auth/oidc")
}

func firstMatch(pattern *regexp.Regexp, body []byte) string {
	match := pattern.FindSubmatch(body)
	if len(match) < 2 {
		return ""
	}
	return string(match[1])
}
