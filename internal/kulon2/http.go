package kulon2

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
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
