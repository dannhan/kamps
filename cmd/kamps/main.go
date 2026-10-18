// Command kamps is a fast, cheerful terminal UI for your college coursework:
// courses, assignments and materials pulled from an LMS (Moodle/Kulon2) plus
// lab drives, synced and stored locally.
//
// This build ships with a demo source so the UI can be used before any
// networking exists. The real Moodle source implements the same interface and
// slots in without touching the UI.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	xterm "github.com/charmbracelet/x/term"

	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/source"
	"github.com/ruyam/kamps/internal/tui"
)

// version is overridable at build time with -ldflags "-X main.version=...".
var version = "0.1.0-dev"

func main() {
	args := os.Args[1:]
	noAnim := false
	var rest []string
	for _, a := range args {
		if a == "--no-anim" || a == "--noanim" {
			noAnim = true
			continue
		}
		rest = append(rest, a)
	}

	if len(rest) > 0 {
		switch rest[0] {
		case "version", "--version", "-v":
			fmt.Printf("kamps %s\n", version)
			return
		case "help", "--help", "-h":
			printHelp()
			return
		}
	}

	cfg := config.Default()
	cfg.NoAnim = noAnim
	src := source.NewDemo()

	// Non-interactive fallback: piping `kamps | less` should still work.
	if !isTTY(os.Stdout) {
		tui.PrintPlain(os.Stdout, src)
		return
	}

	// Full-screen: kamps takes over the terminal (ink-style) and restores your
	// shell exactly as it was on exit.
	p := tea.NewProgram(tui.New(src, cfg, version), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "kamps:", err)
		os.Exit(1)
	}
}

// isTTY reports whether f is attached to a terminal. A plain mode check is not
// enough: /dev/null is a character device too, and `kamps > /dev/null` must not
// try to take over a terminal.
func isTTY(f *os.File) bool {
	return xterm.IsTerminal(f.Fd())
}

func printHelp() {
	fmt.Print(`kamps — college coursework, in your terminal

USAGE
  kamps            launch the interactive UI
  kamps version    print the version
  kamps help       print this help

FLAGS
  --no-anim        no spinner or progress motion

SCREENS
  1 Tasks       2 Materials       3 Sync
  v  cycle the Tasks view: dashboard · list
  c  review & commit what you have decided

KEYS
  ↑/k ↓/j    move            space   select
  ←/h →/l    move (board)    a       select all
  t          track           m       missed
  i          ignore          u       undecided
  c          review & commit /       filter
  s          sync            d       download
  ?          help            q       quit

Demo data is bundled for now; the Moodle source is next.
`)
}
