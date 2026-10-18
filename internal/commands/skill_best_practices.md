## Best practices

### Takeout audit (experimental local extension)

```sh
# All ZIPs must come from the same export. Output is a NEW directory outside inputs.
immich-admin cw audit-takeout --inventory-only --out inventory takeout-001.zip takeout-002.zip
immich-admin --config config.staging.yaml cw audit-takeout --out audit takeout-001.zip takeout-002.zip
immich-admin server version
# Dedicated read-only executable (same workflow, no administrative commands):
takeout-audit --inventory-only --out inventory2 takeout-001.zip takeout-002.zip
```

`--inventory-only` requires no config or network and deliberately exits 2:
media has not been checked remotely. Online mode accepts stable 3.2.x only;
the API key needs `user.read` and `asset.upload`, but a transport allowlist
permits only account/version reads and POST `/assets/bulk-upload-check`.
Contract tests use a mock server; live-server validation is still pending.
No upload/delete/repair is part of this command, so no `--yes` is necessary.

Read `summary.json` first, then `ledger.jsonl` (all entries) and `media.csv`.
Source copies and unique content groups are different counts: `statuses` counts
occurrences, `group_statuses` counts distinct contents, and
`duplicate_media_copies` explains additional copies. Duplicate/overlapping input
roots are rejected before media reads. Legacy path bytes are preserved in
`path_raw_base64` if the ZIP name is not UTF-8. Exit 0 means a
complete scoped catalog comparison, 1 means differences/attention, 2 means
incomplete. `catalog_match` is not a disk-integrity or metadata guarantee;
`not_matched` is not proof of loss. External libraries are outside scope.
Never recommend deleting originals based on this report. The tool does not
know whether all Google export parts were supplied. Limits default to 500,000
entries, 64 GiB/file and 2 TiB total uncompressed bytes; hitting limits produces
an incomplete report. Do not self-update this development build until the
extension is available upstream. See `docs/takeout-audit.md` for release gates.

### Prerequisites

- The `immich-admin` binary and network access to an Immich server **v3.2.0
  or newer** (every command prints `account (server, server X.Y.Z, needs >=
  3.2.0)` to stderr and warns on older servers).
- Config via `--config FILE` (default `config.prod.yaml`);
  `IMMICH_SERVER` / `IMMICH_API_KEY` env vars override file values.
- This skill's command catalog below is generated from the CLI itself; run
  `immich-admin <group> <command> --help` for full flags and examples.

### Safety discipline

- Every command prints the account + server identity to stderr — always
  check it before acting.
- Preview destructive work with `--dry-run` first, confirm with `--yes`
  only in automation.
- Deletions go to **trash** (restorable) by default; `--force` deletes
  permanently. Tag deletion is the exception: tags have no trash and
  deleting a parent deletes its children — always dry-run first.

### ID plumbing (the core pattern)

- `--ids-only` (short `-q`) prints one UUID per line — pipe it into the
  next command or save it: `search metadata --all -q > ids.txt`.
- `--ids-file FILE` reads IDs back (one UUID per line, `#`/`//` comments
  and blank lines skipped, `-` means stdin). Positional IDs and
  `--ids-file` combine.
- Bulk commands continue on per-ID errors and exit non-zero with a
  `N of M failed` summary — inspect stderr, don't assume all-or-nothing.

### Prefer --json for complete information

Human output is lossy by design (one-line summaries plus a count). Almost
every read command also accepts `--json`, which prints the raw API
response with all fields. Whenever you need the full record — for
analysis, filtering, or feeding another tool — prefer it:

```sh
immich-admin immich-workflow list --json
immich-admin assets info <ASSET_ID> --json
immich-admin search metadata --original-file-name IMG --all --json
```

### From server errors to asset IDs

Extract failing asset UUIDs from the Immich server log and feed them to
the CLI (PowerShell):

```powershell
ssh SSH_CONFIG_NAME 'docker logs --tail 2000 immich_server 2>&1 | grep ERROR' |
  ForEach-Object {
    [pscustomobject]@{
      id   = if ($_ -match '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}') { $Matches[0] } else { $null }
      line = $_
    }
  } | ConvertTo-Json
```

Keep the matched IDs, inspect, then act:

```powershell
# ids.txt: one UUID per line, e.g. from ($parsed | Where-Object id).id
immich-admin assets info --ids-file ids.txt
```

### Before merge-album, sanity-check the pair

Human output is lossy — always inspect both albums with `--json` first:

```sh
immich-admin albums get <FROM> --json
immich-admin albums get <INTO> --json
```

