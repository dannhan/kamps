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

Requires Go 1.26+.

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
kamps login --browser  # extract the cookie from a local Firefox/Chromium profile
./kamps logout       # remove the saved session
./kamps workspace init  # print a starter course.json (does not write it)
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

Live mode parses in-progress courses, course pages, assignments, resources, assignment attachments, and folder files. Materials are grouped as `week-0`, `week-1`, … from the course section order (an explicit “Week N” name wins), and downloads go to `library_dir` and resume after interruption. Course rows use the cleaned course name and code instead of Moodle’s category prefixes. An optional `course.json` in a workspace maps a course to that repo so downloads land there; see [`docs/workspace.md`](docs/workspace.md). Use `./kamps --demo` to explore the bundled UI without a session.

Handy keys:

| key | action |
|---|---|
| `↑↓` / `jk` | move |
| `hl` | fold / open (courses, sections, folders, task buckets) |
| `tab`, `1`–`4` | switch tabs |
| `enter` | open / close the detail pane (Tasks, Materials); write the queue (Commit) |
| `space`, `a` | select item / select all |
| `V`, `*` | range-select / invert visible |
| `ctrl+z` | undo the last bulk triage |
| `t` / `m` / `i` / `u` | decide: track / missed / ignore / undecided |
| `f` | cycle the filter (Materials: all / new / downloaded / link / unknown) |
| `/` | search the tree (composes with the filter) |
| `esc` | clear the search, then the filter, then drop back |
| `c` | review & commit |
| `w` | write the queue to `commit_file` (Commit) |
| `y` | copy the queue lines (Commit) |
| `x` / `X` | drop one item / drop all queued (Commit) |
| `s` | reload the source |
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
cmd/kamps/          entry point, flags, login, source registration
internal/model/     Course, Assignment, Material — data types
internal/source/    Source interface, registry, capabilities, demo data
internal/kulon2/    live Moodle parser, session, downloads
internal/library/   safe paths, resumable downloads, library scan
internal/store/     state schema v2, migration, locking/merge
internal/cache/     last-successful snapshot for offline mode
internal/export/    Markdown, todo.txt, Taskwarrior writers
internal/workspace/ optional course.json discovery and resolution
internal/browsercookie/ Firefox/Chromium cookie extraction
internal/tui/       the terminal UI
internal/config/    paths and settings
docs/               developer guides and platform notes
```

Everything is built against one interface, `source.Source`. Choose the bundled
demo dataset with `--demo`, a registered source with `--source`, or the live
Kulon2 source after logging in. Course pages supply activity names, sections, assignments, downloadable resources, assignment attachments, and folder children. Downloaded files are reconciled with the local library on startup, and state writes are merged under a cross-process lock.

See [`docs/export.md`](docs/export.md), [`docs/workspace.md`](docs/workspace.md), and [`docs/plugins.md`](docs/plugins.md) for the new surfaces.

## Status & roadmap

The core roadmap is implemented: live parsing/downloads, resumable transfers, library scan, state locking, real sync, auto-triage, session expiry, offline cache, browser cookie login, exporters, workspace mapping, the `workspace init` template helper, quiz deadlines/status, page links, and the source registry. Next up, in order:

1. Parse quiz questions and more module-specific attachments.
2. Add optional Todoist/out-of-process source integrations.
3. Polish offline cache retention and Library views for files kamps did not download.

Implemented in this pass: assignment attachments, folder files, resumable downloads, download retries/concurrency, startup library scan, state locking/merge, real sync, auto-triage of submitted work, session expiry, offline cache, browser cookie login, Markdown/todo.txt/Taskwarrior exporters, `course.json` workspace mapping, source registry, and range/bulk selection.
