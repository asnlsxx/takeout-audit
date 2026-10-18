# Delivery plan

The objective is a small useful addition to an existing workflow: explain every
source entry in a Google Takeout migration, without inferring that originals can
be deleted. Repository size, generated code volume and stars are not success
criteria. Existing hash-check commands remain the right choice for simple cases.

1. Review correctness: preserve every occurrence; reject repeated/overlapping
   roots before expensive reads; separate content-level and occurrence-level
   counts; fail closed on unreadable data and malformed responses.
2. Review usability: ship a dedicated read-only binary, a short English README,
   Chinese quick start, stable report fields and reproducible examples.
3. Verify: unit and CLI tests, malformed-input fuzzing, a bounded synthetic
   benchmark, generated-code freshness, dependency scan and a real disposable
   Immich 3.2.0 lifecycle test in a remote CI runner.
4. Prepare public delivery: preserve MIT attribution and upstream history,
   exclude credentials/private inputs/local logs, add multi-platform CI and an
   explicitly marked prerelease. Do not claim user adoption or production
   reliability before trials occur.
5. Publish to the authenticated user's GitHub account, verify remote contents
   and CI results, then attach downloadable archives with checksums. GitHub
   authentication is required; a local archive is not equivalent to publication.
6. Keep local storage small: do not install a local container stack; remove
   regenerable build caches, duplicate archives and temporary tooling after
   verification. Preserve source, concise evidence, and the useful Windows build.

Stop expanding scope when the specified workflow is reliable. Do not add upload,
deletion, similarity search, album repair, a hosted service or a generic GUI.
Unmet external gates must be reported explicitly, not waived to call work done.
