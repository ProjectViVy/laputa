# Native fold capacity repair

Real captured-entry provenance exposed a writer/reader mismatch: FoldSession wrote a 945-character capsule that ReadCapsule rejected, then removed its sources from the head. A second test proved oversized metadata was accepted and removed. The native writer now preflights every single source, uses compact semantically equivalent YAML, and refuses an unrepresentable archive before writing or removing anything. Capture normalization reserves space for full metadata.

This is native kernel proof, not complete S05 or crash-matrix acceptance. App session-delete wiring and source/receipt projection are separate work in progress. No release, merge, dependency repin, capacity increase or source lock update.
