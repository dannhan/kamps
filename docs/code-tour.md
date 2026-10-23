# Source Code Tour

This is the practical map for changing Kamps: where things live, which files to read first, and how to verify a change. It describes the code as it exists today. For the intended architecture, see [Architecture](architecture.md); for runtime behavior, see [Execution Flow](execution-flow.md).

---

## 1. Repository Map

```
cmd/kamps/              CLI entry point, help text, TTY detection
internal/config/        XDG paths and default settings
internal/model/         Course, Assignment, Material, Triage
internal/source/        Source interface + bundled demo dataset
internal/store/         state.json: load, apply, atomic save
internal/tui/           Bubble Tea app and all rendering
docs/                   developer guides
TUI_RESEARCH.md         design research and proposals (not a status page)
```

Build and run:

```sh
go build -o kamps ./cmd/kamps
./kamps
```

Verify before committing:

```sh
gofmt -l .
go vet ./...
go test ./...
```

A non-empty `gofmt -l .` output means a file needs formatting. The repository's commit style is a plain imperative one-line subject, with an optional body. Do not touch or stage the untracked `.commandcode/` directory.

---

## 2. Recommended Reading Order

1. [`cmd/kamps/main.go`](../cmd/kamps/main.go) — how the binary chooses TUI vs. plain mode.
2. [`internal/config/config.go`](../internal/config/config.go) — paths and the minimal config shape.
3. [`internal/source/source.go`](../internal/source/source.go) — the one interface the UI is built against.
4. [`internal/model/model.go`](../internal/model/model.go) — the domain types and triage values.
5. [`internal/store/store.go`](../internal/store/store.go) — local state and atomic writes.
6. [`internal/tui/app.go`](../internal/tui/app.go) — the root model and event loop.
7. [`internal/tui/tree.go`](../internal/tui/tree.go) — the flat node contract and row rendering.
8. [`internal/tui/search.go`](../internal/tui/search.go) — search mode and predicates.
9. [`internal/tui/detail.go`](../internal/tui/detail.go) — split pane and detail content.
10. The remaining TUI files: `frame.go`, `row.go`, `keys.go`, `theme.go`, `plain.go`, `views.go`.
11. Tests in `internal/tui`, `internal/store`, and `internal/tui/plain_test.go`.

---

## 3. Data Path

```
source.Source
    │  Courses / Assignments / Materials
    ▼
loadCmd (tea.Cmd, sequential calls)
    │  loadedMsg or errMsg
    ▼
App.Update
    │  sort, applyState, clampCursor
    ▼
visible() / visibleMaterials() / taskNodes() / matNodes()
    │
    ▼
View (trees, detail, commit, footer)
```

Important details:

- `loadCmd` is asynchronous from Bubble Tea's perspective but calls the three source methods sequentially. The first error aborts the rest.
- Courses are sorted by name; materials by course code, section, then name. Assignments are sorted only when the visible Tasks list is built: dated items first, oldest first, then by ID.
- `store.State.Apply` overlays saved decisions and download flags. Saved decisions win where they exist; otherwise the source's `Triage` value remains.
- `pending` is the session-only export queue. Decisions are saved immediately; the queue is not restored after restart.
- Plain mode uses the same source and store but skips Bubble Tea. It applies saved state and ignores source errors.

---

## 4. TUI File Map

| File | Read this when… |
|---|---|
| [`app.go`](../internal/tui/app.go) | You need the `App` struct, tabs, key dispatch, triage, downloads, the commit queue, `View`, the shell, or the footer |
| [`search.go`](../internal/tui/search.go) | You are changing query behavior, match fields, or search-mode keys |
| [`tree.go`](../internal/tui/tree.go) | You are changing tree structure, folding, connectors, row layout, filters, or counts |
| [`detail.go`](../internal/tui/detail.go) | You are changing the split pane, detail fields, or clipping |
| [`frame.go`](../internal/tui/frame.go) | You are changing the title row, rules, body height, tab cells, or hint fitting |
| [`row.go`](../internal/tui/row.go) | You are changing cursor/mark bands, gutters, carets, or selection cells |
| [`keys.go`](../internal/tui/keys.go) | You are adding or renaming a key that should appear in the `?` overlay |
| [`theme.go`](../internal/tui/theme.go) | You are adding a semantic color or reusable style |
| [`plain.go`](../internal/tui/plain.go) | You are changing the non-TTY snapshot |
| [`views.go`](../internal/tui/views.go) | You need shared formatting helpers such as `taskLine`, `shortCode`, or `lastSyncLine` |

`nodeWeek` has been removed; neither tree builder emits week rows today.

---

## 5. The Node Contract

Both tree tabs render a flat `[]node`. A node carries:

