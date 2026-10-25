// Package browsercookie reads the MoodleSession value from local Firefox and
// Chromium-family profiles on Linux. It never prints the cookie.
package browsercookie

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Cookie is one extracted browser session.
type Cookie struct {
	Value     string
	ExpiresAt time.Time
	Browser   string
	Profile   string
}

// Find looks for a MoodleSession cookie for siteURL. mode is "auto",
// "firefox", or "chromium".
func Find(siteURL, mode string) (Cookie, error) {
	parsed, err := url.Parse(siteURL)
	if err != nil || parsed.Host == "" {
		return Cookie{}, fmt.Errorf("invalid site URL %q", siteURL)
	}
	host := strings.Split(parsed.Host, ":")[0]
	if host == "" {
		return Cookie{}, errors.New("site URL has no host")
	}

	var found []Cookie
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "auto":
		found = append(found, firefoxCookies(host)...)
		found = append(found, chromiumCookies(host)...)
	case "firefox":
		found = firefoxCookies(host)
	case "chromium", "chrome":
		found = chromiumCookies(host)
	default:
		return Cookie{}, fmt.Errorf("unknown browser %q (use auto, firefox, or chromium)", mode)
	}
	if len(found) == 0 {
		return Cookie{}, errors.New("no MoodleSession cookie found in local browser profiles")
	}
	best := found[0]
	for _, candidate := range found[1:] {
		if candidate.ExpiresAt.After(best.ExpiresAt) {
			best = candidate
		}
	}
	return best, nil
}

func hostMatches(cookieHost, host string) bool {
	cookieHost = strings.ToLower(strings.TrimSpace(cookieHost))
	host = strings.ToLower(host)
	cookieHost = strings.TrimPrefix(cookieHost, ".")
	if cookieHost == "" {
		return false
	}
	return cookieHost == host || strings.HasSuffix(host, "."+cookieHost)
}

func chromeExpiry(microseconds int64) time.Time {
	if microseconds <= 0 {
		return time.Time{}
	}
	// Chromium stores microseconds since 1601-01-01 UTC.
	return time.Date(1601, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(microseconds) * time.Microsecond)
}
