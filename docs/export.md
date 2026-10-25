# Exporting the commit queue

The Commit tab (`c`) is a review-and-export buffer. Decisions are saved as soon
as you make them; the queue is the batch you want to hand to another tool.

## Preview and write

- `w` writes the queue to the configured target.
- `y` copies the same text through OSC 52.
- `enter` writes when a target is configured, otherwise copies.
- `x` drops one item from the queue; `X` drops all (press twice).

The preview uses the selected exporter, so what you see is exactly what is
appended.

## Formats

```toml
# ~/.config/kamps/config.toml
export_format = "markdown"   # markdown | todo | taskwarrior
commit_file = "~/notes/tasks.md"
```

Or per run:

```sh
kamps --export-format markdown --commit-file ~/notes/tasks.md
```

| Format | Output |
|---|---|
| `markdown` | `- [ ] [course] Name (K123) due:2026-08-31`; missed/ignored use `- [x]` with `#missed`/`#ignored` |
| `todo` | `[course] Name (K123) due:2026-08-31 kamps:K123`; closed items start with `x YYYY-MM-DD` |
| `taskwarrior` | `task add project:<course> +kamps "<name> (K123)"`; closed items also get `+closed` |

Writes are idempotent: each format parses existing IDs and skips items that are
already present. File writers lock the target and replace it atomically, so a
crash cannot leave a half-written task list.

Obsidian vaults work through the Markdown format: point `commit_file` at a note
or inbox file in the vault.

Taskwarrior requires the `task` binary on `PATH`. Kamps uses the CLI with
`rc.confirmation=off` and never edits Taskwarrior's private data files.

Todoist is intentionally not built in; use the clipboard or a Markdown file and
sync that with your task manager.
