# Kamps TUI design

Status: **implemented**. The checklist below is the acceptance criteria; the
redesign tests in `internal/tui/redesign_test.go` and `theme_test.go` guard it.

This document is the redesign plan for the terminal UI: palette, layout,
persistent footer, help overlay, and focus/selection styling. It keeps the
borderless identity described in `CONTRIBUTING.md` (full-width rules, no boxes)
and the current key set. It is a presentation and consistency pass, not a
workflow rewrite.

## Principles

- [x] One semantic token per job. Widgets never read a hex value; they use
      `themeTokens` / the style set.
- [x] Color is never the only carrier of meaning. Every state also has a glyph
      or a label.
- [x] Focus and selection are independent and both survive monochrome.
- [x] The frame is borderless: tab strip, one title row, two rules, body,
      footer.
- [x] Every rendered line is exactly the terminal width; nothing spills or is
      silently clipped.
- [x] The interface degrades by dropping secondary information, never by
      blanking the screen.

## 1. Palette and theme resolution

- [x] Add `theme = "auto" | "dark" | "light" | "mono"` to config and a
      `--theme` flag. Invalid values fail with the accepted list.
- [x] Resolve one concrete theme before the first render:
      - `auto` uses `lipgloss.HasDarkBackground()`;
      - `dark` / `light` force that variant;
      - `mono` uses no color;
      - `NO_COLOR` / `CLICOLOR=0` force `mono` before anything else.
- [x] Rebuild the style set once in `New()` via `applyTheme(cfg.Theme)`. Keep
      the existing style names (`stRow`, `stCursor`, …) so call sites stay
      stable.
- [x] Move every widget to the resolved tokens; no hardcoded hex outside
      `theme.go`.
- [x] Token changes (text ≥ 4.5:1, non-text glyphs/bands ≥ 3:1 or band ≥ 1.5:1;
      enforced by `TestPaletteContrast`):

| Token | Current | Proposed |
|---|---|---|
| `Subtle` | light `#8A8A8A` = 3.45:1, used for hints/tabs/empty | light `#6B6B75` (~5.3:1), dark `#9A9AA4` (~5.9:1) |
| `Done` | light `#1F8A4C` = 4.38:1 | light `#177A43` (~5.4:1), dark `#6FCF97` |
| `Tracked` | light `#A5680F` low margin | light `#96590B` (~5.6:1), dark `#E0AF68` |
| `Surface` (cursor band) | 1.42:1 dark / 1.27:1 light | dark `#3A3A5C` (~1.6:1), light `#D4D4DE` (~1.5:1) |
| `Marked` (selection band) | 1.64:1 dark / 1.28:1 light | dark `#4D4478` (~2.0:1), light `#CFCFE8` (~1.7:1) |
| `Guide` (new) | hierarchy uses plain spaces | `#8A8A8A` both modes, for tree guides/separators |
| `Accent` | overused | light `#3F4390`, dark `#9AA0D0`; focus/mode only |
| `Overdue` | same hue as `Missed`, color-only | same `Danger` hue **plus bold and a leading `!`** |
| `Bg` | painted on tab cells, mismatches user terminal | stop painting it; transparent default |
| Course accents | one equals `Tracked` | keep the 6-color ramp, never use it for state text; verify ≥ 4.5:1 |

- [x] Do not paint a background equal to the assumed canvas. Only the cursor
      band, the mark band, and the active-tab chip set a background, and each is
      paired with a glyph or bold cue.
- [x] Keep the five state glyphs: `○ new`, `◐ tracked`, `◆ ignored`, `× missed`,
      `! overdue`. Add `▪ selected` and `▫ partial`.
- [x] Remove the impossible `● done` legend entry, or map a real state to it.
- [x] Mono mode: active tab = bold + underline, cursor = `▌` + bold, selection
      = `▪`, overdue = `!`, statuses = text labels, no color codes.

## 2. Layout structure

- [x] Replace the hard `w < 80 || h < 24` gate with four size classes resolved
      each frame:

| Class | Trigger | Layout |
|---|---|---|
| `Wide` | ≥ 110 cols and ≥ 28 rows | header; tree 62% + detail 38% when detail is on; auto-open at `-1` |
| `Standard` | 80–109 cols and ≥ 24 rows | single tree pane; `enter` opens detail full-width as a drill-down; footer 2 rows |
| `Compact` | 56–79 cols or 16–23 rows | single pane; hide progress bars but keep `8/9`; footer 1 row; drill-down detail |
| `Tiny` | < 56 cols or < 16 rows | `1..4` tab strip, one rule, tree, footer `q quit · ? help`; drill-down detail |

- [x] Below 40×10, show one line only:
      `terminal too small · q quit · ? help` — once, not three times.
- [x] `q` and `?` remain available in every class.
- [x] Body height: `h - 6` with a 2-row footer, `h - 5` collapsed, `h - 4` in
      Tiny. Recompute on resize and clamp cursors without losing the stable
      key under the cursor.
- [x] Header: one title row with a breadcrumb on the left
      (`Tasks › overdue`, `search "report"`) and counts on the right. Remove
      the duplicated `courses · all · N assignments` wording.
- [x] Tab strip: `1 Tasks  2 Materials  3 Sync  4 Commit`; active is bold accent
      + underline, not a filled block.
- [x] Header status degrades in this order: full labels + counts → glyph+count
      (revive the unused `taskTotals`) → `N overdue` → nothing. It must not
      vanish at 80 columns.
- [x] Remove the unlabeled sparkline from the header. Keep the helper only if a
      labeled use appears.
