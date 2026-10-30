<!-- Parent: ../AGENTS.md -->

# docs/architecture — Current Architecture Decisions

**Current direction:** [ADR-0016: Laputa Protocol and Native Agent Governance](./0016-laputa-protocol-and-native-agent-governance.md)

**Existing Go profile:** [ADR-0012: Laputa Markdown Clean Break](./0012-laputa-markdown-clean-break.md)

**Implementation architecture:** [ADR-0013: Laputa Clean-Break Implementation Architecture](./0013-laputa-clean-break-implementation-architecture.md)

**Mentle recovery authority:** [ADR-0014: Mentle Canonical Authority and Derived-Index Recovery](./0014-mentle-canonical-authority-and-derived-index-recovery.md)

## Use

Read ADR-0016 first for new architecture. Read ADR-0012 before changing the
existing Go profile; it establishes that profile's authority files, context lanes,
deletion rules and module boundaries. Native adapters do not automatically
inherit its file layout, Recap engine or review policy.

## Active ADRs

| ADR | Status | Scope |
| --- | --- | --- |
| 0016 | Accepted direction; implementation pending | Laputa specification/protocol, host-owned persona/Recap, replaceable long-term engine, Garden closed-loop native governance, optional later console |
| 0012 | Accepted | Garden-Laputa contract: seven Markdown authority files, `ACTMEM.MD`, tool-only WORLD/ACTMEM, EvoMap ownership of capability artifacts, deletion-first implementation |
| 0013 | Accepted design | Replacement packages, deletion order, HTTP/Console boundary, EvoMap candidate input, and first vertical-slice acceptance matrix |
| 0014 | Accepted | Canonical SQLite memory authority, transactional index outbox, disposable indexes, safe staged rebuild and Facade-only mutation |
| 0015 | Accepted | `laputa-agent/1` external access plane: canonical REST, thin MCP adapter, typed SDK and AGENT-VIVY built-in preferred MemoryPort lifecycle bridge; no new authority or automatic protected context |
| 0006 | Accepted | Raw-first semantic ingestion and bounded evidence reads. |
| 0007 | Accepted, refined | EvoMap mailbox. Its state and privacy rules remain active; it owns no Laputa authority. |
| 0009 | Accepted | Human report modules outside Laputa authority. |
| 0010 | Accepted, refined | EvoMap Hub GEP-A2A transport and publication gates. |
| 0011 | Proposed, refined by 0014 | Provenance, identity, ingest and health inputs remain; WAL-as-authority and optional-backend sequencing are superseded. |

## Archived ADRs

ADRs 0001, 0002, 0003, 0004, 0005, and 0008 are preserved at [`../archive/2026-08-14-laputa-clean-break/`](../archive/2026-08-14-laputa-clean-break/). They are historical evidence only.

## Documentation Rule

A new active ADR must not reintroduce these retired Go-profile behaviors without
explicitly superseding ADR-0012. Native host files under ADR-0016 are not legacy
Go fallbacks; their write policy is defined by the owning host:

- JSON authority bodies, JSON Patch, or `.laputa/sections/*.json` Persona state.
- `Commitment`, `Preferences`, `MemoryMD`, `MEMORY.MD`, `memory_md`, or `LONGMEM.MD` as target authority concepts.
- Automatic WORLD or ACTMEM projection into a ContextView.
- Generic governance/approval workflows for Persona or ACTMEM.
- Skill creation outside EvoMap, EvoMap mailbox state inside Laputa, or a legacy compatibility path.

Architecture changes require an ADR. Implementation details and test notes belong with their module or in `docs/dev/`.
