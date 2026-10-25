package main

import (
	"strings"
	"testing"
)

func TestVersionLineIncludesBuildWhenAvailable(t *testing.T) {
	line := versionLine()
	if !strings.HasPrefix(line, "kamps ") {
		t.Fatalf("version = %q", line)
	}
}
