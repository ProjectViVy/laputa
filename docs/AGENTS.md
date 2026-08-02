<!-- Parent: ../AGENTS.md -->

# Documentation — Architecture Decisions & Active Docs

**Generated:** 2026-08-01  
**Status:** vNext architecture proposed — implementation phase 0  
**Archive Policy:** Pre-redesign materials preserved without modification

---

## Purpose

The `docs/` directory is the canonical entry point for Garden MemoryOS transformation documentation. It contains:

- **Active architecture plans** for the vNext implementation
- **ADR (Architecture Decision Records)** for design rationale
- **Historical archive** of pre-redesign decisions (preserved as source evidence)
- **Migration roadmap** and wave-by-wave implementation sequence

---

## Structure

```
docs/
├── README.md                             # Navigation and archive policy
├── architecture/                         # Active design documents (ADRs)
│   ├── 0001-memoryos-vnext-architecture.md  # Complete vNext plan, migration waves, HTTP contracts
│   └── 0002-laputa-cognitive-partition-decision.md # Accepted cognitive partition decision
└── archive/                              # Pre-redesign historical documents
    └── 2026-08-01-pre-memoryos-redesign/
        ├── GARDEN-PLAN-2026-07-08.md
        └── (other pre-vNext design artifacts)
```

### Subdirectories (Depth 2)

Depth-2 AGENTS.md files exist for:

- **[architecture/AGENTS.md](./architecture/AGENTS.md)** — ADR process, decision tracking, vNext specifications
- **[archive/AGENTS.md](./archive/AGENTS.md)** — historical preservation policy, accessing old decisions

---

## Key Documents

### Active: Architecture Plan

**File:** `architecture/0001-memoryos-vnext-architecture.md`

Complete specification covering:

- **0001:** vNext vision, core contracts, recall/ingest flows, module ownership and migration work
- **0002:** accepted Laputa cognitive partition: Frozen Core, STM, `MEMRULES.MD`, `WORLD.MD`, human reports, removed LTM and legacy-section migration constraints

This is the implementation contract. Refer here for all architectural decisions and technical requirements.

### Archive: Pre-redesign Decisions

**Location:** `archive/2026-08-01-pre-memoryos-redesign/`

Preserved without modification as source evidence. Used for historical understanding; not the implementation contract for vNext work because they predate current decisions on:

- Governed MemoryOS
- Progressive recall (Fast/Deep)
- Lifecycle semantics
- External EvoMap/Evolver integration

---

## Documentation Conventions

### When to Write Docs

- New ADR for architectural decisions
- New architecture wave entry/exit gates
- HTTP contract changes
- Module boundary changes
- Performance target updates

### When NOT to Write Docs

- Implementation details (use module AGENTS.md instead)
- Temporary debugging or feature branches
- Test or fixture documentation (use internal AGENTS.md instead)
- Archive material (preserve in archive/, do not modify)

### ADR Format

Architecture Decision Records (ADRs) follow a standard format:

```
# ADR-NNN: Title

**Date:** YYYY-MM-DD  
**Status:** [Proposed | Accepted | Deprecated]

## Context

[Problem or question]

## Decision

[What we decided and why]

## Consequences

[Positive and negative outcomes]

## Alternatives Considered

[Other options and why we rejected them]
```

---

## Active Content

### vNext Architecture (Section Outline)

| Section | Focus | Key Topics |
|---------|-------|-----------|
| 1-3 | Vision & rationale | Governance, degradation, authority isolation |
| 4-6 | Core concepts | MemoryCard, EvidenceFragment, ContextView, lifecycle |
| 7-10 | Ownership & contracts | Module boundaries, HTTP v1/v2, authority rules |
| 11 | Migration waves | Wave 0-7 sequence, entry/exit gates, deliverables |
| 12 | Performance | P95 latencies, LongMemEval results, targets |
| 13 | Degradation | Mentle unavailable, LLM unavailable, graceful fallback |
| 14 | EvoMap integration | Evolver adapter, proposal flow, external hookups |

---

## Testing Requirements

Before considering documentation complete:

- [ ] All code examples in ADRs are tested and working
- [ ] All HTTP contract examples are verified with running server
- [ ] All command examples in migration waves are executable
- [ ] Archive materials are accessible and unchanged
- [ ] Cross-references between ADRs are consistent

---

## Conventions

- **Markdown format:** GitHub-flavored Markdown
- **Code blocks:** Specify language for syntax highlighting
- **Links:** Relative paths for internal cross-references
- **Timestamps:** ISO 8601 (YYYY-MM-DD or YYYY-MM-DDTHH:MM:SSZ)
- **Status:** Proposed, Accepted, Deprecated (for ADRs)

---

## Navigation

**Entry point for new contributors:**

1. Start at `README.md` — understand the archive policy
2. Read `architecture/0001-memoryos-vnext-architecture.md` — comprehensive design
3. Reference module AGENTS.md for implementation details
4. Consult parent `../AGENTS.md` for workspace coordination

**For questions about historical decisions:**

- Check `archive/2026-08-01-pre-memoryos-redesign/` for context
- Cross-reference to vNext architecture for current decisions
- Do not edit archive materials

---

## MANUAL

This document is maintained by the oh-my-claudecode writer agent.

When updating:

1. **Keep this file as single source of truth** for documentation organization
2. **Link to architecture/ and archive/ subdirectories** for detailed content
3. **Do not duplicate** architectural decisions or specifications
4. **Update when:**
   - New ADR is written
   - Architecture wave is completed
   - HTTP contract changes
   - Archive policy is updated
5. **Do not update for:**
   - Implementation details (use module AGENTS.md instead)
   - Temporary materials
   - Archive contents (preserve exactly as-is)

Parent reference: `../AGENTS.md`
