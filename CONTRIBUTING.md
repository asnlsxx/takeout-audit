# Contributing

Keep this tool focused on explaining supplied source entries and account
catalog results. Prefer upstream contributions when a change is useful to
immich-admin-cli generally. Read `AGENTS.md` for repository conventions.

Before submitting changes, run `go generate ./...`, `go test ./...`,
`go build ./...`, `go vet ./...` and `go fix -diff ./...`.
Report changes must preserve one terminal state per source entry, distinct
content counts and explicit unknown/error states. Never infer deletion safety.

For a bug report, include tool/server versions, exit code, a sanitized summary
and a small self-generated reproducer. Remove account IDs, server addresses,
personal filenames and absolute paths before posting reports. Never attach
API keys, private configuration, original photo collections or location metadata.

Run the real lifecycle test only on a disposable instance, with the explicit
opt-in documented in `test/immich` and the CI workflow. It creates and cleans
its own fixtures. Ordinary tests use no real server. Keep test data small.
