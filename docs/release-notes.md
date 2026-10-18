# Takeout Audit 0.1 alpha

This prerelease adds a dedicated read-only command for reconciling Google
Takeout ZIPs or extracted directories with an Immich account.

* Streams files without full extraction and retains each source occurrence.
* Explains repeated album copies using SHA1 protocol queries and local SHA256 grouping.
* Writes a JSONL ledger, filterable CSV and source/content-level summary.
* Distinguishes catalog matches, trash, unmatched content, unknown responses,
  read failures, unclassified files and unchecked JSON metadata.
* Rejects duplicate/overlapping roots and never treats incomplete API results as success.
* Supports offline inventory and stable Immich 3.2.x contract checks.

Known limits: no metadata/album reconciliation, external-library content
verification, resumable runs or proof of complete Google export. A catalog match
does not verify current server file bytes. No upload, deletion or self-update
commands are exposed by the dedicated binary.

Verification: 287 local passing test events (including subtests), native
Windows/Linux/macOS CI, bounded malformed-ZIP fuzzing, no govulncheck findings,
and a disposable real Immich 3.2.0 test: one matched, one trashed and one absent
content group, one extra source copy, restricted query key, unchanged asset
state and fixture cleanup. [Initial successful lifecycle and platform run](https://github.com/asnlsxx/takeout-audit/actions/runs/39115445636).

This is a controlled-trial alpha, not a production backup guarantee. Real-user
large exports and other server patches still need trials. Publication requires
successful CI on the release's exact commit. Artifacts include MIT attribution,
English/Chinese instructions and SHA256 checksums.
