# kamps

> Your college coursework, in your terminal — and it feels good to use.

`kamps` is a fast, friendly TUI that pulls courses, assignments and materials
from your LMS (Moodle / Kulon2) plus lab drives, keeps them in sync, and stores
what you download locally. One small binary, no browser tabs, no spreadsheet of
deadlines you never open.

It is written from scratch in Go: type-safe, compiled, single file, runs the
same on Linux, macOS and Windows.

```
◆ kamps  kulon2.undip.ac.id                            smt11  ● ready

  Courses 5   Assignments 12   Materials 16   Downloads 2
──────────────────────────────────────────────────────────────
  DUE        COURSE      STATUS   ASSIGNMENT
▌ 2026-10-22 MIK1624302  ○ todo   Submission Praktikum 6
  2026-10-24 PAIK6304    ◐ doing  Proyek Metode Numerik
  2026-10-15 PAIK6504    ○ todo   Progress Report 5 (UTS)
──────────────────────────────────────────────────────────────
 ↑/k up • ↓/j down • tab section • x done • d download • ? help
```

## Status

**Early, and honest about it.** The interface is built and enjoyable to click
around, running today against a bundled demo dataset. The real Moodle client
lands behind the same interface, so the UI you see now is the UI you keep.

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
```

## Keys

| key | action |
|---|---|
| `↑` / `k`, `↓` / `j` | move |
| `tab`, `shift+tab` | switch section |
| `1`–`4` | jump to a section |
| `space` | select a material |
| `d` | download selected (or the cursor item) |
| `x` | mark an assignment done |
| `s` | sync |
| `?` | help |
| `q` | quit |

## How it is shaped

- **UI first.** The whole app is written against one small `source.Source`
  interface. A demo source fills it today; the Moodle source fills it later.
  Swapping sources changes nothing in the UI.
- **Local by default.** Your data lives in a local store and a configurable
  library directory. Nothing is written into your dotfiles unless you ask.
- **Yours to extend.** Sections, keys, theme colours and sources are separate,
  small files — not one giant controller.
- **Light on its feet.** Standard-library first, few dependencies, no
  background daemons.

## Layout

```
cmd/kamps/        entry point (flags, plain-text fallback, launch TUI)
internal/model/   Course, Assignment, Material — dumb data
internal/source/  the Source interface + the demo source
internal/tui/     the Bubble Tea app: theme, keys, views
internal/config/  paths and settings
docs/kulon2.md    verified notes on the Moodle/Kulon2 instance
```

## Roadmap

1. **Now** — interactive UI over demo data. ✅
2. Real Moodle source: cookie auth, course list, HTML parsing for assignments
   and materials. See `docs/kulon2.md`.
3. Local SQLite store + the `todo → doing → done` task state.
4. Real downloads with progress, retry and a configurable library directory.
5. Praktikum (lab Drive) folders per course.
6. Theming, config file, optional one-way `tasks.md` export.

Later, not now: bulk actions, search, and a plugin surface for new sources.
