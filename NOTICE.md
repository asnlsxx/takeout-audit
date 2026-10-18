# Provenance

This is a focused derivative of [dhcgn/immich-admin-cli](https://github.com/dhcgn/immich-admin-cli),
starting at commit `5514efda26ba46afc3bb2d60a15ff723c3549992`.
The upstream MIT copyright notice is retained in `LICENSE`; the new audit work
is distributed under the same MIT license. Upstream Git history is preserved.

The new standalone command and `cw audit-takeout` extension were developed to
address migration reconciliation. No endorsement by Immich, immich-go, Google,
or the upstream maintainer is implied. This repository contains a generated
Immich API client and inherited administration code; the released dedicated
binary exposes only the audit workflow.

API specifications originate from `immich-app/immich` and are regenerated with
the tooling recorded in this repository. The disposable CI compose fixture is
derived from Immich v3.2.0's documented deployment configuration. Test photos
are generated locally; there are no user photos or Google account exports in
the repository.
