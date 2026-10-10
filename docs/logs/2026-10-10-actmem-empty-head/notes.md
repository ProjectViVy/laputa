# Empty ACTMEM head serialization

A real native pair/fold/reopen recovery test exposed empty entries rendered as a YAML null, which the strict v2 parser rejects. Dedicated empty document round-trip test reproduces the same failure. Emit an explicit empty mapping; keep strict parser admission and v2 grammar unchanged. No automatic legacy head upgrade.
