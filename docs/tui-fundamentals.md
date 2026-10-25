# Terminal UI Fundamentals: Bubble Tea, Bubbles & Lip Gloss

This guide teaches the foundations of the Go Terminal User Interface (TUI) ecosystem through the concrete implementation of Kamps. If you have never built a TUI or worked with Charm's tools, this document explains why these libraries exist and how Kamps uses them.

---

## 1. The Elm Architecture (TEA)

In traditional command-line scripts, printing to the terminal is an uncoordinated sequence of `fmt.Println` or ANSI escape codes. In complex interactive applications with keyboard listeners, background work, animations, and window resizing, this approach leads to screen flicker, race conditions, and visual artifacts.

Kamps uses [**Bubble Tea**](https://github.com/charmbracelet/bubbletea), which implements **The Elm Architecture (TEA)**. TEA enforces unidirectional data flow through three core methods defined on the [`tea.Model`](../internal/tui/app.go) interface:

```
               ┌────────────────────────────────────────┐
               │                                        │
               ▼                                        │
┌──────────────────────────────┐                        │
│   Init() -> tea.Cmd          │                        │
│   (Kick off background work) │                        │
└──────────────┬───────────────┘                        │
               │                                        │
               ▼                                        │
┌──────────────────────────────┐        Msg             │
│   Update(Msg) -> (Model, Cmd)├────────────────────────┘
│   (State transition function)│
└──────────────┬───────────────┘
               │
               ▼
┌──────────────────────────────┐
│   View() -> string           │
│   (Pure rendering function)  │
└──────────────┬───────────────┘
               │
               ▼
       Terminal screen
```

### The Three Methods in Kamps

1. **[`Init() tea.Cmd`](../internal/tui/app.go):**
   Called once when the application starts. It returns an initial command to run in the background. In Kamps, it triggers asynchronous data loading:

   ```go
   func (m *App) Init() tea.Cmd { return loadCmd(m.src) }
   ```

2. **[`Update(tea.Msg) (tea.Model, tea.Cmd)`](../internal/tui/app.go):**
   The central event handler. It receives messages, updates fields on `*tui.App`, and optionally returns a command for further background work:

   ```go
   func (m *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
       switch msg := msg.(type) {
       case tea.WindowSizeMsg:
           m.width, m.height = msg.Width, msg.Height
           return m, nil
       case loadedMsg:
           m.loaded = true
           m.courses = msg.courses
           // ... apply saved state, clamp cursors
           return m, nil
       case tea.KeyMsg:
           return m.handleKey(msg)
       }
       return m, nil
   }
   ```

3. **[`View() string`](../internal/tui/chrome.go):**
   A pure function that converts the model's current state into a single formatted string representing the full terminal frame. `View` never modifies state; it only formats text and ANSI styles.

---

## 2. Messages (`tea.Msg`) and Commands (`tea.Cmd`)

### Messages (`tea.Msg`)

A message is an arbitrary Go value (`any`) representing an event that occurred.

#### Built-in messages received in Kamps

- [`tea.WindowSizeMsg`](../internal/tui/app.go): emitted when the terminal is resized. Kamps updates its width, height, and help width.
- [`tea.KeyMsg`](../internal/tui/app.go): emitted on every keypress. `msg.String()` gives values such as `"q"`, `"ctrl+c"`, `"enter"`, or `"alt+backspace"`; `msg.Runes` carries printable input.

#### Custom application messages

- [`loadedMsg`](../internal/tui/app.go): carries courses, assignments, and materials from background loading.
- [`errMsg`](../internal/tui/app.go): wraps background errors so they can be rendered without crashing.
- [`clipboardMsg`](../internal/tui/commit.go): reports how many lines were copied and whether the OSC 52 write failed.
- [`spinner.TickMsg`](../internal/tui/app.go): drives the sync spinner and toast lifetime.

### Commands (`tea.Cmd`)

A command is a function with the signature `type Cmd func() Msg`.

> [!IMPORTANT]
> **Why do commands exist?**
> `Update` must run synchronously and quickly. If `Update` performed slow network requests or clipboard writes directly, the UI would freeze and drop keystrokes. Instead, `Update` returns a `Cmd`; Bubble Tea runs it in a separate goroutine and sends the resulting `Msg` back into `Update`.

#### Concrete commands in Kamps

1. **Background loading:** [`loadCmd()`](../internal/tui/app.go) fetches data without blocking the UI:

   ```go
   func loadCmd(src source.Source) tea.Cmd {
       return func() tea.Msg {
           snapshot, err := src.Load(context.Background())
           if err != nil {
               return errMsg{err}
           }
           return loadedMsg{snapshot}
       }
   }
   ```

   The command is asynchronous from Bubble Tea's point of view, but `Load` runs synchronously inside it. A network source should honor the context and do its fetching there.

2. **OSC 52 terminal clipboard:** [`clipboardCmd()`](../internal/tui/commit.go) copies Markdown task lines through standard ANSI OSC 52 escape sequences:

   ```go
   func clipboardCmd(text string) tea.Cmd {
       return func() tea.Msg {
           encoded := base64.StdEncoding.EncodeToString([]byte(text))
           _, err := os.Stdout.WriteString("\x1b]52;c;" + encoded + "\a")
           return clipboardMsg{lines: len(strings.Split(text, "\n")), err: err}
       }
   }
   ```

3. **Termination:** [`tea.Quit`](../internal/tui/app.go) restores the terminal and exits `Program.Run()`.

Not every side effect needs a command. `saveState()` writes a small local JSON file synchronously inside the triage handler; doing that in `Update` keeps the code simple and avoids a race where the next keypress reads stale state. A network or large-file write should use a command instead.

---

## 3. Bubbles: Reusable Component Library

The [`github.com/charmbracelet/bubbles`](https://github.com/charmbracelet/bubbles) package provides pre-built, composable widgets that follow TEA. Kamps uses three specific components:

### 1. Animated spinner: [`spinner.Model`](../internal/tui/app.go)

- Created in [`New()`](../internal/tui/app.go) with dot animation and the accent color.
- Advanced on `spinner.TickMsg`.
- Rendered by the sync status path. With `--no-anim`, the animation completes in one tick.

### 2. Declarative keybindings: [`key.Binding`](../internal/tui/keys.go)

- Defined in [`DefaultKeys()`](../internal/tui/keys.go).
- `KeyMap` is the source of truth for the `?` overlay. Footer hints are built separately by `hints`/`taskFooterHints`, because the footer has to prioritize and drop items at narrow widths.

```go
Track:    key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "track")),
Search:   key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
Commit:   key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "commit tab")),
```

### 3. Help overlay: [`help.Model`](../internal/tui/app.go)

- When the user presses `?`, [`App.helpLines()`](../internal/tui/chrome.go) renders `m.help.FullHelpView(m.keys.FullHelp())` plus a state legend and a note about mouse selection.

---

## 4. Search as a Small Input State Machine

Search is deliberately hand-rolled rather than a text-input widget. [`search.go`](../internal/tui/search.go) owns:

- `searching` — whether keys are being captured;
- `search` — the query string.

While `searching` is true, [`handleSearchKey`](../internal/tui/search.go) owns every key. This is a common TUI pattern: a mode temporarily takes over the global keymap.

Behavior worth knowing:

- Printable runes and space append to the query.
- `backspace`/`ctrl+h` deletes one rune; `ctrl+w` deletes a word.
- `ctrl+u` clears the query.
- `enter` leaves edit mode but keeps the query, so the result set stays narrowed while arrows move.
- `esc` leaves edit mode and clears the query.
- `tab` switches tabs; `switchTab` clears the search.
- The query cursor is always at the end.

The tree is filtered live through shared predicates:

```go
func (m *App) searchMatches(fields ...string) bool {
    query := strings.ToLower(strings.TrimSpace(m.search))
    if query == "" { return true }
    for _, field := range fields {
        if strings.Contains(strings.ToLower(field), query) { return true }
    }
    return false
}
```

This predicate composes with the active state filter because `visible()` and `visibleMaterials()` require **both** matches. Keeping filtering in model methods, not in `View`, means the tree, counts, and cursor all see the same set.

---

## 5. Lip Gloss: Terminal Styling & Layout

[**Lip Gloss**](https://github.com/charmbracelet/lipgloss) is Charm's CSS-like library for terminal styling.

### Adaptive colors (light & dark terminals)

Terminals can have light or dark backgrounds. Hardcoding `#FFFFFF` or `#000000` makes text unreadable in half of user environments.

Kamps defines semantic tokens as a struct of [`lipgloss.AdaptiveColor`](../internal/tui/theme.go) values:

```go
type themeColors struct {
    Bg, Surface, Marked, Overlay, Border lipgloss.AdaptiveColor
    Bright, Primary, Dim, Subtle, TreeLine lipgloss.AdaptiveColor
    Accent, Accent2 lipgloss.AdaptiveColor
    New, Tracked, Done, Ignored, Missed lipgloss.AdaptiveColor
}
```

The tree connector color is derived from the current foreground and background with `quietHex`, so guides stay quiet instead of competing with content. Reusable styles such as `stRow`, `stFaint`, `stOverdue`, and `stFlagT` are built from those tokens.

### Full-width row highlights (the “band” pattern)

Styling only the characters of a selected line produces a jagged highlight that ends after the last word:

```
[Bad]   ▌ Structured Data Assignment (MIK1624302)
[Good]  ▌ Structured Data Assignment (MIK1624302)                     new · 12d
```

[`rowHighlight`](../internal/tui/row.go) separates two decorations:

- **cursor:** the row the keyboard is on — strong `Surface` band plus `▌` gutter;
- **mark:** the row is checked — softer `Marked` band plus `▪`.

Every cell, gap, and trailing blank must go through the same band style; otherwise plain spaces punch uncolored holes in the highlight. That is why row code calls `h.style(...)`, `h.gap()`, and `h.blanks(...)` rather than concatenating unstyled strings.

### Safe string truncation

Naive Go slicing (`str[:30]`) breaks when a string contains:

1. multi-byte UTF-8 runes (`—`, `▾`, `○`);
2. ANSI escape sequences (`\x1b[38;2;...m`), which can be cut in half.

Kamps uses:

- [`lipgloss.Width`](../internal/tui/text.go) to measure visible columns;
- [`ansi.Truncate`](../internal/tui/text.go) via `truncate` to cut to width with `…` while preserving escapes;
- [`ansiTruncate`](../internal/tui/text.go) via `lipgloss.MaxWidth` for styled content;
- [`fitLine`](../internal/tui/text.go) to clamp every emitted line to exactly the requested width.

Normal text is truncated with `…`; tree rows reserve a two-column right inset so metadata never touches the terminal edge or the split divider.

### Responsive split-pane layout

[`detail.go`](../internal/tui/detail.go) implements the detail pane:

- It is opt-in: `enter` opens or closes it.
- [`splitWidths`](../internal/tui/detail.go) gives 62% to the tree and the rest to detail, enforcing a 24-column minimum for detail.
- [`splitTreeRows`](../internal/tui/detail.go) renders the left line, a `│` divider, and the right line side by side.
- The override is global, not per tab.

### Borderless shell

The main shell is deliberately not boxed. [`frame.go`](../internal/tui/frame.go) draws the tab bar, title line, full-width rules, body, and footer. This keeps every column available for content and avoids the cramped feeling of a split pane between two vertical borders.

---

## 6. Putting It Together

A rendering pass looks like this:

1. Bubble Tea calls `View`.
2. `View` asks `tabContent` for the active tab's title, hints, and rows.
3. `tabContent` flattens the tree (`taskNodes`/`matNodes`), chooses a window around the cursor, and renders rows.
4. If the detail pane is visible, `splitTreeRows` composes two columns.
5. `bodyLines` clamps the body to the available height.
6. `helpBar` appends hints and the queue/toast status.
7. Bubble Tea writes the resulting string to the alternate screen.

Because every step is a pure formatting function over `App`, the screen can be tested without a terminal. The tests in [`internal/tui`](../internal/tui) call `View` directly and assert widths, row contents, and clipping.
