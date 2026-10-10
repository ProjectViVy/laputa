# Decisions

Use existing ReadScoped and existing EvidenceBatch.Entries, not a new authority/context store. Existing Work is context for reconciling current Work, not fresh captured activity. Scope union comes from the bound native authority.

Fail over-budget reads because the current typed result has no partial-body/truncation field. Silent prefixes could cause model changes against incomplete old Work. Cost: a large single Work section needs a future bounded pagination/typed partial-read contract or explicit reduction before whole-section reconciliation; no cap expansion or contract disguise.

The real S05 continuity regression remains RED for missing Pulse/Recap. Original acceptance gates remain intact; Windows/live availability does not explain this local code gap.
