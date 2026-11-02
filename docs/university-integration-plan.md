# University workspace integration plan

1. Capture the existing folder and CLI behavior using an isolated fixture based on the university course metadata. Verify that the fixture exposes the current mapping and path failures.
2. Resolve each LMS course to its existing course folder using course IDs and codes. Support the existing per-course metadata, keep explicit workspace mappings, and report ambiguity rather than guessing.
3. Put weekly downloads under the existing `materi` or `materials` directory. Preserve existing files and week naming, and retain LMS folder names without moving course work.
4. Expose resolved course and material destinations in structured CLI output. Retain task and sync output and its existing JSON contract.
5. Exercise the executable with isolated config, state, and a copy of the course folder layout. Check course mappings, task and sync JSON, destination previews, repeat downloads and collisions using local fixtures. Run targeted tests and the demo verification journey.

Do not start quizzes, submit assignments, modify the real university folder, or connect to the user's LMS session during verification. Existing quiz parsing remains deferred.
