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

Your actual to-do list lives wherever you keep it; `kamps` can hand decided
items to it via a review & commit wizard (`c`) that previews the exact lines
before anything is written.

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
| `tab`, `1`–`3` | switch tabs |
| `v` | cycle the Tasks view |
| `space`, `a` | select item / select all |
| `t` / `m` / `i` / `u` | decide: track / missed / ignore / undecided |
| `/` | cycle the filter |
| `c` | review & commit |
| `s` | sync |
| `d` | download (Materials) |
| `?` | help |
| `q` | quit |

## Project structure

```
cmd/kamps/        entry point
internal/model/   Course, Assignment, Material — data types
internal/source/  the Source interface + the demo data source
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
2. Local store so decisions and downloads survive restarts.
3. The real commit writer behind the wizard's preview.
4. Downloads with progress and a configurable library directory.

Later, not now: bulk actions, search, and a plugin surface for new sources.
