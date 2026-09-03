# ADR-0007: EvoMap Mailbox

**Status:** accepted  
**Date:** 2026-08-03  
**Supersedes:** none  
**Refines:** ADR-0012 §5. EvoMap remains the only capability-artifact domain; its mailbox is independent of Laputa authority files.

---

## 1. Context

This mailbox replaces the retired proposal-inbox idea with a separately designed inbox/outbox channel: explicit state machines, evidence references, privacy gate, evaluation, approval/rejection, retry, and dead-letter semantics. The live proposal machinery (`garden/internal/evolution`: runs, candidates, proposals, `NormalizeCandidate` + `LeakageReport`, `HubPolicy`) requires this durable communication channel for external Evolver exchange.

This ADR designs that mailbox and delivers its core implementation.

## 2. Decision Summary

1. New Garden package `internal/mailbox`, bridging `evolution.Service`; no Laputa authority file or `ACTMEM.MD` is written by mailbox flows.
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

- `evolution.ValidateBundleInput` bounds evidence references, content hashes, and source revisions.
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

- Mailbox console UI was deferred when this ADR was written; console work now follows ADR-0012.
- External Evolver process policy / MCP sidecar is a later batch.
- Semantic privacy review; Hub fetch/publish transport.
- No write to Laputa authority files or `ACTMEM.MD`.

## 9. Test Matrix

| # | Test | Layer |
|---|---|---|
| 1 | Legal/illegal state transitions (both directions) | mailbox unit |
| 2 | Retry increments then dead-letters past cap | mailbox unit |
| 3 | Privacy gate blocks prohibited refs/payload keys → dead_letter + report | mailbox unit |
| 4 | Hub policy default denies publish/install without explicit approval | evolution policy |
| 5 | v2 endpoints list inbox/outbox/dead-letter; approve/reject mutate state | server |
| 6 | Zero writes to Laputa authority files and `ACTMEM.MD` across all flows | boundary regression |

## 10. Consequences

- External evolution communication has a durable, bounded, auditable channel.
- No new Persona authority; EvoMap retains explicit artifact review and authorization.
- Hub stays off; nothing leaves the host without a future approved batch.
