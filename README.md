# kamps

`kamps` is a terminal application that pulls your college coursework — courses,
assignments and materials — from your LMS (Moodle / Kulon2) and lab drives, and
helps you keep on top of it. Instead of a browser full of tabs and a spreadsheet
of deadlines you never open, everything lands in one place.

For each item you make a decision, and it is remembered:

| decision | means |
|---|---|
| `undecided` | you have not looked at it yet (the default) |
| `tracked` | handled — submitted, a teammate submitted it, or it is tracked elsewhere |
| `missed` | the deadline passed and it is gone |
| `ignored` | not real work — a template, last year's file, an optional extra |

Decisions and download flags are written atomically to
`$XDG_STATE_HOME/kamps/state.json` (`~/.local/state/kamps/state.json` when the
variable is unset) and reapplied on the next launch.

Your actual to-do list lives wherever you keep it; `kamps` can hand decided
items to it via the review & commit queue (`c`), which shows the exact lines
before anything is written. Decisions are saved as soon as you make them; the
queue is the export step. Today `y` copies those lines to the terminal
clipboard (OSC 52), so you can paste them into whatever list you use.

## Installation

Requires Go 1.24+.

```sh
git clone <this repo> kamps
cd kamps
go build -o kamps ./cmd/kamps
```

## Usage

```sh
./kamps              # launch the app
./kamps version      # print the version
kamps | less         # no TTY? prints a plain-text snapshot instead
```

Handy keys:

| key | action |
|---|---|
| `↑↓` / `jk` | move |
| `hl` | fold / open (courses, weeks) |
| `tab`, `1`–`4` | switch tabs |
| `enter` | open / close the detail pane (Tasks, Materials); copy the queue (Commit) |
| `space`, `a` | select item / select all |
| `t` / `m` / `i` / `u` | decide: track / missed / ignore / undecided |
| `f` | cycle the filter |
| `/` | search the tree (composes with the filter) |
| `esc` | clear the search, then the filter, then drop back |
| `c` | review & commit |
| `y` | copy the queue lines (Commit) |
| `x` | drop an item from the queue (Commit) |
| `s` | sync |
| `d` | download (Materials) |
| `?` | help |
| `q` | quit |

The detail pane follows the cursor and opens on its own when the terminal is
wide enough; `enter` pins it open or closed either way. Search narrows both
trees as you type, and `enter` leaves the query on while you move through the
matches.

## Project structure

```
cmd/kamps/        entry point
internal/model/   Course, Assignment, Material — data types
internal/source/  the Source interface + the demo data source
internal/store/   local state: decisions and download flags
internal/tui/     the terminal UI
internal/config/  paths and settings
docs/kulon2.md    notes on the Moodle/Kulon2 instance
```

Everything is built against one interface, `source.Source`. Today it is filled
by a bundled demo dataset; a real Moodle client fills it later without changing
anything else.

## Status & roadmap

Early days: the UI is complete and runs against demo data. Next up, in order:

1. Real Moodle source: cookie auth, course list, HTML parsing. See `docs/kulon2.md`.
2. A real commit writer behind the queue preview (today `y` copies the lines).
3. Downloads with progress and a configurable library directory.

Later, not now: bulk actions and a plugin surface for new sources.
