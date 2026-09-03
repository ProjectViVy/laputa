# garden/internal — Runtime Packages

**Current contract:** [`../../docs/architecture/0012-laputa-markdown-clean-break.md`](../../docs/architecture/0012-laputa-markdown-clean-break.md)

The internal packages implement Garden's runtime responsibilities: activity events, raw-first ingestion, bounded recall, evidence, console aggregation, EvoMap integration, and HTTP transport.

## Package Boundaries

| Package area | Responsibility | Must not do |
| --- | --- | --- |
| `activity`, `lifecycle` | Session events, checkpoints, restart-safe runtime work | Claim to be `ACTMEM.MD` or replace it |
| `recall`, `rag` | Cards, bounded evidence selection, disposable ContextView and trace | Automatically include `WORLD.MD` or `ACTMEM.MD` |
| `authority` | Temporary migration seam for reading the new Persona/Frozen Core contract | Reintroduce JSON sections, generic Persona mutation, or an old authority mapper |
| `evolution`, `mailbox` | EvoMap candidates/proposals, evaluation, mailbox, privacy and Hub policy | Write Persona, ACTMEM, Mentle authority, or directly create/install Skills outside EvoMap |
| `ingest`, `report` | Material ingestion and human-readable continuity artifacts | Become a new Persona or long-term authority store |
| `server` | Versioned HTTP contracts and explicit tool surfaces | Hide a legacy fallback behind v2 handlers |

## Required Rules

- Authority bodies are Markdown documents, never `map[string]any` or JSON Patch.
- The only automatic personality input is the bounded session-frozen six-document Frozen Core.
- `WORLD.MD` and `ACTMEM.MD` are explicit tool reads. No planner, recall, bootstrap, or ContextView path may project them automatically.
- EvoMap alone owns capability artifact lifecycle.
- ContextView remains a temporary response; it is never a new authority or memory store.

Existing code still implements parts of the retired architecture. Do not extend those contracts. Replace them under the clean-break plan and enforce removal through tests and repository scans.
