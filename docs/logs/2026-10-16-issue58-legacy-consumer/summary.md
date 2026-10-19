# Native v3 legacy-consumer compatibility

This producer checkpoint closes incompatible ACTMEM activity delivery beneath a
positively validated v3 owner without claiming an uncertain old effect succeeded
or failed. It is the native compatibility prerequisite for VIVY issue #58, not
VIVY backend activation or installed-product acceptance.

Garden passes its already resolved trusted RecapBinding to the existing ingest
before the real worker starts. A valid same-owner ReadRecap with revision > 0 is
the policy proof. Missing/uninitialized heads, malformed/foreign/incomplete chains,
live C1 unknown state, generic unavailable and IO errors cannot authorize closure.
ReadRecap retains its native identical-byte recovery publication/barrier.

The existing ingestions.activity_json gains an optional typed legacy_consumer
disposition containing policy identity, revision and digest. Original effect
status, Before, source, error and receipt remain separate. Unknown effects stay
unknown; policy closure does not produce negative or applied outcomes, entry IDs,
or C1 coverage. Existing applied receipts return unchanged before current policy
checks. Closure persistence failure leaves the intent and fence intact.

The same strict decoder and needsLegacyActivity predicate serve Start, drain,
HighWatermark and ArchiveSession. Each reads an ordered SQLite statement snapshot,
closes rows, then enters owner IO. This replaces four divergent JSON-in-SQL
predicates so malformed or unknown closure metadata cannot suppress recovery.
HighWatermark stays conservative at the earliest unresolved row. Raw material
accepted/running/spooled replay remains an independent Start branch. ArchiveSession
uses the same serialized native projection and returns a policy no-fold result
under proven v3, while FoldSession itself continues to reject v3.

Laputa's native LookupCaptured can rejoin exact complete legacy archive entries
under a validated, non-unknown v3 owner using the existing parser. Archive-only
Revision=0 explicitly means no current legacy head revision; a C1 revision is
never substituted. Garden resolves an old attempt to Before.Revision+1 only with
a structurally complete valid persisted original Before and exact positive pair
evidence. Revision zero with a valid native hash is a legitimate first snapshot.
Missing/partial/conflicting proof or missing/malformed Before leaves the effect
unresolved. Archive IO/malformed data remains an error. V3 archive scans are strict;
the existing v1/v2 scan behavior is retained.

The original Before is trusted native attempt metadata; capsules do not retain
the pre-append head bytes. This change does not claim cryptographic reconstruction
of a pre-head hash after arbitrary external SQLite tampering.

The producer branch starts at bb57e93a61cfb41460524d00a557a3a9f2cce1e7 and
ordinarily merges main f44ba00eb6ab5aa6ab9a2f5c09868373e36679f5. Merge
1ce1c1e322ff278db258044dfddb640dcf29101c has ordered parents bb57e93 and
f44ba00, tree a6d4515158008ffc96dc51f7ee546e13115d3745. All 71 merged
files are policy documentation/instructions; the merge has no Go source changes.

No public Backend method, new recovery API, database/schema migration, store,
scheduler, authority parser, VIVY source, BML source, dependency pin or UI change
is included. WORLD, human Persona/Mission, ordinary legacy v1/v2 and retained raw
evidence boundaries remain unchanged. Independent review, publication, exact new
pin qualification and VIVY consumer activation belong to the root stage.
