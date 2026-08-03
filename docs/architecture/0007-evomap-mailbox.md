# ADR-0007: EvoMap Mailbox

**Status:** accepted  
**Date:** 2026-08-03  
**Supersedes:** none  
**Depends on:** ADR-0001 §9 (EvoMap/Evolver integration), ADR-0002 §3.5 (no `proposal_inbox` reuse; mailbox state changes audited)

---

## 1. Context

ADR-0002 removed the legacy `11-proposal_inbox` as the EvoMap proposal system and mandated a separately designed mailbox with explicit inbox/outbox, state machine, evidence references, privacy gate, evaluation, approval/rejection, retry, and dead-letter semantics. The live proposal machinery (`garden/internal/evolution`: runs, candidates, proposals, `NormalizeCandidate` + `LeakageReport`, `HubPolicy`) has no durable communication channel for external Evolver exchange.

This ADR designs that mailbox and delivers its core implementation.

## 2. Decision Summary

1. New Garden package `internal/mailbox`, bridging `evolution.Service`; the laputa `11-proposal_inbox` section is never written (remains `tbd` placeholder, guarded).
2. Inbox and outbox share one durable store with per-direction state machines.
3. A privacy gate runs before any outbox item leaves the local boundary; payloads carrying prohibited references are rejected and never sent.
4. Hub publication stays disabled by default; the mailbox default target is local-only.
5. Persistence: SQLite in the garden state database, same pattern as the evolution store.
6. State changes are explicit, user-approved where required, and audited.

## 3. State Machines

Inbox (inbound evidence bundles from an external Evolver):

```
received → evaluating → approved
                     ↘ rejected
```

Outbox (outbound candidates/proposals):

```
queued → sending → acked
                ↘ failed → (retry_count++ ≤ max) → queued
                         ↘ (retry_count > max)  → dead_letter
```

Rules:

- Every transition records `updated_at` and a reason; illegal transitions are rejected with an error.
- Inbox approval is an explicit user action (never automatic).
- Retry cap defaults to 3; dead-lettered items stay readable and are never auto-retried.

## 4. Privacy Gate

Before an outbox item transitions `queued → sending`:

- `evolution.ValidateBundleInput` bounds (evidence refs, content hashes, source revision only — ADR-0001 §9.1 input boundary).
- `evolution.NormalizeCandidate`-style leakage scan over the payload: prohibited keys and refs (raw corpus paths, personality/cognitive files, tokens, private absolute paths — `isProhibitedRef` rules) block dispatch.
- Blocked items transition to `dead_letter` with reason `privacy_gate`, carrying the `LeakageReport`.

Mechanical checks are necessary but not sufficient; semantic privacy review remains a later batch (recorded, not implemented).

## 5. Hub Policy

- `DefaultHubPolicy` remains: `CanPublishHub` / `CanInstallArtifact` require explicit approval; default is disabled.
- The mailbox has no network transport in this batch: `acked` is produced by a local delivery hook; publishing to any Hub requires a future, separately approved batch.

## 6. Persistence

SQLite (garden state db):

```sql
CREATE TABLE mailbox_items (
  id TEXT PRIMARY KEY,
  direction TEXT NOT NULL,          -- inbox | outbox
  state TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  evidence_refs_json TEXT NOT NULL,
  leakage_json TEXT NOT NULL DEFAULT '',
  reason TEXT NOT NULL DEFAULT '',
  retry_count INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX mailbox_state ON mailbox_items(direction, state, updated_at);
```

The mailbox carries communication state only; proposal bodies remain authored in the evolution store (single source of truth for proposals).

## 7. Audit

Approve/reject, dead-lettering, and any future Hub-bound transition are explicit, audited actions through the existing governance mutation/audit channel (same pattern as evolution proposal review). Ordinary retry bookkeeping is logged but not a governance mutation.

## 8. Non-goals

- Mailbox console UI (deferred, ADR-0003).
- External Evolver process policy / MCP sidecar (ADR-0001 §9.2, later batch).
- Semantic privacy review; Hub fetch/publish transport.
- Any write to `11-proposal_inbox` or other laputa sections.

## 9. Test Matrix

| # | Test | Layer |
|---|---|---|
| 1 | Legal/illegal state transitions (both directions) | mailbox unit |
| 2 | Retry increments then dead-letters past cap | mailbox unit |
| 3 | Privacy gate blocks prohibited refs/payload keys → dead_letter + report | mailbox unit |
| 4 | Hub policy default denies publish/install without explicit approval | evolution policy |
| 5 | v2 endpoints list inbox/outbox/dead-letter; approve/reject mutate state | server |
| 6 | Zero writes to `11-proposal_inbox` across all flows | laputa regression |

## 10. Consequences

- External evolution communication has a durable, bounded, auditable channel.
- No new authority concept; proposals still require explicit Laputa application.
- Hub stays off; nothing leaves the host without a future approved batch.
