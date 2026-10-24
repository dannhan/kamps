# Architecture & Package Structure

This document describes the architecture of Kamps, the responsibilities of each package, and the dependency boundaries that exist in the current codebase. It describes the build as it is, not the roadmap.

---

## 1. High-Level Architecture

Kamps follows a strict separation of concerns between **domain models**, **external data sources**, **persisted local state**, and the **terminal UI**.

The data boundary is simple:

- **LMS coursework data** is read-only external input received through [`source.Source`](../internal/source/source.go).
- **User decisions and local flags** are owned by the local machine and stored atomically in JSON via [`store.State`](../internal/store/store.go).
- The **TUI** ([`tui.App`](../internal/tui/app.go)) is the reconciler: it loads source data, overlays saved local state, captures user interactions, and renders the result.
- The **commit queue** is intentionally not persisted. It is a per-run export buffer; decisions themselves are saved immediately.

```mermaid
flowchart TD
    subgraph CLI["CLI Entry Point (cmd/kamps)"]
        Main["main() in main.go"]
        TTYCheck{"isTTY(stdout)?"}
    end

    subgraph ConfigPkg["Configuration (internal/config)"]
        Cfg["config.Config\nXDG paths + defaults"]
    end

    subgraph ModelPkg["Core Domain (internal/model)"]
        ModelCourse["model.Course"]
        ModelAssignment["model.Assignment"]
        ModelMaterial["model.Material"]
        ModelTriage["model.Triage enum"]
    end

    subgraph SourcePkg["Data Ingestion (internal/source)"]
        SourceInterface["source.Source interface"]
        DemoSource["source.Demo\nin-memory dataset"]
        DemoSource -. implements .-> SourceInterface
    end

    subgraph StorePkg["Persistence (internal/store)"]
        StoreState["store.State\ndecisions + download flags"]
        StateFile[("state.json\natomic write")]
        StoreState <--> StateFile
    end

    subgraph TUIPkg["Terminal UI (internal/tui)"]
        TUIApp["tui.App (tea.Model)"]
        Search["search.go\nlive query + predicates"]
        Tree["tree.go + task/material files\nflattened nodes + rows"]
        Detail["detail.go\nsplit pane"]
        Frame["frame.go / row.go\nshell, hints, bands"]
        PlainMode["plain.go\nPrintPlain()"]
    end

    subgraph External["Terminal & Clipboard"]
        AltScreen["Alternate screen\n(Bubble Tea)"]
        PlainOut["stdout\n(plain mode)"]
        OSC52["OSC 52\nsystem clipboard"]
    end

    Main --> Cfg
    Main --> DemoSource
    Main --> TTYCheck

    TTYCheck -- "No (pipe / file)" --> PlainMode
    TTYCheck -- "Yes (interactive)" --> TUIApp

    PlainMode --> SourceInterface
    PlainMode --> StoreState
    PlainMode --> PlainOut

    TUIApp --> SourceInterface
    TUIApp --> StoreState
    TUIApp --> Search
    TUIApp --> Tree
    TUIApp --> Detail
    TUIApp --> Frame
    TUIApp --> AltScreen
    TUIApp -. "y / enter (commit preview)" .-> OSC52

    SourceInterface --> ModelPkg
    StoreState --> ModelPkg
    TUIApp --> ModelPkg
    PlainMode --> ModelPkg
```

There is no HTTP server, database, or daemon. Kamps is a single local binary.

---

## 2. Package Responsibilities

| Package | Path | Responsibility | Key Symbols |
|---|---|---|---|
| `main` | [`cmd/kamps`](../cmd/kamps/main.go) | CLI entry point, flag parsing (`--no-anim`, `version`, `help`), TTY detection, runtime mode selection. | [`main()`](../cmd/kamps/main.go), [`isTTY()`](../cmd/kamps/main.go), [`printHelp()`](../cmd/kamps/main.go) |
| `config` | [`internal/config`](../internal/config/config.go) | Resolve XDG state/config directories, define defaults, return the state file path. There is no config-file loader yet. | [`Config`](../internal/config/config.go), [`Default()`](../internal/config/config.go), [`Config.StatePath()`](../internal/config/config.go) |
| `model` | [`internal/model`](../internal/model/model.go) | Pure domain types. Defines courses, assignments, materials, and the triage enum. | [`Course`](../internal/model/model.go), [`Assignment`](../internal/model/model.go), [`Material`](../internal/model/model.go), [`Triage`](../internal/model/model.go) |
| `source` | [`internal/source`](../internal/source/source.go) | Abstract data-provider interface (`Source`) and the bundled demo implementation (`Demo`). | [`Source`](../internal/source/source.go), [`Demo`](../internal/source/demo.go), [`NewDemo()`](../internal/source/demo.go) |
| `store` | [`internal/store`](../internal/store/store.go) | Atomically persist triage decisions and download flags to JSON; load and apply them over fresh source data. | [`State`](../internal/store/store.go), [`Load()`](../internal/store/store.go), [`State.Save()`](../internal/store/store.go), [`State.Apply()`](../internal/store/store.go) |
| `tui` | [`internal/tui`](../internal/tui/app.go) | Full Bubble Tea user interface: screen management, navigation, tree flattening, search, detail pane, styling, keyboard dispatch, and plain-mode rendering. | [`App`](../internal/tui/app.go), [`New()`](../internal/tui/app.go), [`PrintPlain()`](../internal/tui/plain.go) |

