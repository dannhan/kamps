# Task decisions

## Sub-features

- List tasks and their current triage state.
- Mark a task tracked, missed, ignored, or undecided and keep the choice across launches.

## How to get to it (user POV)

In the TUI, open Tasks with `1`, focus a task, then use the decision key shown in the footer. In the CLI, run `kamps tasks --json` and `kamps triage <id> --tracked --json`.

## Driving it with the isolated demo CLI

Run `bash scripts/verify-demo.sh`. It reads task `K901017` before the action, marks it tracked, launches a new process, and reads the task again. The output `tasks-before.json` must show `undecided`; `triage.json` must report `tracked`; `tasks-after.json` and `state.json` must preserve that choice.

## Gotchas

- The demo uses temporary XDG paths. Never point this journey at the user's real state directory.
- A failed decision command must not be treated as success based on the later display alone; inspect both the command result and state file.
