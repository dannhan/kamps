// Command kamps is a fast, cheerful terminal UI for your college coursework:
// courses, assignments and materials pulled from an LMS (Moodle/Kulon2) plus
// lab drives, synced and stored locally.
//
// The UI can run against bundled demo data. `kamps login` stores a verified
// Kulon2 session for the next source integration step.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	xterm "github.com/charmbracelet/x/term"

	"github.com/ruyam/kamps/internal/config"
	"github.com/ruyam/kamps/internal/plain"
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

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "kamps:", err)
		os.Exit(1)
	}
	cfg.NoAnim = noAnim

	if len(rest) > 0 {
		switch rest[0] {
		case "login":
			if err := login(cfg); err != nil {
				fmt.Fprintln(os.Stderr, "kamps:", err)
				os.Exit(1)
			}
			return
		case "logout":
			if err := logout(cfg); err != nil {
				fmt.Fprintln(os.Stderr, "kamps:", err)
				os.Exit(1)
			}
			return
		}
	}

	src := source.NewDemo()

	// Non-interactive fallback: piping `kamps | less` should still work.
	if !isTTY(os.Stdout) {
		if err := plain.PrintPlain(os.Stdout, src, cfg); err != nil {
			fmt.Fprintln(os.Stderr, "kamps:", err)
			os.Exit(1)
		}
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
  kamps login      verify and save a Moodle session cookie
  kamps logout     remove the saved session
  kamps version    print the version
  kamps help       print this help

FLAGS
  --no-anim        no spinner or progress motion

CONFIG
  ~/.config/kamps/config.toml
    site_url = "https://kulon2.undip.ac.id/"
    library_dir = "~/Documents/kamps"

SCREENS
  1 Tasks       2 Materials       3 Sync       4 Commit
  c  open the commit queue
  enter  toggle the detail pane (Tasks, Materials); copy the queue (Commit)
  /  search     f  filter     esc  clear search/filter

KEYS
  ↑/k ↓/j    move            space   select
  ←/h →/l    fold / open     a       select all
  t          track           m       missed
  i          ignore          u       undecided
  c          review & commit y       copy queue lines
  x          drop from queue s       sync
  d          download        /       search
  f          filter          ?       help
  q          quit

Demo data is bundled for now; login is prepared, but Moodle coursework loading is not implemented.
`)
}
