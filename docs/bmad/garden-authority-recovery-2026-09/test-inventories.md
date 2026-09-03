# Epic 0 Test Inventories

**Story:** E00-S05
**Status:** Complete; executed against the clean-break implementation
**Date:** 2026-09-03
**Rule:** DIVA-observable Persona/ACTMEM behavior is authoritative. The initial planning statuses below are superseded by the Story implementation records and final gate evidence.

## 1. Persona parity matrix

| ID | Contract | Test level | Owner story | Current evidence/status |
|---|---|---|---|---|
| PER-001 | five required documents initialize to revision 1 | unit + HTTP | 1.3/1.6 | PASS; E01-S03/S06 |
| PER-002 | DREAM/DARK remain absent on initialization | unit + HTTP | 1.3 | PASS; E01-S03 |
| PER-003 | any invalid/empty/over-cap input leaves all five live files absent | fault/unit | 1.3 | PASS; E01-S03 |
| PER-004 | abandoned `.init-staging` is safely detected/cleaned | restart/fault | 1.3 | PASS; E01-S03 |
| PER-005 | partial live required-file set returns integrity/repair error | unit + HTTP | 1.3 | PASS; E01-S03/S06 |
| PER-006 | existing revision with base 0 conflicts | unit + HTTP | 1.1 | PASS; E01-S01 |
| PER-007 | exact base revision succeeds and advances once | unit + HTTP | 1.1 | PASS; E01-S01 |
| PER-008 | stale concurrent writers produce one success/one conflict | concurrency | 1.1 | PASS; E01-S01 |
| PER-009 | normalized identical content is no-op with no history change | unit + HTTP | 1.1 | PASS; E01-S01 |
| PER-010 | CRLF/NFC/trim normalization and visible grapheme cap match DIVA fixtures | table/unit | 1.1 | PASS; E01-S01 |
| PER-011 | snapshot/diff files are create-new immutable | fault/unit | 1.2 | PASS; E01-S02 |
| PER-012 | snapshot collision leaves live file/history unchanged | fault/unit | 1.2 | PASS; E01-S02 |
| PER-013 | history log preserves actor/source/reason/base revision/hash | unit + HTTP | 1.2 | PASS; E01-S02/S06 |
| PER-014 | user direct write may update protected documents | policy + HTTP | 1.4 | PASS; E01-S04/S06 |
| PER-015 | agent protected-document write must create review | policy + HTTP/MCP | 1.4/4.3 | PASS; E01-S04/E04-S03 |
| PER-016 | agent direct write limited to DREAM/DARK/USER Observations | policy | 1.4 | PASS; E01-S04 |
| PER-017 | AutoDream actor/kind allowlist matches DIVA | policy | 1.4 | PASS; E01-S04 |
| PER-018 | USER agent review cannot alter Observations | unit/policy | 1.4 | PASS; E01-S04 |
| PER-019 | AutoDream cannot alter USER Preferences | unit/policy | 1.4 | PASS; E01-S04 |
| PER-020 | WORLD review preserves confirmed/user claims and passes R6 bound | unit/policy | 1.4 | PASS; E01-S04 |
| PER-021 | request creation requires ready profile/legal actor-kind/nonempty/cap-valid content | unit + HTTP | 1.5 | PASS; E01-S05/S06 |
| PER-022 | one pending request per kind | unit + HTTP | 1.5 | PASS; E01-S05 |
| PER-023 | stale approval persists state=stale and decided_at | unit + HTTP | 1.5 | PASS; E01-S05/S06 |
| PER-024 | accepted history retains proposal actor/source/reason | unit | 1.2/1.5 | PASS; E01-S02/S05 |
| PER-025 | direct write marks same-kind pending request stale | unit | 1.2/1.5 | PASS; E01-S02/S05 |
| PER-026 | sibling stale updates persist decision timestamps and surface I/O failure | fault/unit | 1.2/1.5 | PASS; E01-S02/S05 |
| PER-027 | agent principal cannot approve/reject review | HTTP/MCP auth | 1.4/4.1 | PASS; E01-S06/E04-S03 |
| PER-028 | actor header spoof cannot elevate principal | HTTP auth | 4.1 | PASS; E01-S06/E04-S01 |
| PER-029 | status detects live-file/history revision or hash mismatch | unit | 1.2/1.3 | PASS; E01-S02 |
| PER-030 | repair is explicit, scoped and never imports retired JSON | fault + HTTP | 1.3/1.6 | PASS; E01-S03/S06 |

