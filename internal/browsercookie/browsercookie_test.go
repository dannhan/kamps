package browsercookie

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func encryptChromium(t *testing.T, password, plaintext string) []byte {
	t.Helper()
	key := pbkdf2SHA1([]byte(password), []byte("saltysalt"), 1, 16)
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	padded := []byte(plaintext)
	padding := aes.BlockSize - len(padded)%aes.BlockSize
	padded = append(padded, bytes.Repeat([]byte{byte(padding)}, padding)...)
	out := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, bytes.Repeat([]byte{' '}, aes.BlockSize)).CryptBlocks(out, padded)
	return append([]byte("v10"), out...)
}

func TestDecryptChromiumWithFallbackPassword(t *testing.T) {
	old := keyringPassword
	keyringPassword = func(string, string) (string, error) { return "", errNoKey }
	t.Cleanup(func() { keyringPassword = old })

	blob := encryptChromium(t, "peanuts", "session-value")
	got, err := decryptChromium(blob, "google-chrome")
	if err != nil {
		t.Fatal(err)
	}
	if got != "session-value" {
		t.Fatalf("decrypted = %q", got)
	}
}

func TestFindFirefoxCookie(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	profile := filepath.Join(home, ".mozilla", "firefox", "abc.default")
	if err := os.MkdirAll(profile, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(profile, "cookies.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE moz_cookies (host TEXT, name TEXT, value TEXT, expiry INTEGER)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO moz_cookies VALUES (?, ?, ?, ?)`,
		".undip.ac.id", "MoodleSession", "firefox-value", time.Now().Add(time.Hour).Unix()); err != nil {
		t.Fatal(err)
	}
	db.Close()

	cookie, err := Find("https://kulon2.undip.ac.id/", "firefox")
	if err != nil {
		t.Fatal(err)
	}
	if cookie.Value != "firefox-value" || cookie.Browser != "firefox" {
		t.Fatalf("cookie = %+v", cookie)
	}
}
