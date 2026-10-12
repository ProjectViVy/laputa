# ADR-0019: Standalone Garden v1 proposal

**Status:** proposed decisions; planning authorized — Story-plan review pending; no runtime implementation authorized
**Date:** 2026-10-12

The authoritative proposal is [Standalone Garden v1 — architecture specification](../superpowers/specs/2026-10-12-garden-standalone-v1-design.md). Its §9 owns the decision/deprecation map; its §10 owns the owner questions. The [proposed Epic/Story index](../superpowers/plans/garden-standalone-v1/index.md) owns dependencies and status. This ADR is a status/pointer record, not a competing specification.

Reconciliation: [pre-existing issue #8](https://github.com/ProjectViVy/laputa/issues/8) was missed in the original overlap audit; specification §1.3 records the correction. Its lean Mentle sharing slice is distinct from the user's broader standalone requirements, and grants no implementation authorization.

Proposed decisions: reuse `ProfileID` as the subject; trusted per-host binding plus named shared-space read/read_write membership; one local writer and one canonical Mentle source for the first sharing slice; independent real BML qualification; one subject Recap with restricted-session policy still for review; opted-in four-host integrations, full UI and persisted DeepSeek setup; and autonomy removal preserving shared DIVA infrastructure. No general IAM/per-record ACL, delegated capability framework, derived-content taint system or optional surface rewrite is required. Unsupported optional multi-persona agent surfaces may be unavailable while existing single-persona capabilities remain.

Owner decisions remain explicit: broader backend release timing (Q2) and restricted-workspace Recap continuity (Q3) are not settled by issue #8. The index separates the lean sharing milestone from broader UI/host/setup and backend acceptance. No full backend-swappability claim is permitted before actual alternate qualification.

Preserve ADR-0012 Persona/Frozen Core/WORLD boundaries, ADR-0014 Mentle canonical authority, ADR-0016 modular library ownership, and ADR-0018 selected VIVY Recap policy. If approved, this proposal refines single-profile authentication/composition, backend-selection restrictions, active Pulse/Work consumers, standalone evolution/console composition and host activation assumptions only as enumerated in the specification. It does not silently change existing VIVY or DIVA backend defaults or historical acceptance results.

Current accepted contracts continue to govern shipped behavior. The owner development assignment now authorizes concrete Story planning and publication on the existing branch; the index links 13 plans and owns their Ready-for-review/Blocked state. No Story is Ready for runtime implementation. Q2/Q3 remain open; parent/owner plan review precedes the preserved native development-session execution method. This status update changes no runtime contract or historical outcome.
