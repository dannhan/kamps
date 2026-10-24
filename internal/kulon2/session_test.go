package kulon2

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVerifySession(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/my/" {
			http.NotFound(w, r)
			return
		}
		cookie, err := r.Cookie("MoodleSession")
		if err != nil || cookie.Value != "cookie-value" {
			http.Error(w, "missing cookie", http.StatusUnauthorized)
			return
		}
		fmt.Fprint(w, `<script>M.cfg = {"sesskey":"key-123","userId":42};</script>`)
	}))
	defer srv.Close()

	identity, err := VerifySession(context.Background(), srv.URL, "cookie-value")
	if err != nil {
		t.Fatal(err)
	}
	if identity.UserID != 42 || identity.SessKey != "key-123" {
		t.Fatalf("identity = %+v", identity)
	}
}

func TestVerifySessionExpired(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/my/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login/index.php", http.StatusFound)
	})
	mux.HandleFunc("/login/index.php", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<form action="/login/index.php">login</form>`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	_, err := VerifySession(context.Background(), srv.URL, "stale")
	if !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("error = %v, want ErrSessionExpired", err)
	}
}

func TestVerifySessionRejectsMissingSesskey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body>unexpected page</body></html>`)
	}))
	defer srv.Close()

	if _, err := VerifySession(context.Background(), srv.URL, "cookie"); err == nil {
		t.Fatal("expected a missing-sesskey error")
	}
}

func TestVerifySessionRejectsBadSiteURL(t *testing.T) {
	if _, err := VerifySession(context.Background(), "kulon2.undip.ac.id", "cookie"); err == nil {
		t.Fatal("expected a site_url error")
	}
}
