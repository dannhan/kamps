# Kulon2 / Moodle — verified platform notes

Working notes on the LMS that `kamps` will talk to. Everything here was
verified by hand against the live site (first recon 2026-10-17) unless it says
**unverified**. Keep it factual: no guesses presented as facts.

## The site

- Base URL: `https://kulon2.undip.ac.id/`
- Stock Moodle 4.5, theme `moove`.
- Auth: UNDIP SSO is **Microsoft Azure AD** (`auth_oidc` plugin,
  `/auth/oidc/?source=loginpage`). Accounts are `@students.undip.ac.id`.

## Auth: use a session cookie

- `login/token.php?service=moodle_mobile_app` exists and replies
  `{"error":"Invalid login...","errorcode":"invalidlogin"}`, so the mobile web
  service is enabled — but it needs a **local password**.
- SSO-provisioned accounts have no local password, and site SMTP is broken
  (`/login/forgot_password.php` → "Error sending password change confirmation
  email"), so a password can never be set. Token auth is unusable here.
- Therefore the only viable route is the **`MoodleSession` cookie**. Its
  lifetime is `sessiontimeout: 28800` seconds (8 h).
- Verified: an authenticated `GET /my/` returns the user id and a `sesskey`
  (`"sesskey":"..."`) which later AJAX calls need.

`kamps` must never automate the SSO login. The user logs in with a browser and
pastes only the cookie value. `kamps login` verifies the cookie against `/my/`
and stores it with `0600` in its own state directory; `kamps logout` removes
it. Session expiry is reported crisply (a login redirect or missing authenticated
page data means "session expired — log in again"). The live source currently
loads the enrolled course list; course pages and their coursework are not yet parsed.

## Data routes that work with a session cookie

1. **Course list (JSON)** — AJAX (ported in `internal/kulon2`; tested with local service-response fixtures):

   ```text
   POST /lib/ajax/service.php?sesskey=<sesskey>&info=core_course_get_enrolled_courses_by_timeline_classification
   body: [{"index":0,
           "methodname":"core_course_get_enrolled_courses_by_timeline_classification",
           "args":{"offset":0,"limit":0,"classification":"inprogress","sort":"fullname"}}]
   ```

   Returns `data.courses[].{id, shortname, fullname}`. 29 enrolled at recon time.

2. **Course page (HTML)** — `GET /course/view.php?id=<courseid>` (parsed by the current live source):

   - sections via `course/section.php?id=` with week labels
     (e.g. "18 September 2026 - 25 ...");
   - activities: `/mod/assign/view.php?id=` (assignments),
     `/mod/resource/view.php?id=` (files), plus `pluginfile.php` refs;
   - activity names live in `class="instancename"`.

## Data routes that are blocked

- `core_calendar_get_calendar_events` → `servicenotavailable`
- `mod_assign_get_assignments` → `servicenotavailable`
- Action-events endpoints return empty lists even on courses with 14+
  assignments.

**Consequence:** this site cannot be driven by Web Services alone. Assignments,
due dates and materials must be parsed from course-page HTML. Use a real HTML
parser (`golang.org/x/net/html`), never regex over raw HTML.

## Semester mapping (SMT11 2026/2027 Ganjil)

| Kulon2 course | id | content seen at recon |
|---|---|---|
| Basis Data C (MIK1624303) | 17964 | 14 resources, 0 assignments |
| Struktur Data B (MIK1624302) | 17638 | 10 resources, 9 assignments |
| Proyek Perangkat Lunak A (PAIK6504) | 18515 | 3 resources, 14 assignments |
| Metode Numerik A (PAIK6304) | 18326 | 16 resources, 19 assignments, 2 quizzes |

IMK is not on Kulon2 at all (it lives on WhatsApp), so it has no LMS course.

## Implications for the client

- **Auth:** cookie only; `kamps login` and `kamps logout` are implemented. Live mode verifies the saved session, fetches in-progress courses and course pages, then fetches assignment pages to populate recognized deadlines. Assignment attachments and folder children are materialized as downloadable materials; downloads authenticate with the saved cookie.
- **Due dates:** assignments fetch `/mod/assign/view.php?id=<cmid>` and map recognized due dates into the snapshot. This endpoint/date shape is fixture-tested from the legacy recon, not yet re-verified against the live site.
- **Downloads:** `mod/resource` links resolve to `pluginfile.php` direct
  downloads — one request per file, politely spaced. Files land under
  `<library_dir>/<course>/<week>/<folder>/<filename>`; an existing file with the same
  size is skipped, and a changed file is saved with a `_v2` suffix instead of
  overwriting local notes. Concurrent writers are serialized by a hashed lock
  file under `$XDG_STATE_HOME/kamps/locks`, not beside the material.
  Attachment filenames are percent-decoded for display and disk paths.
  `mod/url` activities are external links and open in the browser with `o`.
- **Politeness:** at least 1 s between requests, at most 2 requests per page.
- **Unverified:** `GET /my/courses.php` course cards. Treat a parse failure as
  an error, never as an empty course list.
