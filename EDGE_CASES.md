# Edge Cases & Reliability Checklist (EDGE_CASES.md)

This document catalogs all critical edge cases for Kamps, the required behavior, implementation rules, and verification tests. Any agent or developer working on downloading, storage, parsing, or state management should verify their implementation against this checklist.

---

## 1. Filesystem & Local Storage Edge Cases

- [ ] **1.1. Path Traversal & Dangerous Filenames**
  - **Issue**: LMS filenames or week titles containing slashes (`/`), backslashes (`\`), colons (`:`), null bytes, or traversal sequences (`../../etc/passwd`).
  - **Requirement**: All filename and section components must be strictly sanitized using `filepath.Base` after stripping illegal characters (`/`, `\`, `:`, `*`, `?`, `"`, `<`, `>`, `|`, control codes). Replace illegal characters with `_`. Never allow path traversal.
  - **Test**: `TestSanitizeFilename` with traversal strings, control characters, colons, and Windows-reserved names.

- [ ] **1.2. Path & Filename Length Limits**
  - **Issue**: Linux filesystem limit of 255 bytes per filename component, or long paths exceeding 4096 bytes.
  - **Requirement**: If a sanitized filename exceeds 200 bytes, truncate the base name while strictly preserving the file extension (e.g. `Long_Name...[truncated]...pdf`).
  - **Test**: `TestSanitizeFilename_LengthLimit` with 300+ character filenames with various extensions.

- [ ] **1.3. Pre-existing Identical Files (Duplicate / Re-download)**
  - **Issue**: User hits `d` on a file that already exists on disk in the target location.
  - **Requirement**: Check `os.Stat(targetPath)` before making network calls. If the file exists and its byte size matches the remote `Content-Length`, skip the network transfer completely, mark as downloaded, and notify the user: *"Already on disk; skipped download"*.
  - **Test**: `TestResolveDestination_ExistingMatches`.

- [ ] **1.4. Modified Local Files ("No-Clobber" Invariant)**
  - **Issue**: The local file already exists, but the user annotated it (or the lecturer uploaded a revised version), so file sizes/hashes differ.
  - **Requirement**: Kamps must **never** overwrite or delete an existing file. If sizes differ, save the incoming file with an incremented version suffix: `<stem>_v2.<ext>` (or `<stem>_v3.<ext>`), leaving the student's file untouched.
  - **Test**: `TestResolveDestination_ExistingModifiedVersionSuffix`.

- [ ] **1.5. Insufficient Disk Space (ENOSPC)**
  - **Issue**: Downloading large slide decks or VMs on a nearly full disk.
  - **Requirement**: Check available filesystem space before downloading. If disk space runs out during the transfer (`syscall.ENOSPC`), immediately abort, delete the partial `.part` file, and show toast: *"Disk full: download cancelled"*.
  - **Test**: `TestDownload_ENOSPC_CleansPartFile`.

- [ ] **1.6. Unwriteable or Non-Existent `library_dir`**
  - **Issue**: `library_dir` points to an unmounted drive, read-only path, or broken symlink.
  - **Requirement**: Validate path on startup or before first download; attempt directory creation with `0755`. If write fails with `os.ErrPermission` or `syscall.EROFS`, halt with a user-friendly toast rather than a silent failure or crash.
  - **Test**: `TestEnsureLibraryDir_ReadOnlyPath`.

---

## 2. Network & Download Reliability

- [ ] **2.1. Interrupted Downloads (Connection Loss / Cancellation)**
  - **Issue**: Network disconnects, Wi-Fi drops, server times out, or user quits mid-download.
  - **Requirement**: Downloads must write to a temporary file: `<targetPath>.<pid>.<timestamp>.part`. On error or context cancellation, the `.part` file must be removed immediately. The material must **never** be marked as `downloaded = true` in state unless the file was completely written, flushed (`Sync()`), and renamed into place.
  - **Test**: `TestDownload_InterruptedCleansUpPartFile`.

- [ ] **2.2. Content-Length Verification & Zero-Byte Files**
  - **Issue**: Network stream terminates prematurely with a silent truncated response, or remote file is completely empty (0 bytes).
  - **Requirement**: If `Content-Length` header is provided, the written byte count must match before committing. If the remote stream is 0 bytes, reject and delete the `.part` file with notice: *"Remote file is empty (0 bytes)"*.
  - **Test**: `TestDownload_TruncatedStreamRejected`.

- [ ] **2.3. Rate Limiting & Concurrency Throttling**
  - **Issue**: Rapid concurrent downloads overwhelm university servers or trigger campus IP bans.
  - **Requirement**: Restrict concurrent download workers (default $\le 4$) and enforce request pacing gap between connection starts. Honor `Retry-After` headers and HTTP 429/503 responses with exponential backoff.
  - **Test**: `TestDownloader_ConcurrencyAndThrottling`.

- [ ] **2.4. Memory Safety for Large Files**
  - **Issue**: Reading 1GB+ files entirely into memory buffers causes out-of-memory crashes.
  - **Requirement**: Always use buffered chunked streaming (`io.CopyBuffer` with a fixed $\le 32$ KB buffer) directly from the HTTP response body to the `.part` file.
  - **Test**: Memory allocation checks on simulated large stream transfers.

---

## 3. LMS & Content Quirks

- [ ] **3.1. Silent HTML Login Redirects on Expired Session**
  - **Issue**: When a session expires, Moodle returns HTTP 200 OK with an HTML login page rather than HTTP 401. If saved directly, the student gets an HTML page named `document.pdf`.
  - **Requirement**: Check `Content-Type` header (reject `text/html` when downloading resources) and peek the first 512 bytes for `<!DOCTYPE html` or `<html`. If detected, abort download, delete `.part`, and emit: *"Session expired — please run kamps login"*.
  - **Test**: `TestDownload_DetectsHTMLLoginPage`.

- [ ] **3.2. External Cloud Links (Google Drive, Dropbox, YouTube)**
  - **Issue**: Coursework items typed as `modtype_url` or pointing to Google Drive/Dropbox folders cannot be downloaded directly via Moodle pluginfile stream.
  - **Requirement**: Classify external URLs as `Kind = "url"` (icon `↗`). Pressing `d` explains: *"External cloud link; press 'o' to open in browser"*. Pressing `o` or `enter` launches the URL with `xdg-open` / default browser.
  - **Test**: `TestMaterialKind_ExternalCloudLinks`.

- [ ] **3.3. Remote File Deleted on LMS (HTTP 404 / 410)**
  - **Issue**: Assignment or material was tracked locally, but instructor removed it from Moodle before user downloaded it.
  - **Requirement**: Catch 404/410 cleanly. Do **not** remove user's local triage records in `state.json`. Mark node in UI as unavailable with detail notice: *"File no longer available on LMS (HTTP 404)"*.
  - **Test**: `TestDownload_RemoteFileNotFound`.

- [ ] **3.4. Moodle Folder Modules (`modtype_folder`)**
  - **Issue**: Instructor uploaded a multi-file bundle or folder.
  - **Requirement**: Download either individual sub-files into `<Course>/<Section>/<FolderName>/` or stream the folder ZIP and unpack cleanly without path traversal.
  - **Test**: `TestFolderModule_UnpackSafe`.

- [ ] **3.5. Campus Maintenance Mode (HTTP 503)**
  - **Issue**: University IT brings Moodle down for maintenance during exam periods.
  - **Requirement**: Recognize HTTP 503 maintenance pages; allow user to continue reading local tasks, cached coursework, and downloaded materials completely offline without crashing.
  - **Test**: `TestLMS_MaintenanceModeOfflineGrace`.

---

## 4. State & Concurrency Edge Cases

- [ ] **4.1. Rapid Keystroke & In-Flight Download Deduplication**
  - **Issue**: User mashes `d` repeatedly or triggers a course download while a child file is already in flight.
  - **Requirement**: Maintain thread-safe `activeDownloads` registry (`sync.Map`). If an ID is already in flight, ignore subsequent triggers.
  - **Test**: `TestDownloadCoordinator_Deduplication`.

- [ ] **4.2. Multiple `kamps` Processes Running Concurrently**
  - **Issue**: User opens `kamps` in two tmux panes or terminal windows simultaneously.
  - **Requirement**:
    1. Unique PID/nano-timestamp in temp `.part` file names (`<stem>.<pid>.<time>.part`) so instances don't collide during download.
    2. Atomic file replace (`os.Rename`) on the final step.
    3. Advisory file locking (`state.lock`) during `state.json` write cycles, merging state if file modification time changed.
  - **Test**: `TestConcurrentProcesses_NoCorruptPartFiles`.

- [ ] **4.3. Terminal Resizing During Active Download**
  - **Issue**: Window resized below 80x24 while background goroutine emits progress ticks.
  - **Requirement**: Clamp all viewports; if terminal $< 80\times 24$, render minimal warning banner without panic. State updates from downloads must not alter search focus or jump tree cursors.
  - **Test**: `TestTUI_ResizeDuringDownloadMsg`.

- [ ] **4.4. Simultaneous Search & Download State Updates**
  - **Issue**: User is actively typing `/query` while background download finishes and changes file status from `new` to `downloaded`.
  - **Requirement**: Flattened tree rows and cursor clamping must re-evaluate deterministically without resetting query buffer or causing index out of range.
  - **Test**: `TestTUI_SearchStableUnderDownloadCompletion`.
