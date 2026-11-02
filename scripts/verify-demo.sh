#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
evidence_dir="$repo_root/docs/verification/kamps-demo/latest"
run_dir="$(mktemp -d "${TMPDIR:-/tmp}/kamps-verify.XXXXXX")"
trap 'rm -rf "$run_dir"' EXIT

mkdir -p "$run_dir/config" "$run_dir/state" "$evidence_dir"
export XDG_CONFIG_HOME="$run_dir/config"
export XDG_STATE_HOME="$run_dir/state"

binary="$run_dir/kamps"
GOCACHE="${GOCACHE:-${TMPDIR:-/tmp}/kamps-go-cache}" go build -buildvcs=false -mod=readonly -o "$binary" ./cmd/kamps

run_pty() {
  local command
  printf -v command '%q ' "$binary" "$@"
  script -q -e -c "$command" /dev/null
}

run_pty version > "$evidence_dir/version.txt"
run_pty --demo tasks --json > "$evidence_dir/tasks-before.json"
run_pty --demo materials --json --course MIK1624302 > "$evidence_dir/materials.json"
run_pty --demo download M165402 --dry-run --json > "$evidence_dir/download-dry-run.json"
run_pty --demo triage K901017 --tracked --json > "$evidence_dir/triage.json"
run_pty --demo tasks --json > "$evidence_dir/tasks-after.json"
run_pty --demo status --json > "$evidence_dir/status.json"

grep -q '"id":"K901017"[^}]*"triage":"undecided"' "$evidence_dir/tasks-before.json"
grep -q '"id":"K901017"[^}]*"triage":"tracked"' "$evidence_dir/tasks-after.json"
grep -Eq '"K901017":[[:space:]]*1' "$run_dir/state/kamps/state.json"
if grep -q 'M165402' "$run_dir/state/kamps/state.json"; then
  printf 'FAIL: dry-run recorded material M165402 as downloaded\n' >&2
  exit 1
fi
grep -q '"command":"materials"' "$evidence_dir/materials.json"
grep -q '"command":"download"' "$evidence_dir/download-dry-run.json"
grep -q '"command":"status"' "$evidence_dir/status.json"
cp "$run_dir/state/kamps/state.json" "$evidence_dir/state.json"
cat > "$evidence_dir/actions.txt" <<'EOF'
Isolated run with empty XDG config and state directories.
Each command ran in its own PTY via util-linux `script`.
1. kamps version
2. kamps --demo tasks --json
3. kamps --demo materials --json --course MIK1624302
4. kamps --demo download M165402 --dry-run --json
5. kamps --demo triage K901017 --tracked --json
6. kamps --demo tasks --json (relaunch; tracked decision remains)
7. kamps --demo status --json
EOF
printf 'PASS: demo task decision persisted across CLI invocations. Evidence: %s\n' "$evidence_dir"
