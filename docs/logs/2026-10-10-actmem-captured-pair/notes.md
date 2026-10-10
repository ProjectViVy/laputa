# Native captured activity pair

Add the native owner kernel required by terminal activity projection: opaque canonical-head fingerprint, one atomic Pulse/Recap pair under the existing file-writer mutex, exact retained-entry replay and recovery lookup in original head/capsules. Validate both entries before any write. One stale fingerprint, mismatched identity/body/sources or partial existing pair fails closed. Store allocates real IDs and stamps the commit; its existing atomic Markdown writer is the authority.

No v2 header/schema change or new authority. Fresh append scans only the bounded head; archive discovery is an explicit recovery operation, not a per-turn scan. The caller must durably record the original intent/fingerprint/operation receipts. Missing entries alone never prove no effect; a changed pre-write head remains ambiguous. Existing single-entry API behavior is kept; fresh host capture will opt into the pair kernel. Previously accepted events are not backfilled.