- [x] Task tree: draw real guides (`│  `, `├─ `, `└─ `) per depth via
      `treeGuidesFor(nodes, index)`; guides use `Guide`, the cursor row uses
      `Accent`.
- [x] Materials: default sections **open** and folders closed, so files are
      visible on entry.
- [x] Fix the one-column clip in `materialTreeRow`: subtract the prefix width
      from `available`, matching `taskTreeRow`.
- [x] Detail pane: implement `detailPane = -1` as "auto at Wide"; Standard and
      Compact open a full-width drill-down instead of a cramped split.
- [x] Label the right-hand due column (`+14d` / `−23d`) in a header or the
      detail pane.

## 3. Persistent footer

- [x] Two rows below 20 terminal rows, one row at or below it.
- [x] Derive hints from `KeyMap` plus a per-tab priority list. No hardcoded
      hint strings; the footer and help must not drift.
- [x] Row 1 — contextual, priority-ordered:
      - Tasks: `↑↓ move · space select · t/m/i decide · / search · f filter · enter detail · ? all keys`
      - Materials: `↑↓ move · space select · d download · o open · / search · f filter · ? all keys`
      - Commit: `↑↓ move · w write · y copy · x drop · esc back · ? all keys`
      - Sync: `s sync · ? all keys`
      - Search / Palette / Help: their own hints, ending with `esc`.
- [x] Drop the lowest-priority hint first. Never drop `? all keys` or the
      mode's escape hint.
- [x] Row 2 — persistent status:
      - mode/tab chip (`TASKS`, `SEARCH`, `COMMANDS`, `HELP`);
      - `N selected` whenever > 0, plus `all` when a group is fully selected;
      - queue counts and active filter
        (`queue: 3 track · 2 missed`, `filter: overdue`);
      - offline/cache banner and the last action;
      - `err: <message>` persists until `esc` or a new successful action.
- [x] Toasts render in this row instead of disappearing after 20–40 ticks.
      Errors are durable and never the only record of a failure.
- [x] Relabel `c commit` to `c review` (it opens the Commit tab); `w` / `y`
      remain the commit actions.

## 4. Help overlay

- [x] Replace `help.FullHelpView` with a hand-built overlay.
- [x] Sections: `Navigation`, `Selection & bulk` (Tasks), `Materials`, `Queue`,
      `Modes`, `Legend`.
- [x] Show only keys active on the current tab/mode; omit the rest.
- [x] Two columns at ≥ 100 cols, one column below.
- [x] Scroll with `j` / `k` when taller than the body; show a scroll hint in
      the title; never clip silently.
- [x] Legend: `○ new  ◐ tracked  ◆ ignored  × missed  ! overdue  ▪ selected`.
      Drop `● done`.
- [x] `?` / `esc` / `q` close help (`q` closes, not quits).

## 5. Focus and selection

- [x] One `rowStyle` helper used by Tasks, Materials, Commit, Sync, and palette
      results.

| State | Band | Gutter/mark | Text |
|---|---|---|---|
| normal | none | blank mark cell | Primary/Dim |
| cursor | `Surface` | `▌` Accent | Bold primary |
| selected | `Marked` | `▪` Accent | Primary |
| cursor+selected | `Surface` | `▌` + `▪` | Bold primary |
| partial group | none | `▫` Accent | Primary |
| overdue | none | `!` Danger | name Primary, due Bold Danger |

- [x] Cursor and selection stay visually independent; neither relies on the
      band alone.
- [x] Replace `sharedFocusRow` in the palette and Commit tab with
      `paneRow(content, width, state)` so every list looks the same.
- [x] Always expose the selection count in footer row 2.
- [x] Search and palette inputs keep a visible `▌` bar cursor and a mode chip.
- [x] The detail pane never steals focus silently; the footer shows `esc back`
      when it is a drill-down.

## 6. Verification

- [x] `make check` (gofmt, vet, `go test ./...`).
- [x] `TestPaletteContrast`: every text pair ≥ 4.5:1 and every non-text glyph
      pair ≥ 3:1 on light and dark; mono asserts no color codes.
- [x] `TestLayoutBreakpoints`: 40×12, 56×16, 60×20, 79×30, 80×24, 100×23,
      100×30, 120×40, 160×50 —
      - every line ≤ width;
      - body height is exact;
      - `q` and `?` present in every class;
      - "terminal too small" only below 40×10 and only once;
      - Materials course rows never clip the progress counts.
- [x] `TestFooterStatus`: selection, queue, filter, and a persistent error
      survive redraws and resize; no duplicate hints.
- [x] `TestHelpSections`: Tasks/Materials/Commit show only relevant groups;
      no `● done`; scrolling works when content exceeds height.
- [x] `TestFocusRows`: cursor, selected, and cursor+selected render distinct
      gutter/glyph pairs; still distinct after stripping ANSI.
- [x] Resize test: Wide → Compact → Tiny → Wide keeps the cursor on the same
      stable node key, recomputes detail mode, and never spills.
- [x] Manual PTY matrix (tmux): every size class, every tab, help/palette/search
      open + resize, rapid resize, `NO_COLOR=1`, light and dark terminals, and
      shell/cursor restoration on exit.

## Implementation order

- [x] 1. Theme foundation: resolved palette, mono mode, contrast test.
- [x] 2. Unified row/focus/selection styling + selection count.
- [x] 3. Persistent 2-row footer with hints and durable status/errors.
- [x] 4. Tab-aware, scrollable help overlay.
- [x] 5. Size classes, compact/tiny layouts, drill-down detail, materials
      default-open and clip fix.
- [x] 6. Tree guides, header cleanup, status degradation (`taskTotals`).
- [x] 7. Resize/PTY verification, regressions, docs link.