Compare `startDate`/`endDate` (album time range) and scan `albumName`/
`description` of both for year numbers (e.g. `\b(19|20)\d{2}\b`). For the
per-asset spread, use `albums assets --album-id <ID> --json`
(`localDateTime`/`fileCreatedAt`). If the years or time ranges don't
overlap (e.g. source is "2010 USA", target is "2024 Garten"), the merge is
probably unwanted — stop and ask the user whether to merge, keep the
albums separate, or rename instead of merging blindly. Always run
`client-workflow merge-album --dry-run` first.

### Canonical flows

- **Corrupt hunt**: `client-workflow find-no-thumbhash --type IMAGE -q >
  corrupt.txt` (assets Immich couldn't thumbnail), then `client-workflow
  repair-assets --mode all --ids-file corrupt.txt --dry-run` — `marker`
  fixes JPEGs missing the EOI marker, `tiff-tags` patches zero-count TIFF
  IFD entries, `takeout-json` deletes unrecoverable Google Takeout JSON
  sidecars (trash by default). Re-check with `find-no-thumbhash` after.
- **Search → act**: `search metadata` filters by name, type, favorite,
  album membership, location, camera, dates, OCR (`--all` pages the whole
  library). Useful sets: `--is-not-in-album true` (orphans),
  `--is-offline true`, `--make/--model`, `--city/--taken-after`.
- **Replace**: `client-workflow replace-asset <ASSET_ID> <NEW_FILE>`
  uploads, checksum-verifies, copies metadata (albums, favorite, shared
  links, sidecar, stack), then trashes the original. Aborts on checksum
  duplicates instead of touching the wrong asset.
- **Albums**: `albums create --name [--description]`; `albums assets --album-id/--album-name [--json] [-q]` lists every asset (pipe `-q` into `add/remove-assets --ids-file -`); `albums add-assets /
  remove-assets ALBUM_ID [--ids-file] [--dry-run] [--yes]` (reports
  added-or-removed / already-present / not-found per ID); `albums delete
  [--force]` (refuses non-empty albums without `--force`; the server trashes
  and finishes deletion in a background job, so a deleted album can linger
  briefly). `client-workflow merge-album --from --into
  [--delete-empty-source] [--dry-run] [--yes]` moves every asset out of one
  album into another for duplicate cleanup. `client-workflow download-album --album-name NAME --target-dir
  DIR --size original --sync` mirrors an album (manifest-tracked, safe to
  resume); `--size preview` + `--resize`/`--resize-video-preset` for small
  shareable copies. `fix-album-dates` reconciles date-named albums
  (report-only for year albums). `add-users-to-album-with-pattern` bulk-shares.
- **Tags**: `tags upsert`, `tags bulk-tag`, `tags tag` for assignment;
  `client-workflow tag-delete --include/--exclude` for regex bulk deletion
  (permanent — dry-run first).
- **Upload**: `assets upload FILE...` (timestamps default to file mtime;
  `--sidecar`/`--filename` are single-file only; per-file byte bar on stderr, `--quiet` disables, `--json` implies quiet).
  `assets check-remote-exists FILE|DIR...` (alias `check-bulk-upload`) hashes locally and reports
  `uploaded <id>` / `missing` / `unsupported` without mutating (`--json`, `--ids-only -q` pipes duplicate IDs into `albums add-assets`/`tags tag`).
- **Similarity**: `client-workflow find-similar FILE [--all] [--limit N] [--max-distance 0.01] [--type IMAGE] [--json] [-q]`
  asks the external clip-probe service (https://github.com/dhcgn/immich-clip-probe/, configured via
  `clip_probe.server`/`clip_probe.token` or `IMMICH_CLIP_PROBE_*` env) which library assets look like a local
  file before uploading. Empty matches = nothing close enough (not an error).
- **Watch**: `client-workflow watch-upload --watch-dir DIR --mode flat|by-subfolder [--tag-pattern "immich-admin-cli/watch/{yyyy-MM-dd}"] [--album-id|--album-name] --interval 60s --stable-for 10s [--once] [--dry-run] [--yes]`
  polls and uploads only stable files (bulk-check first; duplicates only linked; `--once` forces `--stable-for 0s`, i.e. uploads immediately).
  A finished folder that nothing is still writing into needs just `--once`; run the loop (no `--once`) only while files are still arriving.
  `client-workflow watch-download (--album-id|--album-name|--tag-id|--tag-value) --target-dir DIR --size original --interval 300s [--once]`
  loops the `.immich-sync.json` manifest sync (album or tag source). `--interval <=0` means run once. Per-interval one-line summary on stderr; `--json` prints JSON stats on stdout.
