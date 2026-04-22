# AGENTS.md - Laputa Project Guide

> Project-level guidance for AI agents and human contributors working in the Laputa repository.
> Last updated: 2026-04-21

## Project Identity

- Project: Laputa
- Positioning: a standalone Rust memory system for long-term continuity, temporal recall, and lifecycle-aware organization
- Core idea: memory should support judgment, forgetting, emotion tagging, rhythm, and archival rather than only retrieval

## Repository Contract

- Treat this repository as self-contained for build, test, and run flows.
- Do not assume sibling repositories exist on disk.
- Do not instruct users to clone `mempalace-rs`, `agent-diva`, `UPSP`, or `LifeBook` to satisfy Laputa runtime prerequisites.
- When referring to prior projects, label them as historical lineage, conceptual source, or migration context.

## Minimum Verification Path

Use these commands as the baseline standalone smoke path:

```bash
cargo build
cargo test
cargo run -- init
```

## Architectural Direction

- Time-first memory flows remain a primary retrieval axis.
- Semantic search complements, not replaces, temporal recall.
- Heat-driven lifecycle handling and rhythm organization are first-class project concerns.
- Archive and export behavior should remain local-repo capabilities.

## Design Decisions To Preserve

- Keep Laputa portable across host applications and deployment contexts.
- Prefer extending existing repository modules over introducing cross-repo coupling.
- Preserve emotion coding, recall, rhythm, and lifecycle behaviors already implemented in this repository.
- Keep public CLI, MCP, and storage behaviors documented and testable within this repository.

## Historical Sources

These are references for lineage and design context only:

- `mempalace-rs`: original Rust implementation baseline
- `UPSP`: conceptual source for selected resonance-style ideas
- `LifeBook`: inspiration for longitudinal review and time-flow thinking
- `agent-diva`: future integration direction

If a migration note or architectural explanation needs to mention those projects, make it explicit that the reference is historical and not a required filesystem layout.

## Implementation Guidance

- Favor changes that preserve standalone repository operation.
- Update repository-facing docs when build, test, initialization, or migration expectations change.
- Add or update tests for any repository contract that can regress, especially standalone build assumptions and metadata/docs guarantees.
- Avoid introducing parent-directory paths or sibling-repo assumptions into manifests, scripts, tests, or user-facing docs.
