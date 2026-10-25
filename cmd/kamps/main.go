// Command kamps is a fast, cheerful terminal UI for your college coursework:
// courses, assignments and materials pulled from an LMS (Moodle/Kulon2) plus
// lab drives, synced and stored locally.
//
// The UI can run against bundled demo data or a live Kulon2 course list.
// `kamps login` stores a verified session cookie for live mode.
package main

import (
	"fmt"
	"os"
	"runtime/debug"

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
	opts, rest, err := parseOptions(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "kamps:", err)
		os.Exit(1)
	}
	demoMode := opts.demo

	if len(rest) > 0 {
		switch rest[0] {
		case "version", "--version", "-v":
			fmt.Println(versionLine())
			return
		case "help", "--help", "-h":
			printHelp()
			return
		case "workspace":
			if len(rest) > 1 && rest[1] == "init" {
				opts, err := parseWorkspaceInit(rest[2:])
				if err != nil {
					fmt.Fprintln(os.Stderr, "kamps:", err)
					os.Exit(1)
				}
				if err := printWorkspaceTemplate(os.Stdout, opts); err != nil {
					fmt.Fprintln(os.Stderr, "kamps:", err)
					os.Exit(1)
				}
				return
			}
			fmt.Fprintln(os.Stderr, "usage: kamps workspace init [--lms-id N] [--shortname S] [--path P]")
			os.Exit(1)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "kamps:", err)
		os.Exit(1)
	}
	if err := opts.apply(&cfg); err != nil {
		fmt.Fprintln(os.Stderr, "kamps:", err)
		os.Exit(1)
	}

	if len(rest) > 0 {
		switch rest[0] {
		case "login":
			mode, useBrowser := browserMode(rest[1:])
			if err := loginWithBrowser(cfg, mode, useBrowser); err != nil {
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

	sourceName := cfg.Source
	if demoMode {
		sourceName = "demo"
	}
	if sourceName == "" {
		sourceName = "kulon2"
	}
	if sourceName != "demo" {
		if err := cfg.EnsureLibraryDir(); err != nil {
			fmt.Fprintln(os.Stderr, "kamps: warning:", err)
		}
	}
	src, sourceErr := source.Open(sourceName, source.Options{
		SiteURL:     cfg.SiteURL,
		SessionPath: cfg.SessionPath(),
		StateDir:    cfg.StateDir,
		LibraryDir:  cfg.LibraryDir,
	})
	if sourceErr != nil {
		fmt.Fprintln(os.Stderr, "kamps:", sourceErr)
		os.Exit(1)
	}

	if handled, code := runHeadless(rest, cfg, src, os.Stdout, os.Stderr); handled {
		if code != 0 {
			os.Exit(code)
		}
		return
	}

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
	app := tui.New(src, cfg, version)
	p := tea.NewProgram(app, tea.WithAltScreen())
	app.Attach(p)
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
  kamps            launch live Kulon2 mode (requires a saved session)
  kamps --demo     launch using bundled demo data
  kamps login [--browser[=auto|firefox|chromium]]
                   verify and save a Moodle session cookie
  kamps logout     remove the saved session
  kamps workspace init [--lms-id N] [--shortname S] [--path P]
                   print a starter course.json (does not write it)
  kamps courses [--json]
  kamps tasks [--json] [--filter F] [--search Q] [--course C]
  kamps materials [--json] [--filter F] [--search Q] [--course C]
  kamps status [--json]
  kamps sync [--json]
  kamps triage <id...> --tracked|--missed|--ignored|--undecided [--json]
  kamps download <id...> | --all-new | --course C [--dry-run] [--json]
  kamps export --format markdown|todo|taskwarrior [--out PATH] [--json]
  kamps version    print the version
  kamps help       print this help

FLAGS
  --demo                        use bundled demo data
  --source NAME                 select an in-tree source (demo, kulon2)
  --commit-file PATH            default export target for the Commit tab
  --export-format FORMAT        markdown, todo, or taskwarrior
  --no-anim                     no spinner or progress motion

CONFIG
  ~/.config/kamps/config.toml
    site_url = "https://kulon2.undip.ac.id/"
    library_dir = "~/Documents/kamps"
    source = "kulon2"
    workspace_roots = ["~/projects/university"]
    commit_file = "~/notes/tasks.md"
    export_format = "markdown"
    auto_track_submitted = true

SCREENS
  1 Tasks       2 Materials       3 Sync       4 Commit
  c  open the commit queue
  enter  toggle the detail pane (Tasks, Materials); write the queue (Commit)
  /  search     f  filter     esc  clear search/filter/selection

KEYS
  ↑/k ↓/j    move            space   select
  ←/h →/l    fold / open     a       select all
  V          range select    *       invert visible
  ctrl+z     undo triage
  t          track           m       missed
  i          ignore          u       undecided
  c          review & commit w       write queue
  y          copy queue      x       drop from queue
  X          drop all queued
  d          download        s       sync
  o          open link       ?       help
  q          quit

Live mode parses courses, assignments, resources, assignment attachments, and
folder files; downloads go to library_dir or a mapped course.json workspace.
Use --demo to see the bundled example without a session.
`)
}

func versionLine() string {
	line := "kamps " + version
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return line
	}
	var revision, built, modified string
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.time":
			built = setting.Value
		case "vcs.modified":
			if setting.Value == "true" {
				modified = "-dirty"
			}
		}
	}
	if len(revision) > 12 {
		revision = revision[:12]
	}
	if revision != "" {
		line += " · " + revision + modified
	}
	if built != "" {
		line += " · " + built
	}
	return line
}
