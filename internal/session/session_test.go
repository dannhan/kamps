package session

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	want := Session{
		SiteURL: "https://example.test",
		Cookie:  "abc123",
		UserID:  42,
		SessKey: "sess",
	}
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("mode = %o, want 600", mode)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Cookie != want.Cookie || got.UserID != want.UserID || got.SessKey != want.SessKey {
		t.Fatalf("round trip = %+v, want %+v", got, want)
	}
	if got.SavedAt.IsZero() {
		t.Fatal("SavedAt was not set")
	}

	removed, err := Delete(path)
	if err != nil {
		t.Fatal(err)
	}
	if !removed {
		t.Fatal("Delete did not report the existing file")
	}
	if _, err := Load(path); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Load after Delete = %v, want ErrNotFound", err)
	}
	removed, err = Delete(path)
	if err != nil || removed {
		t.Fatalf("second Delete = %v, %v; want false, nil", removed, err)
	}
}

func TestSaveRejectsEmptyCookie(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	if err := Save(path, Session{}); err == nil {
		t.Fatal("expected an empty-cookie error")
	}
}

func TestLoadRejectsCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected a parse error")
	}
}