## 2. ACTMEM parity matrix

| ID | Contract | Test level | Owner story |
|---|---|---|---|
| ACT-001 | missing ACTMEM returns DIVA-compatible empty/head state | unit | 2.1 |
| ACT-002 | persisted path is `<profile>/actmem/ACTMEM.MD` | unit | 2.1 |
| ACT-003 | front matter revision/updated_at and Pulse/Recap/Work shape round-trip | unit | 2.1 |
| ACT-004 | exact CAS required; stale and zero-bypass cases conflict | unit | 2.1 |
| ACT-005 | normalized no-op preserves revision | unit | 2.1 |
| ACT-006 | Pulse, Recap and Work caps are each 1600 visible chars | table/unit | 2.1 |
| ACT-007 | Pulse item cap 280 and Recap item cap 200 reject overflow | table/unit | 2.1 |
| ACT-008 | failed atomic replace leaves previous valid document | fault/unit | 2.1 |
| ACT-009 | append/edit/complete/drop operations preserve fixed section schema | unit | 2.2 |
| ACT-010 | invalid section/index/path has stable typed error | table/unit | 2.2 |
| ACT-011 | session fold creates at most 800-char bounded capsule | unit | 2.2 |
| ACT-012 | capsule list/read/delete lifecycle survives restart | restart/unit | 2.2 |
| ACT-013 | capsule ID/path traversal is rejected | security/unit | 2.2 |
| ACT-014 | read/query projection is capped at 1200 visible chars | unit + HTTP/MCP | 2.3 |
| ACT-015 | maintenance operations are absent from stable read tool discovery | MCP contract | 2.3/4.3 |
| ACT-016 | ACTMEM update has no Persona files/history side effect | boundary | 2.1/2.2 |
| ACT-017 | ACTMEM update has no automatic Mentle promotion | boundary | 2.1/2.2 |
| ACT-018 | ACTMEM survives Garden process restart and a new session | e2e | 2.8 |
| ACT-019 | bootstrap/Fast/Deep/planner do not call ACTMEM service | negative dependency | 2.6 |
| ACT-020 | ContextView DTO/text/trace contains no ACTMEM field/content | negative e2e | 2.6/2.8 |

## 3. Frozen Core and deletion matrix

| ID | Contract | Test level | Owner story |
|---|---|---|---|
| CTX-001 | first bootstrap captures exactly six bounded projections | unit + HTTP | 2.4 |
| CTX-002 | FrozenCore type has no WORLD or ACTMEM representable field | compile/reflection | 2.4 |
| CTX-003 | same session is immutable after Persona edit | integration | 2.4 |
| CTX-004 | same session snapshot survives Garden restart | restart/e2e | 2.4 |
| CTX-005 | new session sees latest Persona revisions | integration | 2.4 |
| CTX-006 | Persona capture failure fails bootstrap without JSON fallback | fault/e2e | 2.6 |
| CTX-007 | Fast, Deep and bootstrap contain no WORLD body/source/trace | negative integration | 2.6 |
| CTX-008 | explicit WORLD document read still succeeds | HTTP/MCP | 2.8/4.3 |
| CTX-009 | pre-existing `.laputa/sections` is never opened or mutated | filesystem/e2e | 2.7/2.8 |
| CTX-010 | checkpoint and reports survive restart from Garden SQLite | integration | 2.5 |
| CTX-011 | retired governance/cognitive routes return 404 | HTTP contract | 2.7/4.2 |
| CTX-012 | architecture guard `-mode enforce` reports zero runtime violations | repository gate | 2.7/4.8 |
| CTX-013 | Mentle boundary scanner reports zero Persona/WORLD/ACTMEM authority dependency | repository gate | 2.7/4.8 |
| CTX-014 | Console causes no background WORLD/ACTMEM network request | UI behavior | 4.5/4.6 |

