# TUI research

> Status: design research from an earlier UI pass. This note compares an external reference with kamps and records proposals; some were not adopted. Current behavior is documented in [`docs/`](docs/).

Inspiration: [YoanWai/agent-manager](https://github.com/YoanWai/agent-manager) (Apache-2.0). I inspected its `go.mod`, `main.go`, rail renderer, theme palettes, and README/usage sections on Groups, Status, and Themes.

**Stack.** Reference: Go `1.27.2`; Bubble Tea `v1.3.10`, Bubbles `v1.0.0`, Lip Gloss `v1.1.0`, `charmbracelet/x/ansi v0.11.8`, `charmbracelet/x/cellbuf v0.0.15`, `charmbracelet/x/term v0.2.2`, `muesli/termenv v0.16.0`, `clipperhouse/displaywidth v0.11.0`, `mattn/go-runewidth v0.0.24`. Kamps: Go `1.24.2`; Bubble Tea `v1.3.10`, Bubbles `v1.0.0`, Lip Gloss `v1.1.0`, `x/term v0.2.2`, indirect `x/ansi v0.11.6`, `x/cellbuf v0.0.15`, `displaywidth v0.9.0`, `go-runewidth v0.0.19`, `termenv v0.16.0`. The reference's `main.go` runs Bubble Tea in alternate-screen mode at 120 FPS with mouse-cell-motion; kamps also uses the alternate screen, without mouse support. No TUI dependency or Go version upgrade is needed for this tree.

| Topic | Reference | Kamps | Adopt or keep ours |
|---|---|---|---|
| Libraries | Above; direct Bubble Tea family versions match | Same direct versions; Go 1.24.2 | Keep versions; don't raise Go baseline |
| Tree and connectors | Flattened rows retain depth/ancestor continuation; `│  `, `├─ `, `╰─ ` | Existing flat `node` list, course→assignment or course→week→file | Adopt reference guides and flat visible-row model; use ASCII `|  `, `|-- `, `` `-- `` fallback |
| Row layout | Compact single-line sessions/groups with secondary metadata right-aligned | Two-pane course/task rows with fixed columns | Adopt compact single-line tree rows and right metadata |
| Focus highlight | Every selected row painted across full width with selected surface | Cursor band spans rows, with a leading cursor bar | Keep band mechanic; start at connectors and fill whole row |
| Group counters | Nonzero state rollups, each state colored | Course row has shown/total count | Adopt right-aligned nonzero triage counts |
| `[⋯]` marker | Per-row button opens contextual menu | No row action menu; global triage keys exist | Omit; don't add a one-off menu |
| Theme tokens | `Bg`, `Surface`, `Overlay`, `Border`; `Bright`, `Text`, `Dim`, `Subtle`; `Accent`, `Accent2`; state tokens; palettes are token structs | Package-level adaptive colors/styles | Adopt token struct shape and a single adaptive palette; states are `new`, `tracked`, `done`, `ignored`, `missed` |
| Borders/panels | Shared theme tokens drive surfaces, borders, rails and selected backgrounds | Borderless shell with rules and a two-pane split | Keep the borderless shell; align its colors with tokens |
| Responsive width | Reserve row controls, use available width, elide optional metadata, preserve minimum list room | Existing width-aware pane split and Lip Gloss display-width truncation | Keep current layout; tree rows reclaim optional fields and truncate names by display cells |
| Key bindings | Up/down or `jk`, horizontal step/fold, enter; configurable actions | Up/down or `jk`, `h/l` folds tree, tabs and triage keys | Keep kamps bindings; `h/l` operate on focused group/bucket/assignment |
| Name truncation | ANSI/display-cell width, trailing `…` | Lip Gloss-aware `truncate`/`ansiTruncate` helpers | Keep existing helpers; truncate within computed row width |

**Design rules.**

- Keep a flat render list, with row depth and following-sibling information determining connector guides.
- Use `│  ` for an ancestor that continues, `├─ ` for a node with later siblings, and `╰─ ` for a last child.
- Keep group rows one line; align state summaries to the right and omit zero-count states.
- Fill the entire focused row, including its tree gutter; let the selected surface quiet nonessential text.
- Route all colors through semantic theme tokens; state colors identify state, not urgency.
- Truncate by terminal display width with `…`; drop optional cells before sacrificing the name.
- Keep the list visible at narrow widths and scroll its row window to keep focus in view.

**Domain boundary.** Kamps `model.Triage` has only `Undecided`, `Tracked`, `Missed`, and `Ignored`; it has no assignment `Done`. `model.Material` has a `Downloaded` boolean but no assignment foreign key. Only attachment IDs following the existing `M<digits-after-K>:` convention (for example `K165483` → `M165483:`) can be attached reliably. There is no task-directory field, `tasks.md` path, or parsed task-line type in `model`/`source`; `taskLine` formats a proposed string only. The tree therefore omits unavailable filesystem/task-line children and does not invent a fifth assignment state.
