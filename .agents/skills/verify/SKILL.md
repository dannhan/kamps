---
name: verify
description: "Drive kamps, the Go terminal coursework tracker, through its isolated demo CLI and verify user-visible tasks, materials, decisions, and persisted state after changes to CLI, core, storage, or TUI behavior."
---

# Verify kamps

Use this skill to prove behavior through kamps' real executable. The primary surface is the Bubble Tea TUI; the same core service also powers the documented JSON CLI, which is the stable automated driver used here. Live Kulon2 mode needs a user session and network access, so the automated journey uses bundled demo data and never touches the user's normal XDG directories.

## Launch

From the repository root, run:

```sh
bash scripts/verify-demo.sh
```

The script builds `./cmd/kamps`, creates empty temporary `XDG_CONFIG_HOME` and `XDG_STATE_HOME` directories, and runs one short-lived CLI process per action with `--demo`. The demo source is ready when `kamps status --json` returns an object whose source is `demo (no network)`. There is no persistent server. The script removes only the temporary run directory it created; evidence remains under `docs/verification/kamps-demo/latest/`.

## Doctor

Run `bash scripts/verify-demo.sh` from the repository root. A usable instance is a fresh successful build followed by a successful `kamps version`, `tasks`, `materials`, `download --dry-run`, `triage`, and `status` command sequence. If the build fails, a command exits nonzero, or the state assertions fail, stop and investigate before driving the app further.

## Drive

The harness is `scripts/verify-demo.sh`. It starts every short-lived command in a fresh PTY through `script -q -e -c`. Its stable inputs are the documented CLI commands and demo identifiers: task `K901017`, course `MIK1624302`, and material `M165402`. The script queries tasks before changing the decision, lists the course's materials, runs a dry-run download, marks the task tracked, starts a new process, and queries tasks and status again.

For a TUI-specific change, use a separate interactive PTY with isolated state and save its transcript:

```sh
run_dir="$(mktemp -d)"
trap 'rm -rf "$run_dir"' EXIT
mkdir -p "$run_dir/config" "$run_dir/state"
GOCACHE="${GOCACHE:-/tmp/kamps-go-cache}" go build -buildvcs=false -mod=readonly -o "$run_dir/kamps" ./cmd/kamps
printf -v tui_command '%q --demo' "$run_dir/kamps"
XDG_CONFIG_HOME="$run_dir/config" XDG_STATE_HOME="$run_dir/state" \
  script -q -e -c "$tui_command" docs/verification/kamps-demo/latest/tui.typescript
```

Use the keys in `README.md` and exit with `q`. Do not share the user's interactive TUI process.

## Evidence

The script writes `version.txt`, `tasks-before.json`, `materials.json`, `download-dry-run.json`, `triage.json`, `tasks-after.json`, `status.json`, `state.json`, and `actions.txt` under `docs/verification/kamps-demo/latest/`. Review the actions alongside the outputs: the task begins undecided, `triage` reports the tracked decision, a fresh process reports it as tracked, and the persisted state contains `K901017`. The state file must not contain material `M165402`, which confirms the dry-run created no download record. The demo source has no network downloads, and the temporary run is removed after the script exits.

The proof uses the real binary and core service, not internal setters or test-only endpoints. It verifies the state file as well as command output. It does not prove live authentication, LMS parsing, real downloads, or every TUI interaction.

## Cleanup

The script's exit trap removes its own isolated build, config, and state directory. It does not remove evidence. For manual TUI runs, exit with `q` and terminate only the PTY process started for that run. Never kill processes by name or use the user's regular XDG directories for this demo journey.

## Helpers

Run `bash scripts/verify-demo.sh` from the repository root. The script is executable and owns its temporary data directory and evidence outputs.
