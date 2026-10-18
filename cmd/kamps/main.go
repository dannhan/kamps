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

	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/source"
	"github.com/ruyam/kamps/internal/tui"
)

// version is overridable at build time with -ldflags "-X main.version=...".
var version = "0.1.0-dev"

func main() {
	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "version", "--version", "-v":
			fmt.Printf("kamps %s\n", version)
			return
		case "help", "--help", "-h":
			printHelp()
			return
		}
	}

	cfg := config.Default()
	src := source.NewDemo()

	// Non-interactive fallback: piping `kamps | less` should still work.
	if !isTTY(os.Stdout) {
		tui.PrintPlain(os.Stdout, src)
		return
	}

	p := tea.NewProgram(tui.New(src, cfg, version), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "kamps:", err)
		os.Exit(1)
	}
}

func isTTY(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func printHelp() {
	fmt.Print(`kamps — college coursework, in your terminal

USAGE
  kamps            launch the interactive UI
  kamps version    print the version
  kamps help       print this help

KEYS
  ↑/k ↓/j    move            tab/shift+tab  switch section
  space      select          d              download
  x          toggle done     s              sync
  ?          help            q              quit

Demodata is bundled for now; the Moodle source is next.
`)
}
