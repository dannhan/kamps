# Headless core and CLI

Kamps' application workflow lives in [`internal/core`](../internal/core/core.go). The TUI and
the non-TTY renderer both consume the same `core.Snapshot`. JSON CLI commands expose the
same service for scripts and agents.

## Status

- Implemented: `core.Service` load pipeline (workspace mapping, sorting, saved state,
  auto-triage, library reconciliation, cache save/fallback), shared by `tui` and `plain`.
- Implemented: import guard ensuring `internal/core` never imports the TUI or Lip Gloss.
- Implemented: `courses`, `tasks`, `materials`, `status`, `sync`, `triage`, `download`,
  and `export` commands, with `--json` and a structured error envelope.
- Implemented: per-command help (`kamps <command> --help`).
- Planned: full `run(args, stdout, stderr) int` refactor and the optional localhost/MCP
  surface.

## Commands

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

`download` supports `--all-new`, `--course`, and `--dry-run`. `export` defaults to tracked
items and accepts `--ids A,B`; `--format` defaults to `export_format` from config.

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

Item fields include:

- `courses`: `id`, `code`, `short`, `name`, `term`, `workspace_path`
- `tasks`: `id`, `course_id`, `course_code`, `name`, `due` (RFC3339 or null), `triage`
  (`undecided|tracked|missed|ignored`), `submission`, `section`, `overdue`, `attachments`
- `materials`: `id`, `course_id`, `course_code`, `name`, `section`, `section_name`,
  `folder`, `kind`, `size`, `downloaded`, `url`, `assignment_id`
- `status`: `source`, `session_ok`, `session_expires_at`, `last_sync`, `from_cache`,
  `courses`, `assignments`, `materials`, `downloaded`

Material filters are `all`, `new`, `downloaded`, and `link`. Task filters are `all`,
`undecided`, `overdue`, `tracked`, `missed`, and `ignored`.

Errors use a structured object on stderr and a non-zero exit code:

```json
{"error": {"code": "auth", "message": "session expired; run kamps login"}}
```

Exit codes: `0` success, `2` usage, `3` auth/session, `4` source/network, `5` partial.

## Agent workflow

```sh
kamps status --json
kamps tasks --json --filter undecided
kamps materials --json --course MIK1624302
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
