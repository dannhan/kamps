# Kamps Feature Specifications & Project Plan (PLAN.md)

This document provides a comprehensive blueprint of **Kamps**: the current state of implementation, detailed technical mechanics of upcoming and existing features, data storage, network policies, triage handling, and authentication design options.

---

## 1. Executive Summary & Vision

**Kamps** is a terminal application (TUI) for university coursework aggregation and triage. It interfaces with Learning Management Systems (specifically Diponegoro University's Moodle 4.5 LMS, **Kulon2**) and local workspace directories.

Instead of navigating browser tabs and ad-hoc spreadsheets, students use `kamps` to:
1. Aggregate active courses, assignments, deadlines, and learning materials into a single terminal interface.
2. Apply an intentional, sticky **triage verdict** (`undecided`, `tracked`, `missed`, `ignored`) to every piece of coursework.
3. Keep course files locally synchronized in a structured directory under a strict non-destructive policy.
4. Export action items to external task management systems (Markdown to-do lists, Todoist, Obsidian, Taskwarrior) via a review-and-commit export buffer and terminal clipboard (OSC 52).

---

## 2. Feature Status Matrix

| Feature | Status | Package / Module | Description |
|---|---|---|---|
| **Demo Mode** | Implemented | `internal/source/demo.go` | In-memory dataset for offline UI exploration (`./kamps --demo`). |
| **Interactive TUI Shell** | Implemented | `internal/tui` | 4-tab Bubble Tea app (Tasks, Materials, Sync, Commit) with responsive split-pane detail view. |
| **Plain Mode Output** | Implemented | `internal/plain` | Non-TTY stdout snapshot (for `kamps \| less` or CI). |
| **Moodle Cookie Auth** | Implemented | `cmd/kamps/auth.go`, `internal/session` | Manual cookie capture (`kamps login`/`logout`), verification, and private storage (`0600`). |
| **LMS Course List Ingestion** | Implemented | `internal/kulon2/courses.go` | Fetches enrolled in-progress courses via Moodle timeline AJAX service. |
| **LMS Course Page Parsing** | Implemented | `internal/kulon2/pages.go` | Scrapes course HTML for weekly sections, assignments, and resource files. |
| **LMS Assignment Page Parsing** | Implemented | `internal/kulon2/pages.go` | Scrapes assignment deadline ISOs, submission state, and attached files. |
| **Atomic State Persistence** | Implemented | `internal/store` | Persists user triage decisions and download flags atomically to `state.json`. |
| **Live Search & Filters** | Implemented | `internal/tui/search.go`, `filters.go` | Case-insensitive multi-field search composing with status filters. |
| **Commit Queue & OSC 52** | Implemented | `internal/tui/commit.go` | Stages session decisions as Markdown tasks, copies to clipboard via OSC 52. |
| **Material File Downloads** | Implemented | `internal/library`, `internal/kulon2/download.go` | Streams `mod/resource` files to disk; skips identical files, versions changed ones, and surfaces failures. Folder modules and assignment attachments are still planned. |
| **Local Library Management** | Partial | `internal/config`, `internal/library` | `library_dir` scaffolding, `~` expansion, and per-file skip/version rules are done; scanning the library on startup is still planned. |
| **Workspace Course Mapping** | *Planned* | `internal/config`, `internal/source` | Optional `course.json` in local project repos to bind workspace folders to LMS course IDs. |
| **Direct File Commit Writers** | *Planned* | `internal/tui/commit.go` | Direct file appending (e.g. `tasks.md`, `todo.txt`) beyond OSC 52 clipboard. |
| **Automated Triage Sync** | *Planned* | `internal/kulon2/source.go` | Automatically triaging assignments marked "Submitted for grading" on the LMS as `tracked`. |
| **Automatic Cookie Ingestion** | *Planned* | `cmd/kamps/auth.go` | Extracting active MoodleSession cookies directly from local browser storage. |

---

## 3. Deep-Dive Feature Specifications

### 3.1 Material Downloads: How & Where Files Are Downloaded

#### Where Files Are Downloaded
* **Base Directory**: Configured via `library_dir` in `$XDG_CONFIG_HOME/kamps/config.toml` (defaulting to `~/Documents/kamps`).
* **Path Hierarchy**:
  ```text
  ~/Documents/kamps/
  └── <Course Name or Short Code>/
      └── <Section or Week Name>/
          └── <Filename>
  ```
  *Example*: `~/Documents/kamps/Metode-Numerik-A/Week-01/Praktikum-1-Panduan.pdf`
* **Safe, Non-Destructive ("No-Delete") Invariant**:
  * Local files are **never deleted or overwritten** if an instructor removes or replaces a file on Moodle.
  * If a file with identical name exists:
    * If size matches the remote content-length: file is treated as already downloaded and skipped.
    * If size differs: downloaded with a timestamp or sequential suffix (e.g., `Slides_v2.pdf`) rather than clobbering existing student notes.

#### How Files Are Downloaded
* **Protocol & URLs**:
  1. The course page provides links to `/mod/resource/view.php?id=<cmid>`.
  2. Following this with session cookie yields a redirect (`303`/`302`) to Moodle's `pluginfile.php/.../<filename>`.
  3. The downloader issues a `GET` request directly to `pluginfile.php`.
* **Politeness & Rate Limiting**:
  * Shared token bucket or serialized slot allocator (`waitForRequestSlot`) enforcing at least **100 ms – 1000 ms gap** between connection requests.
  * Concurrency capped at **`maxConcurrent = 4`** workers to avoid overloading university servers.
  * Respects `Retry-After` headers and handles HTTP `429 Too Many Requests` or `5xx` server faults with exponential backoff.
* **Progress & Persistence**:
  * Streaming body reads update a Bubble Tea progress model / progress bar in the TUI.
  * Upon complete write and file flush, the material ID is recorded in `state.json` under `downloaded[materialID] = true`.

---

### 3.2 Triage Lifecycle: What Happens with Triaged Coursework

#### Triage States (`model.Triage`)
Every assignment starts as `undecided`. Users press hotkeys in the Tasks tab:
1. **`t` → Tracked (`◐`)**: The assignment is handled. Either submitted, assigned to a teammate, or tracked in an external todo list.
2. **`m` → Missed (`×`)**: The deadline has lapsed without submission. It is closed and dropped from urgent views.
3. **`i` → Ignored (`◆`)**: Irrelevant items, syllabus templates, or optional bonus assignments.
4. **`u` → Undecided (`○`)**: Resets the item back to the default inbox state.

#### Local Persistence Mechanics
* Stored in `$XDG_STATE_HOME/kamps/state.json` (`~/.local/state/kamps/state.json`).
* **Immediate Persistence**: When `t`, `m`, `i`, or `u` is pressed, the change is written to disk immediately (not deferred to quit).
* **Sparse Keys**: Setting an item to `Undecided` deletes its key from `state.json`, preventing unbounded file bloat.
* **Atomic File Replacement**:
  ```text
  State in memory
       │
       ▼ Marshal JSON (0600)
  .state-XXXXXXXX.tmp in ~/.local/state/kamps/
       │
       ▼ Sync() & Flush
  os.Rename over state.json
  ```
  Guarantees that a crash, power outage, or SIGINT never results in a corrupted state file.

#### The Commit Queue (`App.pending`) & Exporters
* Decisions made during the current execution session are queued in `App.pending`.
* **Session-Only Buffer**: The queue intentionally empties when `kamps` exits. Decisions themselves remain saved in `state.json`; the Commit tab represents the *export batch*.
* **Export Preview**:
  Assignments are rendered as GitHub-flavored Markdown task items:
  ```markdown
  - [ ] [metnum] Submission Praktikum 1 (K155170) due:2026-08-31
  - [ ] [ppl] Sprint 2 Retrospective (K165483) due:2026-09-12
  ```
* **Clipboard (OSC 52)**: Pressing `y` or `enter` on the Commit tab wraps the lines in ANSI OSC 52 clipboard sequences (`\x1b]52;c;...\x07`), transmitting them straight to the host OS clipboard even over SSH and tmux.
* **Future Commit Writers**:
  * Append-only writer to a local `tasks.md` or `todo.txt`.
  * Integration hooks for Todoist API / Obsidian vault.

---

### 3.3 Authentication & Session Verification

#### When Does Login Verification Happen?
* **Startup Session Check**:
  * On launch, `kamps` reads `$XDG_STATE_HOME/kamps/session.json`.
  * If `session.json` is missing or belongs to a different `site_url`, startup halts with an informative error prompting `kamps login`.
* **Verification Frequency (Cached SessKey Optimization)**:
  * Verification **does NOT hit `/my/` on every startup** if an authenticated `sesskey` is already cached in `session.json`.
  * The loader immediately attempts `core_course_get_enrolled_courses_by_timeline_classification` using the saved `sesskey`.
  * **Only if** `sesskey` is empty or rejected by Moodle (`ErrInvalidSesskey`) does `kamps` issue a `GET /my/` verification request to refresh the session and extract a new `sesskey`.
  * If Moodle redirects to `/login/index.php` or `/auth/oidc/`, `ErrSessionExpired` is triggered, prompting the user to re-authenticate.
* **Cookie Expiration**:
  * Kulon2's `MoodleSession` cookie has an 8-hour lifetime (`sessiontimeout: 28800` s). Once expired on the server, subsequent launches will require a fresh cookie.

#### Options to Make Authentication More Automatic
Since Kulon2 uses enterprise Microsoft Azure AD SSO and disables local passwords and token endpoints, fully headless username/password login is not directly available via basic HTTP requests. However, several viable automation paths exist:

| Strategy | Feasibility | Security / UX | Mechanism |
|---|---|---|---|
| **1. Local Browser Cookie Extraction (Recommended)** | **High** | Excellent. Zero friction if user already browses Kulon2. | Read cookie SQLite databases directly from user profiles (Chrome, Firefox, Brave, Edge) under `~/.config/...` and `~/.mozilla/...`. Automatically extracts `MoodleSession` with no manual DevTools copy-paste. |
| **2. Local Loopback Companion Extension** | **Medium** | High security. Requires 1-time extension install. | Minimal browser userscript or extension that listens for Moodle auth and POSTs the active session to `http://localhost:<port>/session` or writes directly to `session.json`. |
| **3. Headless Browser Automation (Playwright/CDP)** | **Medium-Low** | Heavyweight. Brittle with SSO MFA/2FA. | Spawns a background browser to handle SSO. If university requires 2FA / Microsoft Authenticator app, headless automation gets stuck or requires interactive webview. |
| **4. Mobile App Token API** | **Blocked** | Infeasible on this LMS. | Investigated in `docs/kulon2.md`: `/login/token.php` requires a local Moodle password; SSO accounts cannot set local passwords because site SMTP is disabled. |

---

## 4. Implementation Milestones

### Phase 1: Live Scraping & Coursework Parsing *(Current Focus)*
- [x] In-progress course listing via Moodle AJAX endpoint.
- [x] Course-page HTML parsing into sections, resources, and assignments.
- [x] Assignment deadline parsing and submission detection.
- [x] Graceful partial error handling and session auto-recovery.
- [ ] Parse embedded attachment URLs from assignment detail descriptions.

### Phase 2: Downloader Engine & Local File Library
- [x] Implement `internal/kulon2/download.go` for streaming `pluginfile.php` downloads.
- [x] Create directory builder honoring `library_dir` (`<course>/<section>/<file>`).
- [x] Add safe-skip / non-destructive versioning if destination files already exist.
- [x] Connect `d` hotkey in Materials tab to background download tasks with progress ticker.
- [ ] Download assignment attachments and folder modules.
- [ ] Resume interrupted downloads instead of restarting them.

### Phase 3: Authentication Enhancements
- [ ] Add `kamps login --browser` to automatically discover and extract `MoodleSession` from local Chromium/Firefox SQLite cookie jars on Linux.
- [ ] Add session lifetime tracking in `session.json` to warn users when their 8-hour session is nearing expiration.

### Phase 4: Integrations & Commit Exporters
- [ ] Support `--commit-file <path>` to directly append completed triage items to a local Markdown file (e.g., `~/notes/tasks.md`).
- [ ] Introduce optional workspace `course.json` discovery to map LMS course IDs to local git repository folders.
