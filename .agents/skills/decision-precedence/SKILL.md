---
name: decision-precedence
description: "Check the bounded exhaustive decision-precedence invariant in kamps' Go state store when changing user decisions, automatic triage, overrides, or state application."
---

# Verify decision precedence

## Property and user impact

A user's explicit decision, including a deliberate reset to undecided, must not be replaced by automatic triage when state is applied to refreshed coursework. If this fails, a sync can silently undo a choice the user made.

## Check

From the repository root, run:

```sh
go run ./tools/verify_decision_precedence
```

The checker calls the production `store.New`, `State.SetDecision`, `State.SetAutoDecision`, and `State.Apply` methods. It enumerates all four supported triage values for the user choice and automatic choice, in both operation orders, for one assignment ID. That is 32 concrete executions. It reports the order and values on a counterexample. It needs only the Go toolchain already required by the project.

## Coverage record

- Production code: `internal/store/store.go`, methods `New`, `State.SetDecision`, `State.SetAutoDecision`, and `State.Apply` (delegating to `apply`).
- Executable checker: `tools/verify_decision_precedence/main.go`.
- Sync commit: `496097a121e62d2eb3b9d90dae87d332acf42ea8`.
- Last confirmed: 2026-10-27; all 32 cases passed.

The project has no installed formal verifier for Go, so this is exhaustive executable checking over the stated finite input pairs and operation orders. It is not a proof of all store behavior, persistence, concurrency, or integration with the LMS. It uses a direct property oracle against production methods, not a separate model of the store.

## Updating the check

When a change touches the listed methods or their callers, inspect the production code and confirm that the property and covered operation orders still match the intended behavior. Update the checker if the valid decision domain or relevant sequence changes. Run it, then move the sync commit to the new commit only after confirming that the checker still describes that code. If the checker finds a failure, report its concrete sequence and determine whether the property or implementation is wrong; do not weaken the property just to get a pass.

## Prove the checker can fail

To confirm the check is live, temporarily remove both protections that enforce overrides: the `Overrides` guard in `State.SetAutoDecision` and the `!s.Overrides[id]` condition in `State.apply`, using a scratch copy of the production files. The `user then automatic` case with user choice `undecided` and automatic choice `tracked` must fail. Restore the scratch changes and rerun the checker; it must pass again. Never commit the scratch mutation.
