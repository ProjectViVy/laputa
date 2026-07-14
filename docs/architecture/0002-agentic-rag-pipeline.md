# ADR 0002: Governed Pipeline Runtime and Agentic RAG

**Status:** implemented  
**Date:** 2026-07-14

## Decision

Garden owns orchestration, Laputa owns governance policy, and Mentle owns the
memory data plane. Garden imports only Mentle's public `facade` package. A
read-only `agentic_recall_v1` pipeline automatically plans and executes hybrid
memory, knowledge-graph, and timeline retrieval, then returns cited context.

Pipeline steps declare required and produced state, capabilities, timeout,
idempotency, and allowed transitions. Runtime configuration is versioned YAML
loaded at startup. Pipeline status and redacted run traces are read-only HTTP
resources; configuration cannot be mutated over HTTP.

## Safety and degradation

Laputa denies are applied immediately after retrieval and again before output,
so forbidden candidates never reach an external model. Model prompts and raw
memory are not logged. An unavailable planner falls back to deterministic
retrieval; an unavailable Mentle falls back to governance-only context. Context
resolution never writes Laputa or Mentle state.

## Public surface

- `POST /v1/context/resolve`
- `GET /v1/pipelines`
- `GET /v1/pipelines/{name}`
- `GET /v1/pipelines/{name}/runs`
- `GET /v1/pipelines/{name}/runs/{trace_id}`
