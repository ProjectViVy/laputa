<!-- Parent: ../AGENTS.md -->

# docs/architecture — Architecture Decision Records & Design Specs

**Generated:** 2026-08-01  
**Purpose:** ADRs, design decisions, and migration wave specifications

---

## Purpose

The `architecture/` directory contains active, current design documents for Garden MemoryOS vNext:

- **Architecture Decision Records (ADRs)** with rationale and consequences
- **Complete specification** of module responsibilities and HTTP contracts
- **Migration wave sequence** with entry/exit gates and deliverables
- **Performance targets** and degradation modes
- **Authority boundaries** and immutable ownership rules

---

## Structure

```
architecture/
├── 0001-memoryos-vnext-architecture.md   # Complete vNext specification (sections 0-14)
└── (additional ADRs as decisions are made)
```

---

## Key Document

### 0001-memoryos-vnext-architecture.md

The single authoritative source for Garden MemoryOS vNext design.

**Sections:**

| Section | Content | Use For |
|---------|---------|---------|
| 0-2 | Executive summary and scope | High-level overview, stakeholder communication |
| 3-6 | Core concepts | MemoryCard, EvidenceFragment, ContextView, lifecycle |
| 7-10 | Module ownership and contracts | Authority boundaries, HTTP v1/v2, API stability |
| 11 | Migration waves 0-7 | Implementation sequence, entry/exit gates, deliverables |
| 12 | Performance targets | P95 latencies, recall metrics, LongMemEval results |
| 13 | Degradation modes | Mentle unavailable, LLM unavailable, graceful fallback |
| 14 | External Evolver integration | EvoMap/Evolver adapter, proposal flow |

**Use this document when:**

- Implementing a new feature or wave
- Designing HTTP API responses
- Setting performance targets
- Questioning module responsibility boundaries
- Reviewing authority enforcement

---

## ADR Format

All ADRs follow this structure:

```
# ADR-NNN: Title

**Date:** YYYY-MM-DD  
**Status:** [Proposed | Accepted | Deprecated]  
**Decision:** [Brief summary]

## Context

[Problem or design question]

## Decision

[What we decided and why]

## Consequences

[Positive and negative outcomes]

## Alternatives Considered

[Other options and why we rejected them]
```

---

## Active ADRs

| ADR | Status | Topic |
|-----|--------|-------|
| 0001 | Proposed | MemoryOS vNext complete architecture; reconciled by ADR-0002 |
| 0002 | Accepted | Laputa cognitive partition: Frozen Core, STM, `MEMRULES.MD`, `WORLD.MD`, human reports, removed LTM |
| 0003 | Accepted | Operations Console design: workbench-first admin UI, governance graph, recall trace, materials/evidence, architecture library, i18n, MVP-0 read-only first |

---

## When to Write New ADRs

- New module boundaries are proposed
- Authority rules are reconsidered
- HTTP contract changes are needed
- Performance targets are adjusted
- Degradation modes are added

---

## When NOT to Write ADRs

- Implementation details (use module AGENTS.md instead)
- Temporary debugging or feature branches
- Archive material (belongs in docs/archive/)
- Test or fixture documentation

---

## Design Principles

From the active architecture:

1. **Governed MemoryOS:** Laputa holds authority; Garden orchestrates; Mentle stores
2. **Progressive Recall:** Fast (deterministic) is default; Deep (explicit) is upgrade
3. **No silent mutations:** All durable changes are explicit and audited
4. **Graceful degradation:** System works when backends are unavailable
5. **Immutable boundaries:** Module ownership is fixed; no silent crossing

---

## Navigation

**For implementation details:**

1. Read `0001-memoryos-vnext-architecture.md` for the complete specification
2. Reference `../AGENTS.md` for documentation structure
3. Check module `AGENTS.md` files for implementation approach

**For historical context:**

- See `../archive/` for pre-vNext decisions (archived, not modified)

---

## Conventions

- Markdown format, GitHub-flavored
- Code blocks with language specified for syntax highlighting
- Relative links for internal cross-references
- ISO 8601 dates (YYYY-MM-DD)
- Status: Proposed, Accepted, Deprecated (for ADRs)

---

## MANUAL

When updating:

1. Keep 0001-memoryos-vnext-architecture.md as the single source of truth
2. Link to this file from module AGENTS.md
3. Do not duplicate architectural decisions
4. Update when:
   - New ADR is written (add to Active ADRs table)
   - Architecture wave is completed
   - Performance targets are revised
   - Authority rules are clarified
5. Do not update for:
   - Implementation details (use module AGENTS.md)
   - Temporary materials

Parent reference: ../AGENTS.md
