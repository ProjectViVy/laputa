# ADR-0019: Standalone Garden v1 proposal

**Status:** proposed — awaiting written-spec review; no implementation authorized
**Date:** 2026-10-12

The authoritative proposal is [Standalone Garden v1 — architecture specification](../superpowers/specs/2026-10-12-garden-standalone-v1-design.md). Its §9 owns the decision/deprecation map; its §10 owns the owner questions. The [proposed Epic/Story index](../superpowers/plans/garden-standalone-v1/index.md) owns dependencies and status. This ADR is a status/pointer record, not a competing specification.

Proposed decisions: one local Garden owner with a subject registry; unique principal-bound grants for every API path; one selected canonical memory backend with qualified Mentle and BML implementations; one scope-safe subject Recap; explicitly opted-in four-host integrations; persisted DeepSeek setup; and removal of standalone autonomous evolution while preserving shared DIVA write/receipt infrastructure.

Preserve ADR-0012 Persona/Frozen Core/WORLD boundaries, ADR-0014 Mentle canonical authority, ADR-0016 modular library ownership, and ADR-0018 selected VIVY Recap policy. If approved, this proposal refines single-profile authentication/composition, backend-selection restrictions, active Pulse/Work consumers, standalone evolution/console composition and host activation assumptions only as enumerated in the specification. It does not silently change existing VIVY or DIVA backend defaults or historical acceptance results.

Current accepted contracts continue to govern shipped behavior. No executable Story plans or Ready implementation are claimed. Written-spec approval permits the next planning stage; reviewed implementation plans and execution selection remain separate gates.