- `kind` — `nodeTerm`, `nodeCourse`, `nodeBucket`, `nodeTask`, `nodeTaskMaterial`, `nodeMaterialBucket`, `nodeFile`, or `nodeMaterialAssignment`;
- `key` — stable identity for folding;
- `parents` — ancestor keys, also used to compute depth;
- `courseID`, `term`, `bucket`;
- `task` and/or `mat` payloads.

The flattened list is the single source of truth. The cursor is an index into it, so whatever is highlighted is exactly what `space`, `h`/`l`, `d`, and triage keys act on. If you change what a tree shows, build it in `taskNodes` or `matNodes`; do not add a parallel row calculation in `View`.

### Tasks hierarchy

```
term
└── course
    ├── needs decision
    │   ├── task
    │   │   └── attached material (only when the ID matches K… → M…:)
    ├── tracked
    └── closed (missed + ignored)
```

### Materials hierarchy

```
course
├── downloaded
│   └── file
│       └── linked assignment (attachments only)
└── new
    └── file
        └── linked assignment (attachments only)
```

Folding:

- Missing keys mean open.
- `expanded` stores explicit collapses (and opens) per tab.
- `taskNodeDefaultOpen` gives buckets their defaults: `closed` starts folded.
- `h` folds the focused container when it has visible children; otherwise it moves to the parent row. `l` opens.

Connectors are produced by `treeGuides` from the flat list and fall back to ASCII when `TERM=dumb` or contains `ascii`.

---

## 6. Change Recipes

### Add a key

1. Add a `key.Binding` in [`DefaultKeys`](../internal/tui/keys.go), including `key.WithHelp`.
2. Add it to `ShortHelp` and/or `FullHelp` if it belongs in the overlay.
3. Handle `msg.String()` in the normal-mode switch in [`handleKey`](../internal/tui/app.go).
4. If it should be captured while typing, add it to [`handleSearchKey`](../internal/tui/search.go) instead.
5. Update the relevant footer hints: `taskFooterHints` in `frame.go` or the `hints` slice in `tabContent`.
6. Add a test in `internal/tui/app_test.go` or `tree_test.go`.
7. Update `README.md` and `kamps help` if the key is user-facing.

### Add a filter

1. Append the name to `filterNames` (Tasks) or `materialFilterNames` (Materials) in [`app.go`](../internal/tui/app.go).
2. Teach `matchesFilter` or `matchesMaterialFilter` the new case.
3. Check `clearFilter` and `cycleFilter` behavior.
4. Add a test covering the filter alone and composed with search.
5. Confirm the title line and footer still fit at 80 columns.

### Add a tab

1. Add a constant before `tabCount` in `app.go` and a case in `Tab.String`.
2. Extend `tabBar` labels and counts.
3. Add a `tabContent` case for title, hints, and rows.
4. Update `move` and `clampCursor`.
5. Update the number-key handling if the tab exceeds four.
6. Add tests for shell width and tab switching.

### Add a source

1. Implement `Label`, `Courses`, `Assignments`, and `Materials` from [`source.Source`](../internal/source/source.go).
2. Return copies, as `Demo` does, or otherwise guarantee the UI cannot mutate shared state.
3. Select the source in `main.go`; the UI should not need changes.
4. Keep the three calls consistent. The current interface makes that awkward for a network source; a cached snapshot inside the source is the simplest fix until the interface is refactored.
5. Remember plain mode calls the same methods sequentially.
6. Add tests with `httptest` or fixtures; never require a live session for `go test ./...`.

### Change persisted state

1. Add or change fields in [`store.State`](../internal/store/store.go).
2. Update `New`, `ensureMaps`, `Apply`, and the relevant setters.
3. If the JSON shape changes incompatibly, bump `version` and decide how old files are handled. There is currently no migration code.
4. Keep writes atomic and `0600`.
5. Add a round-trip test and a corrupt-file test. Use `t.TempDir()`; never write to the real home directory.

### Add detail content

1. Add fields to `detailLines` (Tasks) or `materialDetailLines` (Materials) in [`detail.go`](../internal/tui/detail.go).
2. Use `addKV` for label/value rows and `addWrapped` for prose.
3. Keep every line inside `width`; `detailRows` clips at pane height and appends `…`.
4. Add a rendering test asserting width and content.

### Add an export format

The commit preview is built in [`commitPreviewText`](../internal/tui/app.go) and each line comes from [`taskLine`](../internal/tui/views.go). If you add a real writer, keep the preview and the written output identical; otherwise the “review before write” promise is broken.

---

## 7. Testing Guide

### Commands

```sh
go test ./...
go test ./internal/tui -run TestSearch -v
go test -count=1 ./...
```

