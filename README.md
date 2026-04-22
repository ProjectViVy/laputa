# Laputa

> A standalone local memory system focused on temporal recall, continuity, and long-horizon organization.

## What It Is

Laputa is a Rust-based personal memory engine. It keeps diary-style input, temporal retrieval, wakeup context generation, semantic search, heat-driven lifecycle management, and archive/export capabilities in a single repository.

This repository can be built and run on its own. Historical influences such as `mempalace-rs`, `agent-diva`, `UPSP`, and `LifeBook` are design lineage only, not checkout prerequisites.

## Core Direction

- Time-first memory retrieval instead of semantic retrieval alone
- Continuous long-term memory organization instead of single-session context stuffing
- Full lifecycle handling: capture, emotion tagging, heat decay, rhythm organization, archival

## Standalone Quick Start

The minimum standalone startup path is:

```bash
cargo build
cargo test
cargo run -- init
```

Common follow-up commands:

```bash
# Write a diary entry
cargo run -- diary "Today I went to the convenience store with Yudie Sakura..."

# Generate wakeup context
cargo run -- wakeup

# Search stored memory
cargo run -- search "convenience store"
```

## Current Scope

- Identity initialization and local memory storage
- Diary ingestion and memory record lifecycle
- Timeline and semantic search
- Hybrid recall and wakeup context generation
- Heat service, rhythm scheduler, weekly capsules, and archive/export flows
- CLI and MCP-facing integrations implemented inside this repository

## Historical Lineage

Laputa evolved from prior experiments and upstream work. That lineage matters for architecture decisions, but it is not part of the runtime contract for this repository.

- `mempalace-rs`: implementation baseline for the original Rust memory engine
- `UPSP`: source of selected conceptual ideas such as resonance-oriented modeling
- `LifeBook`: inspiration for time-flow review and long-horizon organization
- `agent-diva`: future integration direction, not a required sibling checkout

See [STATUS.md](./STATUS.md) for tracked upstream provenance and [DECISIONS.md](./DECISIONS.md) for project-level architecture decisions.

## Repository Notes

- Project conventions for agents and contributors: [AGENTS.md](./AGENTS.md)
- Architecture and decision log: [DECISIONS.md](./DECISIONS.md)

## License

MIT
