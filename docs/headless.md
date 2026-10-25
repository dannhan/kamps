# Headless core and CLI

Kamps' application workflow lives in [`internal/core`](../internal/core/core.go). The TUI and
the non-TTY renderer both consume the same `core.Snapshot`. The next phase exposes the same
service as JSON CLI commands so an agent can read and mutate kamps without opening the TUI.

## Status

- Implemented: `core.Service` load pipeline (workspace mapping, sorting, saved state,
  auto-triage, library reconciliation, cache save/fallback), shared by `tui` and `plain`.
- Implemented: import guard ensuring `internal/core` never imports the TUI or Lip Gloss.
- Planned: JSON subcommands and the `run(args, stdout, stderr) int` CLI shell.

## Planned commands

```text
kamps courses   [--json]
kamps tasks     [--json] [--filter ...] [--search Q] [--course ID|CODE]
kamps materials [--json] [--filter ...] [--search Q] [--course ID|CODE]
kamps status    [--json]
kamps sync      [--json]
kamps triage    <id...> --tracked|--missed|--ignored|--undecided [--json]
kamps download  <id...> | --all-new | --course ID|CODE [--json] [--dry-run]
kamps export    --format markdown|todo|taskwarrior [--out PATH] [--json]
```

Existing `login`, `logout`, `version`, and `workspace init` stay unchanged. Running `kamps`
with no subcommand still opens the TUI (or prints plain text when stdout is not a TTY).

## JSON shape

Every `--json` command writes exactly one object to stdout. Diagnostics go to stderr.

```json
{
  "version": 1,
  "command": "tasks",
  "generated_at": "2026-10-25T17:00:00+07:00",
  "items": []
}
```

Planned item fields:

- `courses`: `id`, `code`, `short`, `name`, `term`, `workspace_path`
- `tasks`: `id`, `course_id`, `course_code`, `name`, `due` (RFC3339 or null), `triage`
  (`undecided|tracked|missed|ignored`), `submission`, `section`, `overdue`, `attachments`
- `materials`: `id`, `course_id`, `name`, `section`, `folder`, `kind`, `size`,
  `downloaded`, `assignment_id`

Errors use a structured object on stderr and a non-zero exit code:

```json
{"error": {"code": "auth", "message": "session expired; run kamps login"}}
```

Exit codes: `0` success, `2` usage, `3` auth/session, `4` source/network, `5` partial.

## Agent workflow

```sh
kamps status --json
kamps tasks --json --filter undecided
kamps triage K901017 --tracked --json
kamps download M165402 --dry-run --json
kamps download M165402 --json
kamps export --format markdown --out ~/notes/tasks.md --json
```

## Boundaries

- `internal/core` must not import `internal/tui`, `internal/plain`, `cmd/kamps`, or any
  `charmbracelet/*` package.
- JSON uses dedicated DTOs; `model` structs keep their cache schema.
- The session cookie is never exposed in `status` output.
- A local HTTP server or MCP wrapper is optional and must call the same service; it must
  not reimplement business logic.
