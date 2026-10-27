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
./kamps tasks --json # headless tasks for agents and scripts
./kamps status --json
./kamps triage K901017 --tracked
./kamps download --all-new --dry-run --json
./kamps export --format markdown --out ~/notes/tasks.md
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

The workspace has two primary views, Tasks (`1`) and Materials (`2`). Use the
course sidebar to choose All courses or one course. Below 100 columns, `Tab`
opens the course picker in the content area; Enter applies the choice. Wider
terminals keep the sidebar visible. `Tab` / `Shift+Tab` move between courses,
content, and the optional inspector.

Tasks open with **Needs attention**: up to five undecided tasks due today or
within seven calendar days, earliest first. Enter on “Show all N” expands the
shortlist. Each task shows its name, full course name, and readable deadline.
Elapsed deadlines appear in a separate, collapsed **Overdue** section, most
recent first. Below it, full course-name groups open to undecided work, with
tracked and closed work in separate collapsed subgroups. A task's marks and
decisions stay synchronized wherever it appears.

`f` opens filters; `/` searches the current view and course. Matching course
groups open automatically unless you explicitly fold them. Each view remembers
its query, filter, cursor, and marks. Attention includes all undecided work;
Upcoming includes only unelapsed deadlines within seven days; Closed means
missed or ignored. Tracked remains separate because it means handled elsewhere.
Decision keys stay visible in the Tasks footer; status messages appear above the
content instead of replacing those keys.

Materials opens All courses as a compact list of collapsed weeks, newest week
first, with file and course counts. Enter expands a week into course groups and
files. Select a course in the sidebar to browse its expanded week library.
Press `S` to reverse the week order; the header shows the active order, which is
remembered while switching views. Course accents match the sidebar and Tasks.
Enter opens file details, including any related task, or folds a group. Display labels are cleaned without changing source names or filenames.
The cursor bar and square batch marks are independent. Download states stay
beside filenames, moving below long titles on narrow terminals. Offline messages
and errors leave the action footer visible.

| key | action |
|---|---|
| `↑↓` / `jk` | move in the focused pane; scroll the inspector |
| `g` / `G`, `ctrl+d/u` | first / last, half-page |
| `hl` / `←→` | fold / open task course groups and Materials groups |
| `1` / `2` | Tasks / Materials |
| `tab` / `shift+tab` | next / previous pane |
| `[` / `]`, `3` / `4` | utility views, including Sync status and Commit |
| `enter` | choose course, fold/open a group, or toggle task/file details; Commit copies/writes the preview |
| `space`, `a` | mark item/group, mark all matching items |
| `v`, `*` | range-mark / invert visible marks |
| `u` / `ctrl+z` | undo the last bulk decision |
| `t` / `i` / `m` / `n` | track / ignore / missed / undecided |
| `f`, `A` | open filters / show All tasks |
| `/` | search within the current view and course |
| `S` | reverse Materials week order (newest / oldest first) |
| `esc` | dismiss the current menu, confirmation, inspector, search, filter, or marks |
| `c` | review & commit the export queue |
| `w`, `y` | write / copy the exact queue preview (Commit) |
| `x` / `X` | drop one / all queued items (Commit) |
| `s` / `r` | sync the source |
| `d` | download or retry marked/focused materials |
| `o` / `O` | open LMS source / open the focused local file |
| `:` / `ctrl+k`, `?` | command palette / full help |
| `q` | quit |

The inspector replaces the content on narrow terminals and splits beside it at
140 columns or wider. It shows the task name, full course name, deadline, decision, available text,
and related materials. Raw task/course IDs stay out of ordinary task details. Sync and Commit retain their
existing semantics: decisions save immediately; Commit is the separate export
step, and Sync status remains available with `3` or the command palette.

## Project structure

```
cmd/kamps/          entry point, flags, login, source registration
internal/model/     Course, Assignment, Material — data types
internal/core/      application pipeline: load, state merge, auto-triage, reconcile
internal/api/       JSON DTOs for headless commands
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

See [`docs/export.md`](docs/export.md), [`docs/workspace.md`](docs/workspace.md), [`docs/headless.md`](docs/headless.md), and [`docs/plugins.md`](docs/plugins.md) for the new surfaces.

## Status & roadmap

The core roadmap is implemented: live parsing/downloads, resumable transfers, library scan, state locking, real sync, auto-triage, session expiry, offline cache, browser cookie login, exporters, workspace mapping, the `workspace init` template helper, quiz deadlines/status, page links, and the source registry. Next up, in order:

1. Parse quiz questions and more module-specific attachments.
2. Add optional Todoist/out-of-process source integrations.
3. Polish offline cache retention and Library views for files kamps did not download.

Implemented in this pass: assignment attachments, folder files, resumable downloads, download retries/concurrency, startup library scan, state locking/merge, real sync, auto-triage of submitted work, session expiry, offline cache, browser cookie login, Markdown/todo.txt/Taskwarrior exporters, `course.json` workspace mapping, source registry, and range/bulk selection.
