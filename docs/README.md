# Kamps Developer Documentation

Welcome to the **Kamps** codebase documentation. This documentation is written for developers who are new to Kamps and may be unfamiliar with the Go Terminal User Interface (TUI) ecosystem.

---

## What is Kamps?

[`kamps`](../cmd/kamps/main.go) is a fast, keyboard-driven terminal application that pulls coursework — courses, assignments, and learning materials — from a university learning management system (LMS) such as Moodle/Kulon2. It is also designed to accommodate material from lab drives and hand-outs.

Instead of keeping dozens of browser tabs open and losing track of deadlines in forgotten spreadsheets, Kamps aggregates everything into a structured terminal view. Users make simple, sticky triage decisions on each assignment (`undecided`, `tracked`, `missed`, `ignored`), review uncommitted decisions in an export queue, and copy them directly to their clipboard via terminal escape sequences (OSC 52) to feed their personal task manager.

```
LMS / lab drive data (read-only)
              │
              ▼
       source.Source
              │
              ▼
  internal/model (Course, Assignment, Material)
              │
              ▼
  internal/store.Apply ◄──── state.json
  (decisions + download flags)
              │
              ▼
  internal/tui.App (Bubble Tea)
       │             │
       ▼             ▼
  Alt-screen UI   Plain-mode snapshot
       │
       ▼
  Commit preview ──▶ OSC 52 clipboard
```

The UI ships with both bundled demo data and an early live Kulon2 course-list source. `kamps login` verifies and securely saves a session; live mode fetches courses only. Assignments, materials, and actual downloads are not implemented yet; see [Kulon2 / Moodle notes](kulon2.md).

---

## Documentation Roadmap

The documentation is organized progressively from high-level architecture down to line-by-line TUI mechanics:

| Guide | Focus & Learning Goals | Key Topics |
|---|---|---|
| [1. Architecture](architecture.md) | How the system is designed | Package structure, verified component relationships, dependency map, data boundary, known seams |
| [2. Execution Flow](execution-flow.md) | How code executes at runtime | Startup, TTY detection, async data loading, keyboard event cycle, state persistence, rendering sequence |
| [3. TUI Fundamentals](tui-fundamentals.md) | How terminal UIs work in Go | The Elm Architecture (Bubble Tea Model-Update-View), Msgs, Cmds, Bubbles widgets, Lip Gloss styling, full-width row banding |
| [4. Source Code Tour](code-tour.md) | Where to find things and how to change them | File directory, reading order, investigation recipes, testing guide, verified vs future behavior |
| [5. Learning Path](learning-path.md) | A guided route for a Go newcomer | 30/60/120-minute tours, Go concepts mapped to files, four small exercises |
| [6. Kulon2 / Moodle Recon](kulon2.md) | Platform notes for live LMS integration | Real LMS endpoints, session cookie auth, AJAX formats, verified scraping constraints |

Appendix: [TUI research](../TUI_RESEARCH.md) records the design comparison that informed the tree, row, and theme work. Some ideas there are proposals rather than current behavior; the code tour marks the boundary.

---

## Core Mental Model & Terminology

- **Triage decision:** The user's local verdict on an assignment ([`model.Triage`](../internal/model/model.go)). Not fetched from the LMS. Options are `Undecided` (default), `Tracked` (handled or submitted), `Missed` (deadline passed), and `Ignored` (optional or outdated materials). The demo dataset seeds some of these values so the UI has realistic counts; saved local state wins when it has an entry for that assignment.
- **The source:** The read-only data provider ([`source.Source`](../internal/source/source.go)). It is satisfied by the in-memory [`source.Demo`](../internal/source/demo.go) or the Kulon2 source, which currently returns enrolled courses only.
- **The store:** Atomically persisted user state ([`store.State`](../internal/store/store.go)) saved to `$XDG_STATE_HOME/kamps/state.json` (`~/.local/state/kamps/state.json` when the variable is unset). It holds assignment triage decisions and downloaded-material flags. Missing entries mean `undecided` and `not downloaded`.
- **The commit queue:** The list of decisions changed during this process, held only in `App.pending`. Decisions are saved to disk immediately; the queue is the export step. Restarting Kamps does not restore the queue.
- **The Elm Architecture (TEA):** The design pattern powering Bubble Tea:
  - **Model:** A struct holding the complete application state ([`tui.App`](../internal/tui/app.go)).
  - **Update:** A pure function taking a message and returning an updated model plus an optional command ([`App.Update`](../internal/tui/app.go)).
  - **View:** A pure function transforming the current model into a printable string rendered to the terminal ([`App.View`](../internal/tui/chrome.go)).
- **Tree nodes:** A flattened 1D slice of tree elements ([`node`](../internal/tui/tree.go)) recalculated dynamically. This unified slice guarantees that row rendering, cursor positioning, and selection operations always point to the same row.
- **Search vs. filter:** `/` enters a live, case-insensitive text search over names, IDs, sections, kinds, and courses. `f` cycles the active tab's state filter. The two compose; switching tabs clears the search but restores that tab's filter.

---

## Current Behavior at a Glance

| Area | Implemented today | Notes |
|---|---|---|
| Data source | Demo or Kulon2 | Live mode fetches in-progress courses and parses course-page activities; dates/attachments are not yet parsed |
| Persistence | Decisions and download flags | Atomic `state.json`; corrupt file falls back to empty with a footer warning |
| Tasks tree | Term → course → state bucket → task → attached material | Folding is per tab; assignments identify attachments by the `K…`/`M…:` convention |
| Materials tree | Course → downloaded/new bucket → file → linked assignment | Attachments can link back to an assignment |
| Search | Live, case-insensitive, composes with `f` | `ctrl+w`, `alt+backspace`, and `ctrl+backspace` delete a word |
| Detail pane | Auto-opens at ≥ 100 columns; `enter` pins it | 62/38 split with a 24-column detail minimum |
| Downloads | Marks material as downloaded and persists it | Does not fetch bytes yet; default future library is `~/Documents/kamps` |
| Sync | Animated status and toast | Simulated; does not reload the source |
| Commit | Preview + copy via OSC 52 (`y` or `enter`) | `x` drops a queue item; the stored decision remains |
| Plain mode | Non-TTY snapshot | Applies saved state; source errors exit non-zero |

---

## Quick Start for Developers

To build and run Kamps locally:

```sh
# Clone and build
git clone <repo-url> kamps
cd kamps
go build -o kamps ./cmd/kamps

# Launch the interactive terminal UI
./kamps

# Run tests
go test ./...

# Run in non-interactive / pipeline mode (plain text snapshot)
./kamps | cat
```

Requirements: Go 1.24+.
