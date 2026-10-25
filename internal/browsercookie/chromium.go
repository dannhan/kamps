package browsercookie

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/binary"
	"strings"

	"github.com/zalando/go-keyring"
)

// keyringPassword is a seam for tests.
var keyringPassword = func(service, user string) (string, error) {
	return keyring.Get(service, user)
}

func chromatogramService(browser string) (service, user string) {
	switch browser {
	case "chromium":
		return "Chromium Safe Storage", "Chromium"
	case "brave-browser":
		return "Brave Safe Storage", "Brave"
	case "microsoft-edge":
		return "Microsoft Edge Safe Storage", "Microsoft Edge"
	default:
		return "Chrome Safe Storage", "Chrome"
	}
}

func decryptChromium(blob []byte, browser string) (string, error) {
	if len(blob) < 3 {
		return string(blob), nil
	}
	prefix := string(blob[:3])
	if prefix != "v10" && prefix != "v11" {
		return string(blob), nil
	}
	service, user := chromatogramService(browser)
	password, err := keyringPassword(service, user)
	if err != nil || password == "" {
		password = "peanuts"
	}
	key := pbkdf2SHA1([]byte(password), []byte("saltysalt"), 1, 16)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	iv := bytes.Repeat([]byte{' '}, aes.BlockSize)
	ciphertext := blob[3:]
	if len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return "", errNoKey
	}
	plain := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plain, ciphertext)
	if len(plain) > 0 {
		padding := int(plain[len(plain)-1])
		if padding > 0 && padding <= aes.BlockSize && padding <= len(plain) {
			plain = plain[:len(plain)-padding]
		}
	}
	return strings.TrimSpace(string(plain)), nil
}

// pbkdf2SHA1 is PBKDF2-HMAC-SHA1, matching Chromium's Linux key derivation.
func pbkdf2SHA1(password, salt []byte, iterations, keyLen int) []byte {
	prf := hmac.New(sha1.New, password)
	hashLen := prf.Size()
	blocks := (keyLen + hashLen - 1) / hashLen
	var out []byte
	for block := 1; block <= blocks; block++ {
		prf.Reset()
		prf.Write(salt)
		var counter [4]byte
		binary.BigEndian.PutUint32(counter[:], uint32(block))
		prf.Write(counter[:])
		u := prf.Sum(nil)
		t := make([]byte, len(u))
		copy(t, u)
		for i := 1; i < iterations; i++ {
			prf.Reset()
			prf.Write(u)
			u = prf.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		out = append(out, t...)
	}
	return out[:keyLen]
}
