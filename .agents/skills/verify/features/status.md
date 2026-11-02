# Status

## Sub-features

- Summarize the selected source, session status, course/task/material counts, cache state, and downloads.

## How to get to it (user POV)

In the TUI, open Sync with `3`. In the CLI, run `kamps status --json`.

## Driving it with the isolated demo CLI

Run `bash scripts/verify-demo.sh` and inspect `status.json`. It should report the demo source and nonzero coursework counts. The session cookie is not part of this output.

## Gotchas

- Demo status does not validate a live LMS session or network sync.
