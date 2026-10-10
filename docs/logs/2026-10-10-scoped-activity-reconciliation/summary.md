# Scoped activity reconciliation

Garden Collect previously carried only ACTMEM revision. It now explicitly reads current scoped Work and includes complete visible entries with their actual revision/IDs in reconciliation context. Work remains absent from automatic foreground/recall/FrozenCore input and is not a new input watermark source.

Native ACTMEM ReadScoped ignored MaxChars. It now enforces aggregate Unicode body budget after visibility/section filtering. Zero selects the existing1200-code-point explicit cap. As the DTO has no truncation flag, an over-budget read returns actmem_cap_exceeded without partial entry bodies. Garden propagates that closed contract error and never infers on incomplete old Work.

Actual Pulse/Recap production continuity remains missing: two actual App turns leave both sections empty. This repair does not claim archive or automatic continuity. No authority migration/new store/release.
