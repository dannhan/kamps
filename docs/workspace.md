# Workspace mapping (`course.json`)

By default downloads go to:

```text
<library_dir>/<course>/<week>/<folder>/<file>
```

`library_dir` defaults to `~/Documents/kamps`. If a course already lives in a
repo, add a `course.json` mapping and downloads go there instead.

## Setup

Tell kamps where to look for mappings:

```toml
# ~/.config/kamps/config.toml
workspace_roots = ["~/projects/university"]
```

Existing course metadata can also identify a course without a separate mapping.
For example, `basdat/course.json` may contain `"lms_course_id": 17964`,
`"code": "MIK1624303"`, and `"source": "lms"`. Kamps uses `basdat/materi`
when that directory exists, or `basdat/materials` otherwise. It creates missing
download directories only when downloading. Existing padded week folders such as
`week-05` are reused. New week directories use two digits, such as `week-01`.
An existing `week-1` directory is reused for that week. Course metadata and other course folders are not changed.
Folders supplied by the LMS remain inside their weekly materials folder; Kamps
does not infer a practicum or assignment destination from a file name.

Create one `course.json` per repo (checked in, gitignored, or local):

```json
{
  "version": 1,
  "courses": [
    {
      "lms_id": 17964,
      "shortname": "MIK1624303",
      "path": "."
    }
  ]
}
```

- `lms_id` is the numeric Moodle course id; it wins when present.
- `shortname` (or `code`) is matched case-insensitively as a fallback.
- `path` is relative to `course.json` or absolute. `.` means the repo root;
  `./materials` keeps generated files in a subfolder.
- A course without a mapping keeps using `library_dir`.

Kamps only reads `course.json`; it never creates or edits it.

## Template helper

To print a starter file without writing it:

```sh
kamps workspace init --lms-id 17964 --shortname MIK1624303 --path .
```

Redirect it yourself if you want the file:

```sh
kamps workspace init --lms-id 17964 --shortname MIK1624303 --path . > course.json
```

## Discovery

`workspace_roots` entries are walked up to three directories deep. `.git`,
`node_modules`, and hidden directories are skipped. Duplicate or missing paths
produce a notice in the UI instead of a crash.

## Download layout

For a mapped course with `path: "."`:

```text
~/projects/university/basdat/week-01/Slides/week1.pdf
```

No extra course directory is added inside the workspace. Every path component
is sanitized, and downloads are rejected if they would escape the workspace.
