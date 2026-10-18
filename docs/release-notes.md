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

This is a controlled-trial alpha, not a production backup guarantee. Consult
the repository validation record and exact CI run for real-server and platform
test status. Artifacts include MIT attribution and SHA256 checksums.
