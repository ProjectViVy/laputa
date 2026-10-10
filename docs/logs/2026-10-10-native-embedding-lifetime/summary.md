# Summary

Canceled native embedding no longer races model resource destruction. Real ONNX deadline/Close race RED becomes three repeated race GREENs. The owner also rejects embeddings after shutdown and serializes access to the shared pipeline.

This is developer evidence, not full S11 or sealed-candidate acceptance. Original dependency pins, authority and profile isolation remain unchanged. No push, merge or release.
