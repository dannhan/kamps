# AGENTS.md

Instructions for AI agents working in this repo. Humans: read `README.md`.

## What this is

`kamps` is a Go TUI for college coursework (courses, assignments, materials)
pulled from an LMS (Moodle / Kulon2) and lab drives. It is a deliberate rewrite
of an older Python tool that grew too complex. **Do not reintroduce that
complexity:** no milestone/amendment/decision machinery, no dozen-plus
subcommands, no bespoke safe-writer framework, no heavy test scaffolding.

Module path: `github.com/ruyam/kamps`.

## Commands

```sh
go run ./cmd/kamps            # launch the TUI (needs a real TTY)
go run ./cmd/kamps version    # prints version; works without a TTY
go test ./...                 # tests
go vet ./...
go build -o kamps ./cmd/kamps
```

When there is no TTY (CI, pipes), `kamps` prints a plain-text snapshot. Use
`kamps | less` and `go run ./cmd/kamps version` to verify behaviour headlessly.
`go test -run TestSnapshot -v ./internal/tui` renders every screen — that is how
you eyeball layout changes without a terminal. `go test -run TestViewAlignment`
is the guard that every line is exactly `boxWidth()` wide.

## Architecture

Everything is built against one interface, `source.Source` (see
`internal/source/source.go`):

```go
Courses(ctx)     ([]model.Course, error)
Assignments(ctx) ([]model.Assignment, error)
Materials(ctx)   ([]model.Material, error)
```

- `internal/source/demo.go` implements it with an in-memory dataset.
- A future `internal/source/moodle.go` implements it with HTTP + HTML parsing.
- **The UI must not change when the source changes.** If a change would require
  editing `internal/tui` to accommodate a real source, the abstraction is
  wrong; fix the abstraction instead.

Other packages: `internal/model` (plain structs), `internal/tui` (`app.go` =
Elm-style model/update + key handling, `views.go` = the screens/layouts,
`frame.go` = hand-drawn boxes/chips/hints, `util.go` = pure helpers, `theme.go`
= colours/styles, `keys.go` = keybindings, `plain.go` = no-TTY output),
`internal/config` (paths/settings).

## The task model

`model.Triage` is the local, user-owned decision about an assignment, and it is
**not** a todo status. The vocabulary is fixed:

- `Undecided` (default) · `Tracked` · `Missed` · `Ignored`

Do not reintroduce `todo/doing/done`. `Tracked` already covers "done",
"a teammate submitted it" and "tracked elsewhere". Keys are `t`/`m`/`i`/`u`.

`t`/`m`/`i` apply immediately and *also* queue the item in `App.pending`;
`[c]` opens the wizard, which previews the exact `tasks.md` lines. **Writing is
a dry-run** (`App.dryRun == true`): the wizard must not touch any file until a
real, reviewed writer is added deliberately.

## Rendering invariants

The UI runs **full-screen** (`tea.WithAltScreen()`), sized to the terminal:
`clampInt(width, 40, 400)` × `clampInt(height, 10, 200)`. An inline mode was
tried and parked; if you bring it back, it needs its own sizing rules.

Every line of `App.View()` must be exactly `boxWidth()` columns. Overflows are
caught by `TestViewAlignment` at 100/80/72 columns across every tab and Tasks
layout (dashboard · list · board · wizard) — if you add a row or a column, keep
it inside its width budget or extend `assignRow`/`chips`/`hintsFit`/`barW`, which
already degrade (drop the id column, drop chips, drop hint items) rather than
overflow. Never "fix" an overflow by widening the box past the terminal.

Tasks layouts must all expose their rows through `App.focusedList()` so the
cursor and `t`/`m`/`i`/`u` behave identically in every view. `v` only cycles the
layouts in `viewCycle`; the board and wizard are deliberately outside it.

The **cursor row is always drawn as a solid bar** (`stRowCursor`) and a
space-selected row always carries the `▪` marker — a highlight the user can see
is not optional. `assignRow` is the single place rows are built, and it returns
styled text of at most `contentW` columns; callers pad.

The dashboard's today panel draws exactly the items `App.dashShown()` returns,
and `dashAlloc` is the only thing that decides how many fit. If you change one
without the other, the highlight lands on the wrong row.

## Conventions

- **Elm architecture.** `Init` starts async work; `Update` handles messages;
  `View` is pure and returns a string. Never mutate state inside `View`.
- **Style in one place.** Colours and text styles live in `internal/tui/theme.go`.
  The palette is Charm's defaults: adaptive 256-colour greys for structure plus
  purple/pink accents. Keep it adaptive (works on light and dark terminals),
  keep contrast high, and do not scatter `lipgloss.NewStyle()` calls through the
  views.
- **Keys in one place.** Add bindings to `KeyMap` in `keys.go`; they drive the
  `?` overlay. The one-line footer hints are hand-built in `hintsLine`
  (`views.go`) because `bubbles/help` truncates; keep them short — `hintsFit`
  drops trailing items rather than overflow.
- **One screen, one function.** Each tab (and each Tasks layout) renders in its
  own `*Lines(w, budget int) []string` in `views.go`. Compose them from
  `frame.go` (`panel`, `sideBySide`, `chips`, `spread`); do not hand-roll boxes.
- **Data is dumb; behaviour is elsewhere.** `internal/model` types are plain
  structs with no I/O.
- **Politeness and safety for any networking** (when it lands): ≥1 s between
  requests, ≤2 requests per page, and treat an HTML-parse failure as an error —
  never as "zero courses/assignments".
- Comment the *why*, not the *what*. Match the tone of the existing code:
  calm, concrete, no hype.

## Ground rules

- Never commit secrets, cookies, tokens, or real user data. `.gitignore`
  already excludes `config.toml`, cookies and the library dir — keep it that way.
- Never automate an SSO login. The user pastes a session cookie themselves; the
  tool only stores and checks it.
- Prefer the standard library. Adding a dependency needs a reason: it should be
  small, well-maintained, and pull its weight.
- Keep tests small and meaningful. Unit-test pure helpers; use the snapshot
  test to catch rendering panics. Do not build a large harness.

## Where to look

- `docs/kulon2.md` — verified facts about the target Moodle instance (auth,
  working and blocked data routes). Update it only with things you actually
  verified; mark guesses as **unverified**.
- `README.md` — user-facing description and roadmap.
