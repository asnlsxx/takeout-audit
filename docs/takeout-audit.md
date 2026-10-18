# Takeout Audit

Experimental extension based on `dhcgn/immich-admin-cli` commit
`5514efda26ba46afc3bb2d60a15ff723c3549992` (MIT). The API spec was refreshed
from upstream on 2026-10-18; its info
version is `3.3.0-rc.0`. Runtime audit compatibility is intentionally restricted
to stable **3.2.x**. Mock contracts and a disposable real **3.2.0** lifecycle
[passed](https://github.com/asnlsxx/takeout-audit/actions/runs/39115445636).
Other patch versions and private user exports have not been individually tested.

## Run

Supply every ordinary ZIP from one Google Takeout export, or an extracted
directory. The tool cannot determine whether additional export parts are missing.
PowerShell does not expand wildcard arguments for all native applications; use
an array when selecting several ZIP files:

```powershell
$parts = @(Get-ChildItem -LiteralPath 'D:\Exports' -Filter '*.zip' -File | Sort-Object Name | ForEach-Object FullName)
.\takeout-audit.exe --inventory-only --out 'D:\Reports\inventory-01' @parts
# Expected exit code 2: inventory succeeded but remote verification did not run.
```

For remote checks, copy `config.example.yaml` to a private, ignored config file
and set `server` to an origin such as `https://immich.example.org` without `/api`.
The config file must exist even when `IMMICH_SERVER` and `IMMICH_API_KEY` override
its fields. Store the key in that private file or an environment variable,
not an inline shell argument. Required permissions are `user.read` and
`asset.upload`; the latter is required by the checksum endpoint despite this
command making no upload. Do not commit real configs.

```powershell
.\takeout-audit.exe --config config.audit.yaml --out 'D:\Reports\audit-01' @parts
```

The output directory must not already exist, its parent must exist, and it must
be outside all directory inputs. Files are opened exclusively. No source files
are modified; ZIP members are streamed without extraction. Keep input files
unchanged during a run. Ctrl+C requests cancellation and writes an incomplete
report of observed entries; forcibly killing the process cannot guarantee a
report. No resume/cache support is included yet.

## Read the results

* `ledger.jsonl`: one record per enumerated file, including JSON and unknown
  extensions. Stable within-run IDs use source ID and member index, never name
  alone. JSONL retains UTF-8 paths; `path_raw_base64` preserves original bytes
  for legacy non-UTF-8 names. CSV formula-like cells gain an apostrophe.
* `media.csv`: media records for human filtering, with group IDs and source-copy
  counts. Each Live Photo resource is separate; a matched JPEG cannot cover MOV.
* `summary.json`: input paths, account ID, server version, counts, limitations,
  errors and `run_complete`. It is written last, after the two ledgers close.
  A missing/invalid summary is never success.

`statuses` counts source occurrences; `group_statuses` counts distinct media
contents. `duplicate_media_copies` is hashed media occurrences minus distinct
content groups. Each source also records its observed entry count. `bytes_read`
includes metadata and unsupported files read to check stream/CRC integrity.

| Status | Meaning |
|---|---|
| `catalog_match` | Same protocol checksum found, explicitly not in trash |
| `trashed` | Matching record is in trash |
| `not_matched` | No byte-identical record found in this account's query scope |
| `unknown` | Not queried, unsupported response/version, missing fields, request failure or local SHA1 collision |
| `read_error` | Read/CRC/size/limit failure; never submitted as a valid hash |
| `metadata_not_checked` | JSON read successfully; semantic migration not verified |
| `unclassified` | Unknown extension, retained but not queried |
| `excluded` | Symlink, nonregular file or overlong member name; reason retained |

Copies of the same bytes in annual and album directories remain separate ledger
rows with one content group. SHA1 serves the Immich protocol; SHA256 plus size
separates local content and detects local SHA1 collisions. Server byte contents
are not downloaded or rehashed. Edited/reencoded media and external libraries
can legitimately be unmatched. Source metadata, album relationships, export
completeness, server disk integrity and backup restorability remain unverified.
Never use this result alone as authorization to delete originals.

Exit codes: **0** complete scoped audit with all classified media matched and no
other attention items; **1** completed with differences, excluded/unknown-format
files or no media; **2** incomplete (including offline inventory, read errors,
corrupt/unreadable input, unsupported server or incomplete API results).
`metadata_not_checked` does not make a scoped content audit incomplete; this
explicit exclusion is always recorded. Unenumerated sources do.

Limits default to 500,000 entries, 64 GiB per file and 2 TiB total uncompressed
bytes, configurable with `--max-entries`, `--max-file-bytes` and
`--max-total-bytes`. Memory grows with entry count, not total media bytes; ZIP
central-directory metadata is still read by Go's ZIP library. This is not a
general hostile-archive sandbox. Encrypted ZIP, split-volume ZIP and tgz are
unsupported. Hidden files are included; directory inputs must be a stable
snapshot. Repeated physical inputs (including hard-link aliases) and overlapping
directory inputs are rejected before media reads. Different ZIPs containing the
same media remain separate source occurrences, as intended.

## Verification and release gate

The automated suite includes valid generated PNGs, multi-ZIP duplicates,
duplicate member names, CRC corruption, limits, cancellation, CSV injection,
mock HTTP contracts, missing/duplicate/extra response IDs, trash flags, account
failure, version rejection, CLI output/exit codes, unchanged input bytes and
transport rejection of destructive requests and redirects. Fake endpoints do
not stand in for an Immich deployment.

```sh
go test ./...
go vet ./...
go generate ./...
go build ./...
go test -tags integration ./internal/commands -run AuditTakeoutReadOnlyIntegration -v
```

The opt-in integration smoke test skips without `config.staging.yaml`. It only
queries a server. A separate lifecycle test runs only with explicit disposable
opt-in on loopback: it creates two generated PNG assets, trashes one, leaves a
third absent and audits all three plus a source copy using a restricted key.
It compares asset statistics/states before and after audit, and cleans up its own
fixtures. `scripts/start-immich-ci.sh` is restricted to disposable GitHub Actions
runners; `test/immich/compose.yml` pins Immich 3.2.0 and disables ML downloads.
That test is selected by its exact name, never by running all staging tests.
Configuration of this job is not evidence it has passed. Check actual CI logs.

Before a downloadable public alpha: pass real-instance lifecycle validation and
native CI on Windows/Linux/macOS. Before calling this stable: test a large owned
export, get 2–3 target-user trials, review permission failures on a live server,
and seek upstream technical review. Do not imply upstream endorsement. Local
cross-compilation and a bounded synthetic benchmark do not replace those checks.
The dedicated `takeout-audit` binary has no administrative or updater commands.

## Implementation map

* `internal/takeoutaudit`: network-free scan, grouping, ledger, CSV, summary.
* `internal/workflows/takeoutaudit.go`: generated API types, account/version
  checks, batches of 200, bounded retries, strict result mapping.
* `internal/client/audit.go`: opt-in transport allowlist; redirects rejected.
* `internal/commands/clientworkflow_audit.go`: CLI, cancellation and reports.
* The existing client behavior is unchanged unless `WithAuditOnly` is requested;
  missing JSON body checks additionally prevent identity/version panics.
* The root executable now honors explicit CLI exit codes. The native
  `server version` command exposes the API operation used by this workflow.

All new code follows the repository MIT license. Generated API updates are
included because the repository requires current-spec regeneration. No new
third-party runtime dependency was added.
