package kulon2

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func cloneClientWithCookie(client *http.Client, base *url.URL, cookie string, timeout time.Duration) *http.Client {
	jar, _ := cookiejar.New(nil)
	jar.SetCookies(base, []*http.Cookie{{
		Name: "MoodleSession", Value: cookie, Path: "/", HttpOnly: true,
	}})
	if client == nil {
		return &http.Client{Jar: jar, Timeout: timeout}
	}
	copyClient := *client
	copyClient.Jar = jar
	if copyClient.Timeout == 0 {
		copyClient.Timeout = timeout
	}
	return &copyClient
}

type httpStatusError struct {
	statusCode int
	retryAfter time.Duration
}

func (e *httpStatusError) Error() string { return fmt.Sprintf("HTTP %d", e.statusCode) }

func getPageWithRetry(ctx context.Context, rawURL, cookie string, client *http.Client, wait func(context.Context) error, sleep func(context.Context, time.Duration) error) ([]byte, error) {
	const maxAttempts = 3
	for attempt := 1; ; attempt++ {
		if wait != nil {
			if err := wait(ctx); err != nil {
				return nil, err
			}
		}
		body, err := getPage(ctx, rawURL, cookie, client)
		var statusErr *httpStatusError
		if !errors.As(err, &statusErr) || (statusErr.statusCode != http.StatusTooManyRequests && statusErr.statusCode < 500) || attempt >= maxAttempts {
			return body, err
		}
		delay := statusErr.retryAfter
		if delay <= 0 {
			delay = time.Duration(attempt) * 250 * time.Millisecond
		}
		if delay > 5*time.Second {
			delay = 5 * time.Second
		}
		if sleep != nil {
			if err := sleep(ctx, delay); err != nil {
				return nil, err
			}
		} else {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
			}
		}
	}
}

func retryAfterDelay(value string, now time.Time) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if date, err := http.ParseTime(value); err == nil {
		if delay := date.Sub(now); delay > 0 {
			return delay
		}
	}
	return 0
}

func getPage(ctx context.Context, rawURL, cookie string, client *http.Client) ([]byte, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	base, err := url.Parse(parsedURL.Scheme + "://" + parsedURL.Host)
	if err != nil {
		return nil, err
	}
	httpClient := cloneClientWithCookie(client, base, cookie, 30*time.Second)
	if httpClient.CheckRedirect == nil {
		httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) > 0 && req.URL.Host != via[0].URL.Host {
				return http.ErrUseLastResponse
			}
			return nil
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "kamps/0.1 (Kulon2 client; polite requests)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.Request.URL.Host != parsedURL.Host || resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return nil, ErrSessionExpired
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			return nil, &httpStatusError{statusCode: resp.StatusCode, retryAfter: retryAfterDelay(resp.Header.Get("Retry-After"), time.Now())}
		}
		return nil, fmt.Errorf("GET course page: %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 12<<20))
	if err != nil {
		return nil, err
	}
	if strings.Contains(strings.ToLower(string(body)), "/login/index.php") {
		return nil, ErrSessionExpired
	}
	return body, nil
}
