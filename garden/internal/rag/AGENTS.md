<!-- Parent: ../AGENTS.md -->

# garden/internal/rag — Recall Planning

This package provides deterministic and optional LLM-assisted planning for explicit Deep Recall. Fast Recall remains deterministic and does not call the planner, KG or timeline.

## Boundaries

- Recall consumes session-frozen `personactx.FrozenCore` and bounded Mentle cards/evidence through narrow interfaces.
- Runtime evidence/scope/privacy policy belongs to Garden; it does not become a Persona authority projection.
- WORLD and ACTMEM tool services must not be dependencies of planner, Fast/Deep Recall or ContextView assembly.
- Mentle cards are discovery candidates; full evidence is read only after selection and budget enforcement.
- LLM failure degrades explicitly to the deterministic planner. It does not permit a JSON Governance fallback.

## Verification

- Fast Recall never invokes planner/KG/timeline.
- Deep Recall emits a trace for success and degraded paths.
- Context rendering contains Frozen Core and selected bounded evidence only.
- Tests assert WORLD/ACTMEM fields are structurally absent, not merely empty.

```bash
cd garden
GOSUMDB=off go test ./internal/rag/... ./internal/recall/...
```

Parent reference: `../AGENTS.md`
