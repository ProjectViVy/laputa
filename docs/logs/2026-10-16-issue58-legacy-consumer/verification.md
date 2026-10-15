# Verification

All test commands use the unchanged `/tmp/issue58-localonly-runner.py` with
Go1.26.4 at `/workspace/scratch/5dda642e718d/toolchain/go/bin/go`. The runner strips
credentials by name, disables Go dependency networking, installs a non-forwarding
deny proxy, and permits loopback fixtures. Raw logs and command/HEAD/exit/network
metadata are preserved in the task evidence directory; no failed attempt is
replaced by a later result.

## TDD evidence

- `issue58-s5-native-owner-red`: real append/fold/transition/reopen archive lookup
  failed with recap_unavailable. `owner-green`: exact native entries rejoin under
  v3 with explicit archive-only Revision=0, C1/archive bytes unchanged.
- `issue58-s5-native-runtime-red` and `runtime-red2` are fixture setup failures
  (missing state directory, then missing legacy capture binding), retained in full.
  `runtime-red3` is the substantive RED: real runtimecore.Open/worker never closes
  the pending consumer after three seconds. `runtime-green` proves real Open,
  two restarts and observation beyond the one-second interval with one durable
  metadata write, unchanged C1 and no permanent archive/watermark fence.
- `issue58-s5-native-policy-red`: a real capsule-directory IO failure was swallowed
  as policy success. `policy-green`: failure stays visible and the intent unchanged.
- `issue58-s5-native-archive-red` initially passed a too-broad assertion because
  ignoring one malformed capsule yielded a partial-pair error. The stronger
  `archive-red2` failed requiring the actual native malformed error; `archive-green`
  preserves it through the existing parser's strict v3 scan.

## Final focused commands

From `garden/`:

```text
go test -json -p=1 ./internal/ingest ./internal/runtimecore ./agentapi -count=1 -timeout=4m
go test -race -json -p=1 ./internal/ingest ./internal/runtimecore ./agentapi -count=1 -timeout=5m
```

From `laputa/`:

```text
go test -json ./actmem -count=1 -timeout=3m
go test -race -json ./actmem -count=1 -timeout=3m
```

Final Garden normal and race commands each passed **161 tests/subtests and three
packages**, zero failures/skips; elapsed 10.136s and 41.897s. Final Laputa normal
and race commands each passed **120 tests/subtests and one package**, zero failures/
skips; elapsed 0.782s and 4.802s. Every runner metadata file records zero external
requests rejected/forwarded. `gofmt` and `git diff --check` passed. Exact candidate/
source hashes are recorded in the producer report. This tracked record does not
pretend that a future commit SHA was already the HEAD reported by the working-
source test runner.

Coverage includes exact old positive outcomes; legitimate first Before revision
zero; missing/malformed Before; unknown, partial and conflicting old outcomes;
malformed/foreign/incomplete/IO owner state; strict invalid closure fields/digests/
revisions/bindings/duplicate keys; persistence failure; independent raw material
replay; earliest watermark gap; concurrent drain/archive/read; real runtime Open
binding/Start/restart; unchanged authority/archive bytes; existing v1/v2 behavior
and every native v3 legacy writer denial.

Windows-specific missing-head/orphan/zero-view fixture is included. Actual Windows
execution is NOT RUN locally and is a root-owned qualification gate. No cross-
compilation result is presented as execution. Full native module dependency
resolution/remote pin qualification and integrated VIVY CI are root-owned pending
gates. No main merge, publication, new pin, tenant migration or consumer activation
was performed by this source writer.
