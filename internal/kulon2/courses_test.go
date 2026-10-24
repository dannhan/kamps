package kulon2

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListCoursesPostsExpectedRequestAndMapsCourses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/lib/ajax/service.php" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("sesskey") != "fixture-key" || r.URL.Query().Get("info") != courseListMethod {
			t.Errorf("query = %v", r.URL.Query())
		}
		cookie, err := r.Cookie("MoodleSession")
		if err != nil || cookie.Value != "fixture-cookie" {
			t.Errorf("MoodleSession cookie = %v, err = %v", cookie, err)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		var calls []struct {
			Index      int            `json:"index"`
			MethodName string         `json:"methodname"`
			Args       map[string]any `json:"args"`
		}
		if err := json.Unmarshal(body, &calls); err != nil {
			t.Fatal(err)
		}
		if len(calls) != 1 || calls[0].MethodName != courseListMethod {
			t.Errorf("request body = %s", body)
		} else {
			args := calls[0].Args
			if args["classification"] != "inprogress" || args["sort"] != "fullname" || args["offset"] != float64(0) || args["limit"] != float64(0) {
				t.Errorf("request args = %#v", args)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `[{"data":[{"id":17638,"shortname":"MIK1624302","fullname":"Struktur Data B"},{"id":17964,"shortname":"MIK1624303","fullname":"Basis Data C"}]}]`)
	}))
	defer srv.Close()

	courses, err := ListCourses(context.Background(), srv.URL, "fixture-cookie", "fixture-key")
	if err != nil {
		t.Fatal(err)
	}
	if len(courses) != 2 || courses[0].ID != 17638 || courses[0].Name != "Struktur Data B" || courses[0].Code != "MIK1624302" {
		t.Fatalf("courses = %+v", courses)
	}
	if courses[0].Source != "lms" {
		t.Fatalf("source = %q, want lms", courses[0].Source)
	}
}

func TestListCoursesRejectsServiceErrorAndEmptyList(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{name: "service error", body: `[{"error":true,"exception":"moodle_exception","message":"denied"}]`},
		{name: "empty list", body: `[{"data":[]}]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				io.WriteString(w, tc.body)
			}))
			defer srv.Close()
			_, err := ListCourses(context.Background(), srv.URL, "cookie", "key")
			if err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestListCoursesRejectsSessionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `[{"error":true,"exception":"invalidsesskey","message":"session key error"}]`)
	}))
	defer srv.Close()
	_, err := ListCourses(context.Background(), srv.URL, "cookie", "key")
	if !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("error = %v, want ErrSessionExpired", err)
	}
}

func TestListCoursesRejectsLoginRedirect(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/lib/ajax/service.php", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login/index.php", http.StatusFound)
	})
	mux.HandleFunc("/login/index.php", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "login")
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	_, err := ListCourses(context.Background(), srv.URL, "cookie", "key")
	if !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("error = %v, want ErrSessionExpired", err)
	}
}

func TestDecodeCoursesRefusesMalformedItems(t *testing.T) {
	_, err := decodeCourses([]byte(`[{"data":{"courses":[{"id":0,"shortname":"x","fullname":""}]}}]`))
	if err == nil || !strings.Contains(err.Error(), "valid id or name") {
		t.Fatalf("error = %v", err)
	}
}
