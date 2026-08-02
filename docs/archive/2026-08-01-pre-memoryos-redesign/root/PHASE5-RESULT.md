# Phase 5 Result — Governed Pipeline and Agentic RAG

**Date:** 2026-07-14  
**Status:** complete

## Delivered

- Garden Pipeline Runtime with step contracts, capability gates, timeouts,
  idempotent retry, bounded transitions, configuration revision, and redacted
  in-memory run history.
- `agentic_recall_v1`: Laputa policy loading, rule + optional LLM planning,
  Mentle hybrid retrieval, KG/timeline expansion, immediate and final policy
  filtering, deduplication, reranking, citation validation, and ContextPackage
  assembly.
- `POST /v1/context/resolve` plus read-only Pipeline status and run APIs.
- Real Mentle facade CRUD and public retrieval/KG/timeline DTOs. Hybrid results
  retain RRF score and vector/BM25 channel provenance.
- Backward-compatible Laputa `agentic_rag` policy fields for new sections.
- Deterministic planner and governance-only degradation paths.

## Verification

- `laputa: go test ./governance/...` — passed.
- `mentle: go test ./facade/... ./internal/hybrid/... ./internal/search/...` — passed.
- `garden: go test ./...` — passed.
- `garden: go test -tags=e2e ./e2e/...` — passed with the real executable.
- External OpenAI-compatible planner E2E using environment values sourced from
  desktop `KEYS.TXT` — passed; no secret value was printed or persisted.

Mentle's unrelated full-suite tests still contain environment-sensitive
failures in model download locking and a Windows invalid-path assertion. The
changed Mentle packages and all Garden/Laputa integration paths pass.

## Runtime contract

- Pipeline configuration: `GARDEN_PIPELINE_CONFIG`, default
  `~/.garden/pipelines.yaml`, built-in defaults when absent.
- Planner configuration: `GARDEN_RAG_BASE_URL`, `GARDEN_RAG_API_KEY`,
  `GARDEN_RAG_MODEL`.
- Resolution is read-only. Forbidden sources/wings/rooms are filtered before
  any refinement or summarization call and again before output.
