# Materials

## Sub-features

- List materials for a course.
- Preview a download with `--dry-run`.

## How to get to it (user POV)

In the TUI, open Materials with `2`, choose a course, and expand a week. In the CLI, run `kamps materials --json --course MIK1624302`.

## Driving it with the isolated demo CLI

Run `bash scripts/verify-demo.sh`. Inspect `materials.json` for the selected course and `download-dry-run.json` for material `M165402`. The harness uses bundled demo data, so no external transfer is made.

## Gotchas

- Demo download behavior does not prove live server transfer, resume, or filesystem permission behavior.
- Check the reported command and output; the `--dry-run` label alone does not prove which side effects a code path skips.
