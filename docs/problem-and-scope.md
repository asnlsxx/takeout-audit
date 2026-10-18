# Problem evidence and scope

Research cutoff: 2026-10-18. These are independent reconciliation requests,
not evidence of a widespread data-loss bug or an estimate of market size.

* [immich-go discussion #650](https://github.com/simulot/immich-go/discussions/650),
  January–February 2025: after an older metadata-name issue was fixed, users
  still struggled to reconcile import counters. The fixed importer bug is not
  a reason to create another uploader; the remaining need is understandable counts.
* [43,000 source items versus about 34,000 processed](https://www.reddit.com/r/immich/comments/1nh869r/immichgo_wont_upload_all_photos/),
  September 2025: the user tried dry runs and options but could not confidently
  account for the difference, eventually resorting to manual uploads.
* [Explaining discarded duplicate assets](https://www.reddit.com/r/immich/comments/1pvhgmk/dealing_with_discarded_assets_when_importing_from/),
  December 2025 with a May 2026 follow-up: debug logs did not explain a large
  duplicate total clearly enough; a user wrote a SHA256 grouping script to
  understand album copies. Fewer uploaded bytes did not establish lost data.

## Existing solutions constrain the opportunity

[immich-go](https://github.com/simulot/immich-go) already handles Takeout uploads,
metadata, albums, duplicates and dry runs. [immich-admin-cli](https://github.com/dhcgn/immich-admin-cli)
already implements local hash checking, which is why this project reuses it.
[PhotoProof](https://github.com/cmer/photoproof) provides resource-level checking
for Apple Photos. Ordinary hashing, being free, and being local are not enough
to justify this fork.

The narrow additional workflow is: supplied Takeout ZIPs → every observed file
and distinct content group → strict account catalog comparison → reports whose
counts and unknown states can be explained. No upload, cleanup, visual matching,
album repair, hosted account or background service is needed.

## What would invalidate further investment

Stop expanding the fork if a maintained existing tool delivers this complete
ZIP ledger, or if 2–3 migration users find existing hash checks sufficient.
If normal imports commonly alter original bytes so that the report cannot
explain differences, reassess the need instead of silently adding a similarity
system. No user adoption has yet been established; feedback should use sanitized
counts and self-generated reproducers, never require private photo exports.

Publication of this repository is not permission to post unsolicited messages
to the linked discussions or imply their authors endorse the implementation.
