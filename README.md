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
./kamps              # launch live mode (requires login)
./kamps --demo       # launch using bundled demo data
./kamps login        # sign in through your browser, then paste MoodleSession
./kamps logout       # remove the saved session
./kamps version      # print the version
kamps | less         # no TTY? prints a plain-text snapshot instead
```

Login requires `site_url` in `~/.config/kamps/config.toml` (or the XDG config
path). The default is Kulon2. Kamps does not automate UNDIP SSO: sign in with
your browser, copy the `MoodleSession` cookie value, and paste it at the hidden
prompt. The verified session is stored privately at
`$XDG_STATE_HOME/kamps/session.json` (normally `~/.local/state/kamps/session.json`);
`logout` removes it.

The default material library is `~/Documents/kamps`. You can override it:

```toml
# ~/.config/kamps/config.toml
site_url = "https://kulon2.undip.ac.id/"
library_dir = "~/projects/university"
```

The `~/projects/university` mapping is **not implemented yet**. Live mode shows the in-progress course list, parses course pages into assignments and materials, and downloads resource files into the library directory. Assignment attachments, folder modules, and resume-after-interruption are not available yet. Use `./kamps --demo` for the complete bundled UI.

Handy keys:

| key | action |
|---|---|
| `↑↓` / `jk` | move |
| `hl` | fold / open (courses, weeks) |
| `tab`, `1`–`5` | switch tabs |
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
| `o` | open an external link in the browser (Materials) |
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
internal/source/  Source interface, live source, and demo data source
internal/store/   local state: decisions and download flags
internal/tui/     the terminal UI
internal/config/  paths and settings
docs/             developer guides and Kulon2 notes
```

Everything is built against one interface, `source.Source`. Choose the bundled
demo dataset with `--demo`, or use the live Kulon2 course-list source after
logging in. Course pages supply activity names, sections, assignments, and downloadable resources; assignment attachment parsing and folder downloads remain future work.

## Status & roadmap

Early days: the UI is complete and runs against demo data; live Kulon2 mode loads courses, parses course pages, and downloads resource files into the library directory. Next up, in order:

1. Parse assignment attachments and folder modules as downloadable materials. See `docs/kulon2.md`.
2. Optionally map LMS courses to `course.json` in a user's workspace, without requiring that layout.
3. Resume interrupted downloads and guard `state.json` against multiple running instances.
4. Add a real commit writer behind the queue preview (today `y` copies the lines).

Later, not now: bulk actions and a plugin surface for new sources.
