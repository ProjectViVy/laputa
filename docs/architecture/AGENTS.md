<!-- Parent: ../AGENTS.md -->

# docs/architecture — Current Architecture Decisions

**Selected context policy:** [ADR-0018](./0018-vivy-unified-context-and-recap-policy.md), accepted direction with implementation pending.

**Preserved authority contract:** [ADR-0012: Laputa Markdown Clean Break](./0012-laputa-markdown-clean-break.md)

**Implementation architecture:** [ADR-0013: Laputa Clean-Break Implementation Architecture](./0013-laputa-clean-break-implementation-architecture.md)

**Mentle recovery authority:** [ADR-0014: Mentle Canonical Authority and Derived-Index Recovery](./0014-mentle-canonical-authority-and-derived-index-recovery.md)

## Use

Read ADR-0018 together with ADR-0012 before implementing any Laputa, Persona, ACTMEM, ContextView, Evolution, or EvoMap change. It establishes the authority-file contract, context lanes, clean-break deletion rules, and module boundaries.

## Active ADRs

| ADR | Status | Scope |
| --- | --- | --- |
| 0019 | Baseline partly implemented; controlled-editing/native-Persona increment planned and self-reviewed; runtime pending | Standalone Garden; D13–D15 local edit rights, fresh retry and actual native host updates; index owns Story readiness |
| 0018 | Accepted direction; implementation pending | VIVY unified context, subject-level free-form Recap, revision-aware human control, targeted clear, deferred tools and capability-aware backend ownership |
| 0017 | Proposed change record; v2 implementation recorded | Existing Mission/Frozen Core v2, scoped ACTMEM and backend/strategy deltas; selected ACTMEM policy refined by 0018 |
| 0012 | Accepted; ACTMEM policy refined by 0018 | Garden-Laputa contract: seven Markdown authority files, `ACTMEM.MD`, tool-only WORLD and baseline ACTMEM exclusion (selected policy refined by 0018), EvoMap ownership of capability artifacts, deletion-first implementation |
| 0013 | Accepted design | Replacement packages, deletion order, HTTP/Console boundary, EvoMap candidate input, and first vertical-slice acceptance matrix |
| 0014 | Accepted | Canonical SQLite memory authority, transactional index outbox, disposable indexes, safe staged rebuild and Facade-only mutation |
| 0015 | Accepted, refined by 0016 | External `laputa-agent/1` REST/MCP wire profile, typed SDK and host lifecycle; Vivy transport default no longer frozen to REST |
| 0016 | Accepted direction | Three embeddable domain libraries plus one Laputa application; one policy/conformance path for embedded and wire adapters |
| 0006 | Accepted | Raw-first semantic ingestion and bounded evidence reads. |
| 0007 | Accepted, refined | EvoMap mailbox. Its state and privacy rules remain active; it owns no Laputa authority. |
| 0009 | Accepted | Human report modules outside Laputa authority. |
| 0010 | Accepted, refined | EvoMap Hub GEP-A2A transport and publication gates. |
| 0011 | Proposed, refined by 0014 | Provenance, identity, ingest and health inputs remain; WAL-as-authority and optional-backend sequencing are superseded. |

## Archived ADRs

ADRs 0001, 0002, 0003, 0004, 0005, and 0008 are preserved at [`../archive/2026-08-14-laputa-clean-break/`](../archive/2026-08-14-laputa-clean-break/). They are historical evidence only.

## Documentation Rule

A new active ADR must not reintroduce any of the following without explicitly superseding ADR-0012:

- JSON authority bodies, JSON Patch, or `.laputa/sections/*.json` Persona state.
- `Commitment`, `Preferences`, `MemoryMD`, `MEMORY.MD`, `memory_md`, or `LONGMEM.MD` as target authority concepts.
- Automatic WORLD projection, or ACTMEM admission outside the selected ADR-0018 scope/budget policy. ACTMEM never becomes a Frozen Core slot.
- Generic governance/approval workflows for Persona or ACTMEM.
- Skill creation outside EvoMap, EvoMap mailbox state inside Laputa, or a legacy compatibility path.

Architecture changes require an ADR. Implementation details and test notes belong with their module or in `docs/dev/`.
