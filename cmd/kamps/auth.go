package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	xterm "github.com/charmbracelet/x/term"

	"github.com/ruyam/kamps/internal/browsercookie"
	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/kulon2"
	"github.com/ruyam/kamps/internal/session"
)

// loginWithBrowser tries browser cookie extraction when requested, then falls
// back to the manual hidden prompt.
func loginWithBrowser(cfg config.Config, mode string, useBrowser bool) error {
	if !useBrowser {
		return login(cfg)
	}
	cookie, err := browsercookie.Find(cfg.SiteURL, mode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Browser cookie lookup failed: %v\n", err)
		fmt.Fprintln(os.Stderr, "Falling back to manual paste.")
		return login(cfg)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	identity, err := kulon2.VerifySession(ctx, cfg.SiteURL, cookie.Value)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Browser cookie was rejected: %v\n", err)
		fmt.Fprintln(os.Stderr, "Falling back to manual paste.")
		return login(cfg)
	}
	expires := cookie.ExpiresAt
	if expires.IsZero() {
		lifetime := identity.Timeout
		if lifetime <= 0 {
			lifetime = session.DefaultLifetime
		}
		expires = time.Now().Add(lifetime)
	}
	if err := saveSession(cfg, cookie.Value, identity, expires); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "✓ signed in as user %d via %s\n", identity.UserID, cookie.Browser)
	fmt.Fprintf(os.Stdout, "Session saved to %s\n", cfg.SessionPath())
	return nil
}

func saveSession(cfg config.Config, cookie string, identity kulon2.Identity, expires time.Time) error {
	saved := session.Session{
		SiteURL:   cfg.SiteURL,
		Cookie:    cookie,
		UserID:    identity.UserID,
		SessKey:   identity.SessKey,
		ExpiresAt: expires,
	}
	if err := session.Save(cfg.SessionPath(), saved); err != nil {
		return fmt.Errorf("save session: %w", err)
	}
	return nil
}

// login opens the LMS in a browser, asks for the MoodleSession cookie, verifies
// it, and saves the resulting session. The cookie is never written to output.
func login(cfg config.Config) error {
	if cfg.SiteURL == "" {
		return fmt.Errorf("site_url is not set in %s", cfg.ConfigPath())
	}
	if cfg.SessionPath() == "" {
		return errors.New("session path is not configured")
	}

	loginURL := strings.TrimRight(cfg.SiteURL, "/") + "/my/"
	fmt.Fprintf(os.Stdout, "Opening %s\n", loginURL)
	if err := openBrowser(loginURL); err != nil {
		fmt.Fprintf(os.Stderr, "Could not open a browser: %v\n", err)
		fmt.Fprintln(os.Stderr, "Open the URL above manually.")
	}
	fmt.Fprintln(os.Stdout, "Sign in, then copy the MoodleSession cookie value from your browser.")
	fmt.Fprintln(os.Stdout, "DevTools → Application → Cookies → the LMS domain → MoodleSession.")

	cookie, err := promptSecret(os.Stdin, os.Stderr, "MoodleSession cookie: ")
	if err != nil {
		return err
	}
	if len(cookie) > 4096 {
		return errors.New("MoodleSession cookie value is unexpectedly long")
	}
	if cookie == "" {
		return errors.New("empty MoodleSession cookie")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	identity, err := kulon2.VerifySession(ctx, cfg.SiteURL, cookie)
	if errors.Is(err, kulon2.ErrSessionExpired) {
		return errors.New("the LMS rejected that cookie; copy a fresh MoodleSession value")
	}
	if err != nil {
		return err
	}

	lifetime := identity.Timeout
	if lifetime <= 0 {
		lifetime = session.DefaultLifetime
	}
	if err := saveSession(cfg, cookie, identity, time.Now().Add(lifetime)); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "✓ signed in as user %d\n", identity.UserID)
	fmt.Fprintf(os.Stdout, "Session saved to %s\n", cfg.SessionPath())
	return nil
}

// logout removes the saved session, if any.
func logout(cfg config.Config) error {
	removed, err := session.Delete(cfg.SessionPath())
	if err != nil {
		return fmt.Errorf("remove session: %w", err)
	}
	if !removed {
		fmt.Fprintln(os.Stdout, "No saved session.")
		return nil
	}
	fmt.Fprintln(os.Stdout, "Session removed.")
	return nil
}

// promptSecret reads a secret without echoing it when stdin is a terminal.
// Piped input is accepted for scripts, but the prompt still goes to stderr.
func promptSecret(in *os.File, errOut io.Writer, label string) (string, error) {
	fmt.Fprint(errOut, label)
	defer fmt.Fprintln(errOut)

	if in == nil {
		return "", errors.New("stdin is not available")
	}
	if xterm.IsTerminal(in.Fd()) {
		data, err := xterm.ReadPassword(in.Fd())
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(data)), nil
	}
	data, err := io.ReadAll(io.LimitReader(in, 4097))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
