# Recap recovery cancellation fix

Base: `0e0027c5f60ad67614d3613c2d5e21d567c4b412` (Vivy's selected source).

First-use recovery previously marked every failure commit-unknown. A context
canceled before head replacement therefore poisoned an otherwise safe owner.
Preserve the native context error without adding that mark. `publishRecapHead`
still fences the owner and returns commit-unknown once replacement is attempted;
non-context recovery failures retain the conservative fence as well.

The minimal classification fix also appears in PR #12, but that PR is based on
an older source. This change preserves the selected source's later Windows and
proposal-admission fixes instead of moving Vivy to that older branch.

## Verification (Linux amd64, Go 1.26.4)

- Native regression before the fix: 3/3 failures at `before_replace`, followed
  by failed read retry. The regression uses real temporary Markdown storage.
- After the fix: `go test -mod=readonly -run
  '^TestRecapRecoveryCancellationKeepsReplacementBoundary$' -count=10 ./actmem`
  passed, including same-owner read/write retry, fail-closed read/write after
  replacement, and an I/O failure coinciding with cancellation.
- `go test -mod=readonly ./...`: passed for the complete Laputa module.
- `go vet -mod=readonly ./...`: passed.
- `go test -race -mod=readonly ./...`: passed for the complete Laputa module.
- `gofmt` and `git diff --check`: clean.

No schema, API, authority ownership, publication barrier, or dependency changed.
Native Windows execution and the consuming Vivy source-lock update remain
separate acceptance/publication steps; these Linux results do not claim either.
