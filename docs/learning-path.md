# Learning Path

This is a route for a developer who knows a little Go but has never worked on
a Bubble Tea application, and who does not want to hold the whole repository in
their head at once. Each stop is short and points at one idea.

The goal is not to read every line. The goal is to make one safe change and
watch the tests keep you honest.

---

## 30 minutes: the shape of the program

Read these in order:

| Time | File | What to notice |
|---|---|---|
| 5 min | [`cmd/kamps/main.go`](../cmd/kamps/main.go) | The binary is thin: parse flags, choose TUI vs. plain mode, start the program. |
| 5 min | [`internal/source/source.go`](../internal/source/source.go) | The UI talks to an interface, not to Moodle directly. |
| 5 min | [`internal/model/model.go`](../internal/model/model.go) | Plain structs and a small enum. No framework types leak in here. |
| 5 min | [`internal/store/store.go`](../internal/store/store.go) | User state is JSON, loaded once, and overlaid after source data arrives. |
| 10 min | [`internal/tui/app.go`](../internal/tui/app.go), first ~200 lines | `App` is the root model. `Init` starts loading; `Update` handles events; `View` renders. |

After this pass, answer:

- Where does the data come from?
- Where do user decisions live?
- What happens when a key is pressed?
- Which file would you open to change the Tasks tab?

If the last answer is not obvious, that is the problem the code tour is meant
to fix. Start with [Source Code Tour](code-tour.md).

---

## 60 minutes: follow one interaction

Trace a triage decision end to end:

1. In [`internal/tui/keys.go`](../internal/tui/keys.go), find `handleKey` and the
   `t`/`m`/`i`/`n` cases.
2. Follow `applyTriage` in [`tasks.go`](../internal/tui/tasks.go).
3. Notice the three separate things it does:
   - changes the in-memory assignment;
   - records the decision in `store.State`;
   - adds or removes the item from the session-only `pending` queue.
4. Follow `saveState` in [`state.go`](../internal/tui/state.go) into `store.State.Save`.
5. Open `commitPreviewText` in [`commit.go`](../internal/tui/commit.go) and
   `taskLine` in [`format.go`](../internal/tui/format.go) to see how a queued
   decision becomes exported text.

Now trace a search:

1. `handleKey` sends keys to `handleSearchKey` while `searching` is true.
2. Search state is just two fields: `searching` and `search`.
3. `visible()` asks both the filter and the search predicate.
4. `taskNodes()` in [`task_tree.go`](../internal/tui/task_tree.go) builds the
   rows from that filtered list. The cursor indexes the same list, which is why
   the highlight and the keys cannot disagree.

This is the central design idea: **derive the visible rows once, then let
rendering and input share them**.

---

## 120 minutes: make a real change

Pick one exercise. Each is deliberately small, and each has an obvious test.

### Exercise 1: add a filter

Add an `undated` (or `dated`) filter for tasks.

Files:

- `filterNames` in [`filters.go`](../internal/tui/filters.go) and `matchesFilter` in [`tasks.go`](../internal/tui/tasks.go)
- a test in `search_test.go` or `task_rows_test.go`

Acceptance:

- `f` cycles to the new filter;
- the title shows the filter name and match count;
- `esc` clears it;
- search still composes with it.

### Exercise 2: add a detail field

Show the course term in the task detail pane.

Files:

- `detailLines` in [`internal/tui/detail_tasks.go`](../internal/tui/detail_tasks.go)
- a rendering test in `detail_test.go`

Acceptance:

- the field appears for a task with a term;
- the line stays inside the pane width;
- a long value ends with `…`.

### Exercise 3: add a test before a change

Find a key path that has no direct test, such as `d` on a course row or `x` on
the commit tab. Write a failing test first, then make the smallest change that
passes.

### Exercise 4: improve a stale comment or remove dead code

Search for comments that no longer match the code. Good candidates are layout
constants, demo-only helpers, and old design notes. Make a move-only or
delete-only change and run `make check`.

---

## Go concepts, mapped to this repo

| Go idea | Where to look |
|---|---|
| Structs and methods | `internal/model/model.go`, `App` in `internal/tui/app.go` |
| Interfaces | `source.Source` in `internal/source/source.go` |
| Errors as values | `store.Load`, `store.State.Save`, `errMsg` in the TUI |
| Slices and maps | Tree nodes, `selected`, `pending`, `expanded` |
| Sorting | `sort.Slice` in `visible`, `taskNodes`, `commitItems` |
| Pointer receivers | Almost every `func (m *App)` method |
| Commands and goroutines | `loadCmd`, `clipboardCmd`, `tea.Cmd` |
| Pure rendering | `View` and the row/detail builders |
| Table-driven tests | `internal/store/store_test.go` |
| Temporary files in tests | `t.TempDir()` and `testhelpers_test.go` |

---

## A good first contribution loop

```sh
./scripts/check.sh         # confirm the baseline is green
go run ./cmd/kamps         # try the behavior yourself
# edit one small thing
./scripts/check.sh         # verify formatting, vet, and tests
git diff                   # read the whole change before committing
```

When in doubt, make the change smaller. A one-file move or a five-line test is
a perfectly good contribution, and it keeps the codebase friendly for the next
tired person.