### TUI file map

| File | Responsibility |
|---|---|
| [`app.go`](../internal/tui/app.go) | Root model, tabs, messages, `Update`, app construction |
| [`keys.go`](../internal/tui/keys.go) | Key bindings, key dispatch, `?` overlay data |
| [`navigation.go`](../internal/tui/navigation.go) | Tab switching, cursors, folding, selection |
| [`filters.go`](../internal/tui/filters.go) | Filter names, cycling, clearing |
| [`tasks.go`](../internal/tui/tasks.go) | Task filtering, target resolution, triage, task footer |
| [`task_tree.go`](../internal/tui/task_tree.go) | Flattened task hierarchy and grouping |
| [`task_rows.go`](../internal/tui/task_rows.go) | Task rows, counts, totals, Tasks tab body |
| [`detail_tasks.go`](../internal/tui/detail_tasks.go) | Task detail content |
| [`materials.go`](../internal/tui/materials.go) | Material filtering, tree data, downloads, attachment links |
| [`material_rows.go`](../internal/tui/material_rows.go) | Material rows and Materials tab body |
| [`detail_materials.go`](../internal/tui/detail_materials.go) | Material detail content |
| [`commit.go`](../internal/tui/commit.go) | Session queue, preview, clipboard, Commit tab |
| [`sync.go`](../internal/tui/sync.go) | Simulated sync state and Sync tab |
| [`chrome.go`](../internal/tui/chrome.go) | Shell, tab bar, footer, help overlay |
| [`detail.go`](../internal/tui/detail.go) | Split-pane visibility, widths, clipping |
| [`search.go`](../internal/tui/search.go) | Search state, title, predicates, search-mode keys |
| [`state.go`](../internal/tui/state.go) | App-to-store apply/save bridge |
| [`tree.go`](../internal/tui/tree.go) | Shared node contract, fold keys, connectors |
| [`frame.go`](../internal/tui/frame.go) | Title line, body clamping, tab cells, rules, footer hints |
| [`row.go`](../internal/tui/row.go) | Cursor/mark bands, gutters, carets, selection cells |
| [`theme.go`](../internal/tui/theme.go) | Semantic color tokens and reusable Lip Gloss styles |
| [`text.go`](../internal/tui/text.go) | Width, padding, truncation, wrapping |
| [`window.go`](../internal/tui/window.go) | Viewport windows and scroll hints |
| [`format.go`](../internal/tui/format.go) | Shared formatting helpers such as `taskLine`, `shortCode`, `lastSyncLine` |
| [`numbers.go`](../internal/tui/numbers.go) | Small integer helpers |
| [`plain.go`](../internal/tui/plain.go) | Non-TTY snapshot with saved state applied |

---

## 3. Verified Dependencies & Ecosystem Usage

From [`go.mod`](../go.mod) and direct code imports:

### Direct Dependencies

1. **`github.com/charmbracelet/bubbletea` (v1.3.10)**
   - Used for the core event loop: [`tea.Model`](../internal/tui/app.go), [`tea.Msg`](../internal/tui/app.go), [`tea.Cmd`](../internal/tui/app.go), and [`tea.NewProgram`](../cmd/kamps/main.go).
   - Kamps runs the interactive UI in the alternate screen (`tea.WithAltScreen()`), then restores the shell on exit.
   - Manages terminal setup, key listeners, window resize events, and spinner ticks.

2. **`github.com/charmbracelet/bubbles` (v1.0.0)**
   - [`spinner.Model`](../internal/tui/app.go) drives the simulated sync indicator.
   - [`help.Model`](../internal/tui/app.go) and [`key.Binding`](../internal/tui/keys.go) provide the `?` help overlay.

3. **`github.com/charmbracelet/lipgloss` (v1.1.0)**
   - Declarative style engine for terminal rendering.
   - Provides [`lipgloss.AdaptiveColor`](../internal/tui/theme.go) for light/dark terminals and width measurement used throughout the layout.

4. **`github.com/charmbracelet/x/term` (v0.2.2)**
   - Used by [`isTTY`](../cmd/kamps/main.go) via `xterm.IsTerminal` to choose between the TUI and plain mode.

### Auxiliary / Indirect Libraries

- **`github.com/charmbracelet/x/ansi`**
  - Used by [`ansiTruncate`](../internal/tui/text.go) and tree rows to truncate styled strings without cutting ANSI escapes or multi-byte runes.

---

## 4. The Data Separation Boundary

```
[ LMS platform (Moodle / Kulon2) or demo data ]
                    │
                    │ read-only
                    ▼
          source.Source
                    │
                    ▼
    []model.Course / Assignment / Material
                    │
                    │ overlay local decisions
                    ▼
       store.State.Apply() ◄── $XDG_STATE_HOME/kamps/state.json
                    │
                    ▼
       TUI trees, detail pane, commit preview
```

