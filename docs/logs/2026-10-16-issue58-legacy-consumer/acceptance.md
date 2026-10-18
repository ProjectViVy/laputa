# Acceptance boundary

- A queued legacy activity beneath a valid v3 owner gains a durable explicit
  consumer policy closure. Its original pending/applying/unknown effect fields
  remain unchanged when no positive old native outcome can be established.
- Real worker Start and restart stop retrying that closed consumer across the
  existing one-second retry interval. HighWatermark and session finalization no
  longer remain permanently fenced on that consumer.
- Session finalization under confirmed v3 does not call the denied FoldSession or
  claim that a new capsule exists. C1 text, revision, digest, coverage and retained
  history/archive bytes remain unchanged.
- Actual committed archived capture entries remain recoverable with their exact
  native IDs and provenance. Missing/partial/conflicting proof is never interpreted
  as a negative result; archive IO/malformed proof remains visible as an error.
- Malformed, unknown, forged or foreign consumer policy metadata does not bypass
  strict decoding/current-owner validation. Failed metadata persistence does not
  produce an acknowledgement or remove the source fence.
- A current invalid/foreign/unknown/uninitialized C1 owner cannot authorize a
  policy closure. Already persisted applied receipts remain readable independently.
- Raw canonical material replay remains independently eligible. Legacy v1/v2
  capture/fold and all v3 legacy-writer denials retain their native behavior.

These outcomes are covered by disposable native fixtures. Windows-specific source
is prepared but actual Windows execution is pending root qualification. This is
not installation, live model, VIVY BML binding or subject projection acceptance.
