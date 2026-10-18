# kamps

> Your college coursework, in your terminal — and it feels good to use.

`kamps` is a fast, friendly TUI that pulls courses, assignments and materials
from your LMS (Moodle / Kulon2) plus lab drives, keeps them in sync, and stores
what you download locally. One small binary, no browser tabs, no spreadsheet of
deadlines you never open.

It is written from scratch in Go: type-safe, compiled, single file, runs the
same on Linux, macOS and Windows.

```
╭─ kamps  Tasks 42   Materials 16   Sync  ────────────── backup 0d · sync 1/2 · cookie ok ─╮
│  today   courses  materials  sync                                                   smt11 ▾ │
│ ╭─ today ─────────────────────────────────────╮ ╭─ triage ───────────────────────────────╮ │
│ │ OVERDUE  19                                 │ │ track      6  ███░░░░░░░░░░░░░░░░░░░░░ │ │
│ │     ppl      08-19 Progress Report 1        │ │ missed     3  █░░░░░░░░░░░░░░░░░░░░░░░ │ │
│ │ ▌   strukdat 08-24 Submission Praktikum 1   │ │ ignore     5  ██░░░░░░░░░░░░░░░░░░░░░░ │ │
│ │     metnum   08-27 Tugas Kelompok I. Galat  │ │ undecided 28  ████████████████░░░░░░░░ │ │
│ │    … 11 more                                │ │                                        │ │
│ │                                             │ │ 19 past-due undecided                  │ │
│ │ NEXT 3 DAYS  1                              │ │ → [c] review & commit                  │ │
│ │     ppl      10-19 Progress report 9        │ │ by course                              │ │
│ ╰─────────────────────────────────────────────╯ │ strukdat  8/11 █████████████████░░░░░░ │ │
│                                                 │ metnum    7/12 ██████████████░░░░░░░░░ │ │
│                                                 ╰────────────────────────────────────────╯ │
│  queue: nothing to commit                                        [c] review & commit      │
│  ↑↓ move · space select · t/m/i/u triage · / filter · v view · c commit · ? help           │
╰────────────────────────────────────────────────────────────────────────────────────────────╯
```

It runs **full-screen** and restores your shell exactly as it was when you quit.
The cursor row is a solid bar, so you always know what `t`/`m`/`i` will hit. The
colours are Charm's defaults: a purple frame, pink accents, and 256-colour greys
that adapt to a light or dark terminal.

## What it is (and is not)

`kamps` is not a todo list. It manages coursework that arrives from an LMS, from
your own manual tracking, or from a WhatsApp hand-out — and it records what you
decided about each item:

| decision | means |
|---|---|
| `undecided` | you have not looked at it yet (the default) |
| `tracked` | handled — submitted, a teammate submitted it, or it is tracked elsewhere |
| `missed` | the deadline passed and it is gone |
| `ignored` | not real work — a template, last year's file, an optional extra |

Your actual to-do list lives wherever you keep it; `kamps` can hand decided
items to it (see *review & commit* below).

## Status

**Early, and honest about it.** The whole UI is built and runs today against a
bundled demo dataset. The real Moodle client lands behind the same interface, so
the UI you see now is the UI you keep. The board view and the inline (non
full-screen) mode were tried and are parked: the board is unreachable from the
keyboard, and the app always uses the alternate screen.

## Build and run

Requires Go 1.24+.

```sh
git clone <this repo> kamps
cd kamps
go run ./cmd/kamps          # launch the UI
go test ./...               # run the (small) test suite
go build -o kamps ./cmd/kamps
```

Piping works too — when stdout is not a terminal, `kamps` prints a plain-text
snapshot instead of the TUI:

```sh
kamps | less
kamps version
kamps --no-anim        # plain motion-free session
```

## Screens

Three tabs, switched with `1`–`3` or `tab`:

- **Tasks** — your backlog, with two everyday views cycled by `v`:
  - **dashboard** — overdue, due today and coming up, the triage split, where the
    backlog lives per course, and sync state (the default)
  - **list** — a two-pane list plus a detail panel showing what would be written
    to `tasks.md`
- **Materials** — course files, slides and lab Drive links; `d` downloads.
- **Sync** — sync state, backup age and your local library path.

The guided **review & commit** wizard opens with `c`, and a parked **board** view
still exists in the code (§ *Status*).

Every Tasks view drives the same cursor and the same keys: `↑↓` moves, `space`
selects, `t`/`m`/`i`/`u` decide — including inside the dashboard's today list.
Switch views freely; a decision made in one shows up in all of them.

## Keys

| key | action |
|---|---|
| `↑` / `k`, `↓` / `j` | move |
| `←` / `h`, `→` / `l` | move between board columns |
| `tab`, `shift+tab` | next / previous tab |
| `1`–`3` | jump to a tab |
| `v` | cycle the Tasks view (dashboard · list) |
| `space` | select an item (shown with a `▪` marker) |
| `a` | select all |
| `t` / `m` / `i` / `u` | decide: track / missed / ignore / undecided |
| `/` | cycle the filter |
| `c` | review & commit |
| `s` | sync |
| `d` | download (Materials) |
| `?` | help |
| `q` | quit |

### Review & commit

`t`/`m`/`i` apply immediately **and** queue the item. `c` opens the wizard, which
shows the *exact* lines that would be written before anything happens:

```
   └ Preflight   └ Select   └ Action   ◆ Review   ◇ Done

   - [ ] [ppl] Report 0 (K150004) due:2026-08-17
   - [ ] [strukdat] Kuis Struktur Data awal (K150001) due:2026-08-21
```

Writing is currently a **dry-run** — the wizard shows what it would write and
reports what it did, but touches no files. The real writer is a later step.

## How it is shaped

- **UI first.** The whole app is written against one small `source.Source`
  interface. A demo source fills it today; the Moodle source fills it later.
  Swapping sources changes nothing in the UI.
- **Local by default.** Your data lives in a local store and a configurable
  library directory. Nothing is written into your dotfiles unless you ask.
- **Yours to extend.** Tabs, keys, theme colours and sources are separate, small
  files — not one giant controller.
- **Light on its feet.** Standard-library first, few dependencies, no background
  daemons.

## Layout

```
cmd/kamps/        entry point (flags, plain-text fallback, launch TUI)
internal/model/   Course, Assignment, Material — dumb data
internal/source/  the Source interface + the demo source
internal/tui/     the Bubble Tea app: app.go (state), views.go (screens),
                  frame.go (boxes), util.go (helpers), theme.go (colours),
                  keys.go (bindings), plain.go (no-TTY output)
internal/config/  paths and settings
docs/kulon2.md    verified notes on the Moodle/Kulon2 instance
```

## Roadmap

1. **Now** — interactive UI over demo data: three tabs, a dashboard and
   two-pane Tasks view, triage decisions, review & commit (dry-run). ✅
2. Real Moodle source: cookie auth, course list, HTML parsing for assignments
   and materials. See `docs/kulon2.md`.
3. Local SQLite store so decisions and downloads survive restarts.
4. The real commit writer: append decided lines to `tasks.md` behind the wizard's
   preview.
5. Real downloads with progress, retry and a configurable library directory.
6. Praktikum (lab Drive) folders per course.
7. Theming and a config file.

Later, not now: bulk actions, search, and a plugin surface for new sources.
