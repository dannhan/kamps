# Workspace mapping (`course.json`)

By default downloads go to:

```text
<library_dir>/<course>/<section>/<folder>/<file>
```

`library_dir` defaults to `~/Documents/kamps`. If a course already lives in a
repo, add a `course.json` mapping and downloads go there instead.

## Setup

Tell kamps where to look for mappings:

```toml
# ~/.config/kamps/config.toml
workspace_roots = ["~/projects/university"]
```

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
~/projects/university/basdat/Week 1/Slides/week1.pdf
```

No extra course directory is added inside the workspace. Every path component
is sanitized, and downloads are rejected if they would escape the workspace.
