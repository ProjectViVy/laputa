# Bounded post-model evolution packets

The internal reflection packet carried a full evidence batch beside a memory candidate derived from that batch. A valid 4,500-byte source plus its corresponding candidate produced a 10,006-byte packet and failed the host's existing 8 KiB per-packet bound.

Only the model consumes Batch. Effects and finish consume Input, typed Candidates, NoChangeReason and receipts. Drop the unused batch after decoding model output, preserving complete candidate body/source references, admitted window and receipts. Persisted source authority is unchanged; this cuts redundant disposable stage data, not durable evidence. No new authority, scheduler or backend.

ImplementationRevision advances to diva-cognitive/v1-review-2 so definition/program identity distinguishes the changed stage wire behavior. Existing admitted-run identities are not rewritten. Paired VIVY request preservation is needed to deliver the complete source to the model.
