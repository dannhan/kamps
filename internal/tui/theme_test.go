package tui

import "testing"

func TestQuietTreeLineColorTracksForegroundAndBackground(t *testing.T) {
	for _, pair := range [][2]string{{"#1B1B26", "#E4E4E4"}, {"#EDEDED", "#333355"}} {
		got := quietHex(pair[0], pair[1])
		if got == pair[0] || got == pair[1] || len(got) != 7 {
			t.Fatalf("quietHex(%q, %q) = %q", pair[0], pair[1], got)
		}
	}
	if got := quietHex("bad", "#000000"); got != "#8A8A8A" {
		t.Fatalf("invalid colors fallback = %q", got)
	}
}
