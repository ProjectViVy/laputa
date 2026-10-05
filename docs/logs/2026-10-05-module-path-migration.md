# ProjectViVy module path migration

The Garden, Mentle and Laputa module identities now match their directories in
`ProjectViVy/laputa`. Production imports, tests, architecture-guard patterns,
the embedding example and entry-point documentation use the canonical paths.
Historical records and agent instruction files are not rewritten.

Laputa now requires INOFY at `v0.0.0-20260930141905-71e2c9bbe47d` rather than
requiring an external `../../INOFY` checkout. Module checksum metadata was
refreshed, including Mentle's previously missing renameio checksums.

## Verification

- Before the fix, VIVY `go list -m all` failed on the missing Garden sibling.
- Before the fix, Laputa's evolution tests failed on the missing INOFY sibling.
- `go test ./...`: passed in each of `laputa`, `mentle`, and `garden`.
- `go vet ./...`: passed in each module.
- Garden console: frozen dependency installation and production build passed.
- Garden architecture guard: zero violations after updating module paths.
- Runtime Go/module scan and `git diff --check`: passed.
- Garden `go test -tags=e2e ./e2e/...`: `TestGardenCleanBreakEndToEnd`
  fails at `POST /v2/memories` with `memory_unavailable` (503 instead of 201).
  The identical failure was reproduced on unchanged main `6f2eed2` with the
  same built console. This is a baseline/environment limitation, not a passing
  end-to-end acceptance claim. Embedding/model provisioning is outside this fix.

## Consumer handoff

VIVY must update its imports and replacements together and prepare the full
Laputa Git checkout at a single pinned commit before invoking Go. Its sealed
go-host packer requires that Git source closure. A version-only or naming-only
consumer update does not close the original fresh-clone startup failure.
