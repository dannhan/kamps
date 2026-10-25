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
7. [`internal/tui/task_tree.go`](../internal/tui/task_tree.go) and [`materials.go`](../internal/tui/materials.go) — the two flattened trees.
8. [`internal/tui/task_rows.go`](../internal/tui/task_rows.go) and [`material_rows.go`](../internal/tui/material_rows.go) — row rendering and tab bodies.
9. [`internal/tui/search.go`](../internal/tui/search.go) and [`filters.go`](../internal/tui/filters.go) — narrowing the trees.
10. [`internal/tui/detail.go`](../internal/tui/detail.go), [`detail_tasks.go`](../internal/tui/detail_tasks.go), and [`detail_materials.go`](../internal/tui/detail_materials.go) — the split pane.
11. [`internal/tui/commit.go`](../internal/tui/commit.go), [`sync.go`](../internal/tui/sync.go), and [`chrome.go`](../internal/tui/chrome.go) — the other tabs and the shell.
12. [`internal/plain/plain.go`](../internal/plain/plain.go) and [`internal/format/format.go`](../internal/format/format.go) — the non-TTY snapshot and the formatting it shares with the TUI.
13. Tests next to the feature they cover, starting with [`testhelpers_test.go`](../internal/tui/testhelpers_test.go).

---

## 3. Data Path

