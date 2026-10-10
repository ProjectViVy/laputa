# Fold capacity and retained provenance

Keep the existing ACTMEM/capsule schema, 800-character total capsule cap, complete scoped provenance, original entry IDs and deterministic archive names. Cut the writer's acceptance of a single oversized source: preflight all chunks before any archive or head removal. Flow-style YAML values and omission of optional zero fields preserve the same parsed metadata without a new reader or format version.

Captured pairs also bound their body by available capsule capacity using fixed native ID/timestamp widths. Raw admitted source remains complete in ingestion; Pulse/Recap are bounded activity projections. Oversized provenance with no body capacity refuses capture. Lookup applies the same deterministic normalization. Existing accepted ingestion rows are not opted into this new projection.

Ruling: preserve capacity and provenance rather than increase the cap or drop source references. A captured body can be shorter than its section maximum when metadata consumes capacity; the actual two-turn short-fact probe must pass before App wiring is claimed. No automatic ACTMEM context or personality write is added.
