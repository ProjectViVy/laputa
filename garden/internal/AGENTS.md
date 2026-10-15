# garden/internal — Runtime Packages

> **2026-10-11 policy supersession:** WORLD remains tool-only. ACTMEM context/schema/tool assumptions below are the baseline; selected VIVY subject-Recap policy overrides only those assumptions. Preserve existing Frozen Core immutability and Mission/Persona authorization. See [ADR-0018](../../docs/architecture/0018-vivy-unified-context-and-recap-policy.md). Original design, interface and verification records below remain evidence of their recorded snapshots; this notice does not change their pass/fail results or claim the new policy is implemented.

**Current contract:** [`../../docs/architecture/0012-laputa-markdown-clean-break.md`](../../docs/architecture/0012-laputa-markdown-clean-break.md)

The internal packages implement Garden's runtime responsibilities: activity events, raw-first ingestion, bounded recall, evidence, console aggregation, EvoMap integration, and HTTP transport.

## Package Boundaries

| Package area | Responsibility | Must not do |
| --- | --- | --- |
| `activity`, `lifecycle` | Session events, checkpoints, restart-safe runtime work | Claim to be `ACTMEM.MD` or replace it |
| `recall`, `rag` | Cards, bounded evidence selection, disposable ContextView and trace | Automatically include `WORLD.MD` or admit ACTMEM outside the selected policy |
| `authority` | Temporary migration seam for reading the new Persona/Frozen Core contract | Reintroduce JSON sections, generic Persona mutation, or an old authority mapper |
| `evolution`, `mailbox` | EvoMap candidates/proposals, evaluation, mailbox, privacy and Hub policy | Write Persona, ACTMEM, Mentle authority, or directly create/install Skills outside EvoMap |
| `ingest`, `report` | Material ingestion and human-readable continuity artifacts | Become a new Persona or long-term authority store |
| `server` | Versioned HTTP contracts and explicit tool surfaces | Hide a legacy fallback behind v2 handlers |

## Required Rules

- Authority bodies are Markdown documents, never `map[string]any` or JSON Patch.
- Personality input preserves existing session-frozen Persona projections and human-controlled Mission v2 semantics; Recap is not a new Frozen Core slot.
- `WORLD.MD` remains explicit and absent from automatic context. ACTMEM is admitted only under the selected VIVY policy using existing context/runtime/Journal abstraction, never a second LIFE pipeline.
- EvoMap alone owns capability artifact lifecycle.
- ContextView remains a temporary response; it is never a new authority or memory store.

Existing code still implements parts of the retired architecture. Do not extend those contracts. Replace them under the clean-break plan and enforce removal through tests and repository scans.