1. **Source data is read-only in practice:** [`source.Source`](../internal/source/source.go) exposes `Label`, `Courses`, `Assignments`, and `Materials`. `Demo` returns copies of its slices, so UI mutations cannot corrupt the dataset.
2. **Saved state is an overlay:** when a decision exists in the store, [`store.State.Apply`](../internal/store/store.go) writes it over the source value. If there is no stored entry, the source's `Triage` value remains. Real sources should therefore return `Undecided`; the demo deliberately seeds a few states to make its screens realistic.
3. **Undecided deletes the entry:** storing `Undecided` removes the decision instead of writing noise.
4. **Atomic writes:** decisions are saved immediately through a temporary file in the same directory, `Sync()`, `chmod 0600`, and an atomic rename.
5. **The queue is not state:** `App.pending` tracks what this run would export. `x` drops an item from the queue without reverting the stored decision.

### Load-time behavior

- A **missing** `state.json` is not an error; Kamps starts empty.
- A **corrupt** file returns an error. `tui.New` keeps an empty state, and the first `loadedMsg` shows a footer warning. Kamps does not overwrite the corrupt file until the next successful save.
- Unknown or zero triage values are dropped while loading.
- Download flags are applied only when true; absent means not downloaded.

### Plain mode

[`tui.PrintPlain`](../internal/tui/plain.go) uses the same source and store, but no Bubble Tea loop:

1. Call `Courses`, `Assignments`, and `Materials` synchronously.
2. Load and apply saved state when a state path is configured.
3. Sort assignments by due date (dated first, oldest first, then ID).
4. Print a plain-text snapshot.

Plain mode is deliberately best-effort: source errors are currently ignored, and a corrupt state file is silently skipped.

---

## 5. TUI Composition

### Shell

The interactive screen is borderless. The top tab bar, title line, body, and footer are separated by full-width rules rather than boxes. This leaves every column for content and keeps the split pane from sitting between two borders.

- Minimum comfortable size: 80 columns × 24 rows. Below that, Kamps renders a short “terminal too small” notice.
- Header: tab names with live counts and a status label.
- Title row: current filter or search query, match/total counts, scroll hint.
- Body: the active tab's rows, clamped to the available height.
- Footer: key hints on the left, queue/toast on the right.

### Trees

Both tree tabs render a flat `[]node` list. [`node`](../internal/tui/tree.go) stores its kind, depth via `parents`, a stable fold key, and the underlying assignment or material. The cursor is an index into the same list, so a highlight always corresponds to the row the keys will act on.

- **Tasks:** term → course → state bucket (`needs decision`, `tracked`, `closed`) → task → attached materials.
- **Materials:** course → `downloaded`/`new` bucket → file → linked assignment (for attachments).
- **Connectors:** `│`, `├─`, `╰─`, with an ASCII fallback when `TERM=dumb` or contains `ascii`.
- **Folding:** missing keys mean open; collapsing is the explicit act. Each tab keeps its own fold state.
- **Right padding:** both trees keep a two-column inset so metadata does not touch the divider or terminal edge.

### Search and filters

- `/` is search, not a view toggle. It is available on Tasks and Materials.
- Search is live, case-insensitive, and trims the query before matching.
- Tasks match name, ID, goal, and course name/code/short label.
- Materials match name, ID, section, kind, course code, and course name/code/short label.
- `f` cycles the tab's state filter; search and filter compose.
- Switching tabs clears the search and restores that tab's filter.
- `esc` clears the search first, then the filter, then selections/back.

### Detail pane

- Visible on Tasks and Materials only.
- Auto-opens when `width >= 100`; `enter` pins it open or closed.
- Split: 62% tree / remaining detail, with a 24-column minimum for detail.
- Long descriptions wrap; clipped detail content ends with `…`.

### Commit queue

- `t`, `m`, `i`, `u` update in-memory assignments, store the decision, and add/remove the item in `App.pending`.
- `c` opens the Commit tab; `esc` returns to the previous tab.
- The preview groups items by action and shows the exact task lines.
- `y` (or `enter`) copies the preview with OSC 52; `x` removes the focused item from the queue only.
- The queue is session-only. Restarting Kamps keeps decisions but starts an empty queue.

---

## 6. Known Seams

These are current limitations, not mysteries:

- **The source interface has three data methods.** A network source would have to fetch the same course pages three times unless it caches internally. A future refactor may replace this with one snapshot call.
- **There is no `config.toml` loader yet.** [`config.Default`](../internal/config/config.go) only computes paths and defaults.
- **The demo source seeds triage values.** They are useful for screenshots but are not user state; real sources should return `Undecided`.
- **Downloads do not transfer bytes.** `d` flips the flag and saves it.
- **Sync is simulated.** `s` runs an animation and a toast; it does not reload the source.
- **The commit writer does not exist.** The queue copies text to the clipboard.
- **`similarManualTask` is a demo stand-in** for a future working-directory cross-check.
