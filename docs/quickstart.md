# Takeout Audit — alpha quick start

Extract this archive and run `takeout-audit` (`takeout-audit.exe` on Windows).
Linux/macOS: if needed, run `chmod +x ./takeout-audit` after extraction.

```sh
./takeout-audit --inventory-only --out inventory takeout-001.zip takeout-002.zip
```

Offline inventory exits 2 by design. Supply all ZIPs from one export, or an
extracted directory. Output must be a new directory with an existing parent.
Do not put output inside a directory being scanned.

To check a server, copy `config.example.yaml` to `config.audit.yaml` and set the
server origin and private API key (`user.read`, `asset.upload`). Stable Immich
3.2.x is the initial online scope.

```sh
./takeout-audit --config config.audit.yaml --out audit takeout-001.zip takeout-002.zip
```

Open `summary.json`, then filter `media.csv`; `ledger.jsonl` preserves every
enumerated file. Source occurrences and unique content groups are different.
Exit 0 means scoped completion, 1 means attention/differences, 2 means incomplete.
Catalog matches do not prove server disk integrity or metadata preservation.
Unmatched files are not automatically lost. Never delete originals based only
on this report. This alpha is intended for controlled trials; see repository
CI and validation records for which tests have actually run.
