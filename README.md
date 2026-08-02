# LAPUTA

[中文文档](README_CN.md)

A governed memory operating system for continuous AI agents. LAPUTA connects personal work materials to recalled context and reusable capability — without treating storage, retrieval, or evolution output as authority by themselves.

## Core Idea

```text
MemoryOS = a memory operating system centered on agent identity and governed memory,
           capable of understanding, locating, and invoking all personal work information.
```

Three independent Go modules enforce strict ownership boundaries:

| Module | Responsibility |
|--------|---------------|
| **Laputa** | Identity, authority, lifecycle, policy, audit |
| **Mentle** | Canonical material, evidence, retrieval, taxonomy, knowledge graph |
| **Garden** | Source ingestion, recall orchestration, ContextView assembly, HTTP gateway |

No module holds authority over the others. Each degrades gracefully.

## Key Design Decisions

- **Progressive Recall** — Fast Recall (default): zero LLM, deterministic, low-latency, cacheable. Deep Recall (explicit upgrade): independent budget, KG/timeline/graph expansion, full trace.
- **Candidate ≠ Evidence ≠ ContextView** — discovery, bounded evidence read, and final assembly are separate stages with separate budgets.
- **No silent high-impact mutation** — authority changes, skill approval, host installation, and physical deletion are always explicit and audited.
- **Governed evolution** — external Evolver proposes capability; only Laputa approves and applies authority.

## Architecture

```text
┌─────────────────────────────────────────────────────┐
│  Host Adapters (Hermes / Claude Code / Codex)       │
└──────────────────────┬──────────────────────────────┘
                       │ HTTP
┌──────────────────────▼──────────────────────────────┐
│  Garden — orchestration gateway                     │
│  /v2/recall/fast · /v2/recall/deep                  │
│  /v2/activity/*  · /v2/governance/*                 │
│  /v2/evolution/* · /v1/* (compat)                   │
└───────┬─────────────────────────────┬───────────────┘
        │                             │
┌───────▼────────┐          ┌─────────▼──────────────┐
│  Laputa        │          │  Mentle                │
│  governance    │          │  material + retrieval  │
│  authority     │          │  evidence + graph      │
│  audit         │          │  hybrid search (HNSW)  │
└────────────────┘          └────────────────────────┘
```

## Quick Start

```bash
# Prerequisites: Go 1.26+, CGO enabled (for SQLite)

# Build all modules
cd laputa  && go build ./...
cd ../mentle && go build ./...
cd ../garden && go build -o garden.exe .

# Run the server (default: http://127.0.0.1:7373)
./garden.exe

# Health check
curl -s http://127.0.0.1:7373/health
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `GARDEN_PIPELINE_CONFIG` | Path to pipelines.yaml | `~/.garden/pipelines.yaml` |
| `GARDEN_RAG_BASE_URL` | OpenAI-compatible LLM endpoint | _(disabled)_ |
| `GARDEN_RAG_API_KEY` | API key for LLM planner | _(disabled)_ |
| `GARDEN_RAG_MODEL` | Model name for planner | _(disabled)_ |

Without LLM environment variables, Garden uses a deterministic planner and reports degradation without failing.

## Testing

```bash
cd laputa  && GOSUMDB=off go test ./governance/...
cd ../mentle && GOSUMDB=off go test ./facade/...
cd ../garden && GOSUMDB=off go test ./internal/...
GOSUMDB=off go test -tags=e2e ./e2e/...
```

## Repository Layout

```text
laputa/    Go governance module — authority, identity, lifecycle, audit
mentle/    Go material & retrieval module — canonical catalog, evidence, hybrid search, graph
garden/    Go application module — HTTP gateway, recall, activity orchestration
docs/      Architecture decisions, migration plans, historical archive
```

## Performance Targets

| Operation | Target |
|-----------|--------|
| Governance projection (warm) | P95 ≤ 5 ms |
| SearchCards | P95 ≤ 80 ms |
| Filter / rank / dedupe | P95 ≤ 10 ms |
| Bounded ReadEvidence | P95 ≤ 40 ms |
| Fast Recall total | P95 ≤ 150 ms |
| Governance-only degradation | P95 ≤ 30 ms |

## Documentation

- [Architecture Plan (vNext)](docs/architecture/0001-memoryos-vnext-architecture.md)
- [ADR-0002: Cognitive Partition](docs/architecture/0002-laputa-cognitive-partition-decision.md)
- [ADR-0003: Operations Console](docs/architecture/0003-operations-console-design.md)
- [Documentation Index](docs/README.md)
- [Historical Archive](docs/archive/2026-08-01-pre-memoryos-redesign/)

## References & Inspiration

- [MemGPT / Letta](https://github.com/letta-ai/letta) — LLM memory management with virtual context paging
- [Mem0](https://github.com/mem0ai/mem0) — memory layer for AI agents
- [Zep](https://github.com/getzep/zep) — long-term memory service for AI assistants
- [LangChain Memory](https://github.com/langchain-ai/langchain) — composable memory modules for LLM applications
- [LlamaIndex](https://github.com/run-llama/llama_index) — data framework for LLM-based retrieval
- [Cognee](https://github.com/topoteretes/cognee) — memory management for AI agents using knowledge graphs
- [HNSW (govector)](https://github.com/DotNetAge/govector) — HNSW vector index used in Mentle
- [Eino (CloudWeGo)](https://github.com/cloudwego/eino) — LLM orchestration framework used in Laputa

## License

[MIT](LICENSE)