## 4. Mentle recovery and concurrency matrix

| ID | Crash/fault condition | Expected invariant | Owner story |
|---|---|---|---|
| MEM-001 | two concurrent updates with expected version N | one success; one typed conflict; version N+1 | 3.1 |
| MEM-002 | update/delete affects zero rows | distinguish not-found from version conflict | 3.1 |
| MEM-003 | duplicate idempotency key | same canonical ID; no duplicate index job | 3.1 |
| MEM-004 | REST memory create/update/delete | canonical row and transactional index job always present | 3.2 |
| MEM-005 | MCP/miner/import mutation | same facade/canonical invariants as REST | 3.2 |
| MEM-006 | canonical commit then process crash before index apply | restart claims and applies pending job | 3.3 |
| MEM-007 | vector Add fails transiently | canonical remains valid; retry metadata/backoff recorded | 3.3 |
| MEM-008 | poison job repeatedly fails | quarantined/failed; unrelated jobs and startup continue | 3.3 |
| MEM-009 | worker crashes after Add before job completion | replay idempotent; no duplicate logical result | 3.3 |
| MEM-010 | lease owner dies | expired lease is reclaimable exactly once | 3.3 |
| MEM-011 | embedding dimension mismatch | typed reject; no mixed index mutation | 3.4 |
| MEM-012 | metric/normalization/model identity mismatch | health degraded and explicit rebuild required | 3.4 |
| MEM-013 | missing legacy identity | deterministic adopt/rebuild policy, never silent guess | 3.4 |
| MEM-014 | delete/corrupt vectors.db only | canonical-derived rebuild restores active search | 3.5 |
| MEM-015 | rebuild embedding fails mid-run | canonical and prior index untouched | 3.5 |
| MEM-016 | staged index verification count/search fails | no swap; typed operator failure | 3.5 |
| MEM-017 | swap fails on Windows | rollback retains prior usable derived index | 3.5 |
| MEM-018 | canonical DB path monitored during repair | never renamed, deleted or overwritten | 3.5 |
| MEM-019 | many old versions/deleted records | active results not crowded out before filtering | 3.6 |
| MEM-020 | BM25 rebuild above 50k active records | no silent fixed-cap omission | 3.6 |
| MEM-021 | canonical/vector/BM25 count divergence | IndexHealth=degraded with reasons | 3.7 |
| MEM-022 | failed/pending jobs and oldest age | accurately exposed through live probe/Garden API | 3.7 |
| MEM-023 | repeated identical session ingest | no duplicate canonical memories | 3.8 |
| MEM-024 | crash after cursor advance boundary | resume without skip or duplicate | 3.8 |
| MEM-025 | two workers ingest same session | durable lease prevents concurrent replacement | 3.8 |
| MEM-026 | legacy JSONL WAL absent/corrupt | canonical memory remains authoritative; rebuild unaffected | 3.9 |
| MEM-027 | service close | DB, vector store, WAL/diagnostic resources close cleanly | 3.9 |

## 5. Gate execution order

```text
1. focused unit / policy / contract tests
2. persistence fault and concurrency tests
3. Garden + Laputa + Mentle full Go suites
4. target clean-break and recovery e2e
5. Console behavior tests and production build
6. architecture-guard -mode enforce
7. git diff --check on owned changes
```

A legacy test that asserted JSON Governance or automatic WORLD projection was classified as **retired behavior** and rewritten or deleted during the Epic 2 clean-break cutover; it is not counted as acceptance evidence.

## 6. Execution evidence

The complete matrix was exercised on 2026-09-03. Focused package tests, the three module suites, real-process clean-break e2e, Console typecheck/build, and the architecture guard enforce scan all passed. The Story records under `implementation/` identify the concrete test package and acceptance evidence for each row. Host adapters and external Skill installation remain Deferred by scope.
