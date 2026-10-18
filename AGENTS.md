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
you eyeball layout changes without a terminal.

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
Elm-style model/update, `views.go` = rendering, `theme.go` = colours/styles,
`keys.go` = keybindings), `internal/config` (paths/settings).

## Conventions

- **Elm architecture.** `Init` starts async work; `Update` handles messages;
  `View` is pure and returns a string. Never mutate state inside `View`.
- **Style in one place.** Colours and text styles live in `internal/tui/theme.go`.
  Do not scatter `lipgloss.NewStyle()` calls through the views.
- **Keys in one place.** Add bindings to `KeyMap` in `keys.go`; they feed both
  the footer and the `?` overlay automatically.
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
