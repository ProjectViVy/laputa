<!-- Parent: ../AGENTS.md -->

# garden/internal/rag — Recall Planning

> **2026-10-11 policy supersession:** WORLD remains tool-only. ACTMEM context/schema/tool assumptions below are the baseline; selected VIVY subject-Recap policy overrides only those assumptions. Preserve existing Frozen Core immutability and Mission/Persona authorization. See [ADR-0018](../../../docs/architecture/0018-vivy-unified-context-and-recap-policy.md). Original design, interface and verification records below remain evidence of their recorded snapshots; this notice does not change their pass/fail results or claim the new policy is implemented.

This package provides deterministic and optional LLM-assisted planning for explicit Deep Recall. Fast Recall remains deterministic and does not call the planner, KG or timeline.

## Boundaries

- Recall consumes session-frozen `personactx.FrozenCore` and bounded Mentle cards/evidence through narrow interfaces.
- Runtime evidence/scope/privacy policy belongs to Garden; it does not become a Persona authority projection.
- WORLD tool services must not be dependencies of automatic context. Baseline ACTMEM exclusion remains tested; selected Recap admission uses the existing VIVY context policy, not a second recall authority.
- Mentle cards are discovery candidates; full evidence is read only after selection and budget enforcement.
- LLM failure degrades explicitly to the deterministic planner. It does not permit a JSON Governance fallback.

## Verification

- Fast Recall never invokes planner/KG/timeline.
- Deep Recall emits a trace for success and degraded paths.
- Baseline context rendering contains Frozen Core and selected bounded evidence; the new selected policy additionally admits bounded Recap through existing policy controls.
- Baseline tests assert WORLD/ACTMEM absence. Selected-policy tests must retain WORLD absence and prove ACTMEM scope/budget admission, revision invalidation and conflict handling.

```bash
cd garden
GOSUMDB=off go test ./internal/rag/... ./internal/recall/...
```

Parent reference: `../AGENTS.md`
