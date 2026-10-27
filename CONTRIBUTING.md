# Contributing to Kamps

Kamps is intentionally small: one binary, a handful of packages, and no
network service. The fastest way to become productive is to read the guided
tour, then make one small change with the test suite as your safety net.

## Start here

1. [Source Code Tour](docs/code-tour.md) — where everything lives and recipes
   for common changes.
2. [Learning Path](docs/learning-path.md) — a 30/60/120-minute route through
   the code plus small exercises.
3. [Architecture](docs/architecture.md) and
   [Execution Flow](docs/execution-flow.md) — the wider picture.

## Build and run

```sh
make run          # go run ./cmd/kamps
make build        # build ./kamps
make test         # go test ./...
```

If `make` is unavailable:

```sh
go run ./cmd/kamps
go build -o kamps ./cmd/kamps
go test ./...
```

## Verify before committing

```sh
make check          # or ./scripts/check.sh when make is unavailable
```

That runs:

- `gofmt -l .` (fails if any file needs formatting)
- `go vet ./...`
- `go test ./...`

To fix formatting, run `make fmt`.

## Code conventions

- Keep the UI borderless: full-width rules, not boxes. The visual system —
  palette, size classes, footer, help, and focus states — is described in
  [DESIGN.md](DESIGN.md).
- Keep colors in `internal/tui/theme.go`; widgets use semantic styles.
- Trees keep a two-column right inset, and long text ends with `…`.
- `/` is search; `f` opens filters. Do not bind `v` as a view key.
- A user decision is saved immediately; the commit queue is session-only.
- Do not write tests against the real home directory. Use `t.TempDir()`.
- Prefer moving code before renaming it or changing behavior. Small,
  reviewable commits are easier to verify.

## First change ideas

- Add a filter and test it alone and composed with search.
- Add a field to the task or material detail pane and assert its width.
- Add a key binding, then update the footer hints and `?` overlay.
- Add a plain-mode test for a saved decision or download flag.

## Never touch

The untracked `.commandcode/` directory is not part of the project. Do not
stage, edit, or delete it.
