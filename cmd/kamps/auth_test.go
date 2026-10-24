package main

import (
	"os"
	"strings"
	"testing"
)

func TestPromptSecretReadsPipedInput(t *testing.T) {
	input, err := os.CreateTemp(t.TempDir(), "secret-input")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := input.WriteString("  secret-value\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := input.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	var prompt strings.Builder
	got, err := promptSecret(input, &prompt, "Cookie: ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "secret-value" {
		t.Fatalf("secret = %q", got)
	}
	if !strings.Contains(prompt.String(), "Cookie: ") {
		t.Fatalf("prompt was not written: %q", prompt.String())
	}
}