### Test helpers

[`demoApp`](../internal/tui/tree_test.go) builds an app against the demo source with a temporary state directory:

```go
func demoApp(t *testing.T) *App {
    t.Helper()
    return newDemoApp(t, config.Config{StateDir: t.TempDir()})
}

func newDemoApp(t *testing.T, cfg config.Config) *App {
    t.Helper()
    app := New(source.NewDemo(), cfg, "test")
    app.Update(app.Init()())
    return app
}
```

`app.Init()()` synchronously executes the load command, so tests get a fully loaded model without starting Bubble Tea. Use `newDemoApp` when you need to reuse a state directory across restarts.

### What the tests cover

- **Shell/layout:** row widths, rules instead of boxes, footer priority, small-terminal notice.
- **Trees:** grouping, sorting, folding, connectors, right padding, cursor vs. mark styling.
- **Search:** activation, narrowing, compound input, `ctrl+w`, `enter`/`esc` semantics, materials, filter composition, tab-switch clearing.
- **Detail:** auto-open threshold, toggling, content, ellipsis clipping, split widths.
- **Commit:** exact preview lines, grouping, drop behavior, empty queue, clipping.
- **Persistence:** restart survival, `undecided` deletion, downloads, corrupt fallback, missing file, directory creation.
- **Plain mode:** saved decisions and downloads appear in the snapshot.

### Test-writing rules

- Do not write to `~/.local/state`; use `t.TempDir()`.
- Avoid executing `clipboardCmd` in tests. Test the preview text, not the OSC 52 write.
- Prefer asserting visible content and widths over raw ANSI bytes.
- When a test sets `app.tab` directly, remember that `switchTab` is what normally restores per-tab state.
- Time-dependent tests should avoid assumptions about the exact current date; compare relative values where possible.

---

## 8. Running & Debugging

```sh
go build -o kamps ./cmd/kamps
./kamps                 # interactive, alternate screen
./kamps | cat           # plain snapshot
./kamps --no-anim       # disable spinner/progress motion
TERM=dumb ./kamps       # ASCII tree connectors
```

Useful checks:

- Resize below 80×24 to see the “terminal too small” path.
- Resize across 100 columns to see the detail pane auto-open/close.
- Press `?` to inspect the full key map.
- Remove or move `$XDG_STATE_HOME/kamps/state.json` to test a clean first run. Do this only when you do not need the saved decisions.
- To test corrupt-state handling, write invalid JSON to that path and restart.
- A copy failure usually means the terminal does not support OSC 52; the toast reports it.

---

## 9. Verified Today vs. Future

| Feature | Status |
|---|---|
| Demo source | Implemented; bundled in `internal/source/demo.go` |
| Moodle/Kulon2 source | Not implemented; recon notes in [kulon2.md](kulon2.md) |
| Config file (`config.toml`) | Not implemented; `config.Default()` only |
| Cookie auth / expiry | Not implemented |
| Live search | Implemented on Tasks and Materials; composes with filters |
| Saved decisions | Implemented; applied at load and in plain mode |
| Saved download flags | Implemented |
| Actual file downloads | Not implemented; `d` flips and persists a flag |
| Sync | Simulated animation and toast; does not reload the source |
| Commit writer | Not implemented; `y`/`enter` copies OSC 52 |
| Commit queue persistence | Not implemented; session-only by design |
| Attachment → assignment linking | Implemented for the `K…` / `M…:` ID convention |
| Week-grouped materials | Not implemented; no week rows are emitted |
| Working-directory cross-check | Demo stand-in (`similarManualTask`) |

When documenting a change, keep this table honest. The fastest way for docs to rot is to describe planned behavior as shipped behavior.

---

## 10. Gotchas and Conventions

- **Borderless shell:** separators are rules, not boxes. Do not reintroduce pane borders without revisiting the layout tests.
- **Right inset:** both trees reserve two columns on the right; the cursor band still reaches the edge.
- **Ellipsis:** long text ends with `…` by display width, not rune count.
- **Detail threshold:** 100 columns; split is 62/38 with a 24-column detail minimum.
- **Keys:** `/` is search, `f` is filter. Do not bind `v` for views.
- **Search clearing:** switching tabs clears search; `esc` clears search before filter before selections.
- **Queue semantics:** decisions persist immediately; the queue is an export buffer and is not restored.
- **State path:** `$XDG_STATE_HOME/kamps/state.json` or `~/.local/state/kamps/state.json`; `statePath == ""` disables persistence.
- **Atomic writes:** temp file, `Sync`, `chmod 0600`, rename. Keep that order.
- **No real home in tests:** use `t.TempDir()`.
- **Untouched directory:** never stage `.commandcode/`.