```
source.Source
    │  Load(ctx) -> source.Snapshot
    ▼
loadCmd (tea.Cmd, one call)
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

- `loadCmd` is asynchronous from Bubble Tea's perspective but makes one `Load` call. The snapshot contains courses, assignments, and materials; any error aborts the load.
- Courses are sorted by name; materials by course code, section, then name. Assignments are sorted only when the visible Tasks list is built: dated items first, oldest first, then by ID.
- `store.State.Apply` overlays saved decisions and download flags. Saved decisions win where they exist; otherwise the source's `Triage` value remains.
- `pending` is the session-only export queue. Decisions are saved immediately; the queue is not restored after restart.
- Plain mode uses the same source and store but skips Bubble Tea. It returns source errors to the CLI instead of printing a partial snapshot; its code lives in [`internal/plain`](../internal/plain/plain.go).

---

## 4. TUI File Map

| File | Read this when… |
|---|---|
| [`app.go`](../internal/tui/app.go) | You need the `App` struct, construction, messages, `Update`, or `Tab` |
| [`keys.go`](../internal/tui/keys.go) | You are adding a key, changing dispatch, or updating the `?` overlay |
| [`navigation.go`](../internal/tui/navigation.go) | You are changing tab switching, cursors, folding, or selection |
| [`filters.go`](../internal/tui/filters.go) | You are adding or cycling a state filter |
| [`tasks.go`](../internal/tui/tasks.go) | You are changing task filtering, target resolution, triage, or the task footer |
| [`task_tree.go`](../internal/tui/task_tree.go) | You are changing the task hierarchy, grouping, or attachment association |
| [`task_rows.go`](../internal/tui/task_rows.go) | You are changing a task row, count, total, or the Tasks tab body |
| [`detail_tasks.go`](../internal/tui/detail_tasks.go) | You are changing task detail content |
| [`materials.go`](../internal/tui/materials.go) | You are changing material filtering, tree data, downloads, or attachment links |
| [`material_rows.go`](../internal/tui/material_rows.go) | You are changing a material row or the Materials tab body |
| [`detail_materials.go`](../internal/tui/detail_materials.go) | You are changing material detail content |
| [`commit.go`](../internal/tui/commit.go) | You are changing the session queue, preview, clipboard, or Commit tab |
| [`sync.go`](../internal/tui/sync.go) | You are changing sync, cursor restoration, session status, or the Sync tab |
| [`chrome.go`](../internal/tui/chrome.go) | You are changing the shell, tab bar, footer, or help overlay |
| [`detail.go`](../internal/tui/detail.go) | You are changing split-pane visibility, widths, or clipping |
| [`search.go`](../internal/tui/search.go) | You are changing query behavior, match fields, or search-mode keys |
| [`state.go`](../internal/tui/state.go) | You are changing how App applies or saves local state |
| [`tree.go`](../internal/tui/tree.go) | You are changing the shared node contract, fold keys, or connectors |
| [`frame.go`](../internal/tui/frame.go) | You are changing the title row, rules, body height, tab cells, or hint fitting |
| [`row.go`](../internal/tui/row.go) | You are changing cursor/mark bands, gutters, carets, or selection cells |
| [`theme.go`](../internal/tui/theme.go) | You are adding a semantic color or reusable style |
| [`text.go`](../internal/tui/text.go) | You are changing width handling, padding, truncation, or wrapping |
| [`window.go`](../internal/tui/window.go) | You are changing viewport windows or scroll hints |
| [`format.go`](../internal/tui/format.go) | You need shared formatting helpers such as `taskLine`, `shortCode`, or `lastSyncLine` |
| [`numbers.go`](../internal/tui/numbers.go) | You need a small integer helper |

Plain mode lives outside this package in [`internal/plain/plain.go`](../internal/plain/plain.go). Shared byte-size formatting and assignment ordering live in [`internal/format/format.go`](../internal/format/format.go), so plain mode never imports Bubble Tea.

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
3. Handle `msg.String()` in the normal-mode switch in [`handleKey`](../internal/tui/keys.go).
4. If it should be captured while typing, add it to [`handleSearchKey`](../internal/tui/search.go) instead.
5. Update the relevant footer hints: `taskFooterHints` in `frame.go` or the `hints` slice in the tab content function.
6. Add a test in the feature file, such as `search_test.go`, `navigation_test.go`, or `task_rows_test.go`.
7. Update `README.md` and `kamps help` if the key is user-facing.

### Add a filter

1. Append the name to `filterNames` (Tasks) or `materialFilterNames` (Materials) in [`filters.go`](../internal/tui/filters.go).
2. Teach `matchesFilter` (in `tasks.go`) or `matchesMaterialFilter` (in `materials.go`) the new case.
3. Check `clearFilter` and `cycleFilter` behavior.
4. Add a test covering the filter alone and composed with search.
5. Confirm the title line and footer still fit at 80 columns.

### Add a tab

1. Add a constant before `tabCount` in [`app.go`](../internal/tui/app.go) and a case in `Tab.String`.
2. Extend `tabBar` in [`chrome.go`](../internal/tui/chrome.go).
3. Add a tab content function next to the feature (for example `tasksContent`), and wire it into `tabContent`.
4. Update `move` and `clampCursor` in [`navigation.go`](../internal/tui/navigation.go).
5. Update the number-key handling if the tab exceeds four.
6. Add tests for shell width and tab switching.

### Add a source

1. Implement `Label` and `Load` from [`source.Source`](../internal/source/source.go).
2. Return a [`source.Snapshot`](../internal/source/source.go) with courses, assignments, and materials. Return copies, as `Demo` does, or otherwise guarantee the UI cannot mutate shared state.
3. Respect `ctx` so cancellation stops network work promptly.
4. Select the source in `main.go`; the UI and plain mode should not need changes.
5. Keep the snapshot internally consistent. Fetching shared LMS pages once inside `Load` is the point of the interface.
6. Add tests with `httptest` or fixtures; never require a live session for `go test ./...`.

### Change persisted state

1. Add or change fields in [`store.State`](../internal/store/store.go).
2. Update `New`, `ensureMaps`, `Apply`, and the relevant setters.
3. If the JSON shape changes incompatibly, bump `version` and decide how old files are handled. There is currently no migration code.
4. Keep writes atomic and `0600`.
5. Add a round-trip test and a corrupt-file test. Use `t.TempDir()`; never write to the real home directory.

### Add detail content

1. Add fields to `detailLines` in [`detail_tasks.go`](../internal/tui/detail_tasks.go) or `materialDetailLines` in [`detail_materials.go`](../internal/tui/detail_materials.go).
2. Use `addKV` for label/value rows and `addWrapped` for prose.
3. Keep every line inside `width`; `detailRows` clips at pane height and appends `…`.
4. Add a rendering test asserting width and content.

### Add an export format

The commit preview is built in [`commitPreviewText`](../internal/tui/commit.go) and each line comes from [`taskLine`](../internal/tui/format.go). If you add a real writer, keep the preview and the written output identical; otherwise the “review before write” promise is broken.

---

## 7. Testing Guide

### Commands

```sh
go test ./...
go test ./internal/tui -run TestSearch -v
go test -count=1 ./...
```

### Test helpers

[`demoApp`](../internal/tui/testhelpers_test.go) builds an app against the demo source with a temporary state directory:

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
