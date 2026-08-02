<!-- Parent: ../AGENTS.md -->

# garden/config — Pipeline Configuration

**Generated:** 2026-08-01  
**Purpose:** Runtime pipeline definitions and example configurations

---

## Purpose

The `config/` directory contains YAML pipeline definitions and configuration examples for Garden's HTTP server. These files define:

- **Pipeline execution sequences** for recall, activity ingestion, and governance
- **Example configurations** for quick-start deployments
- **Environment variable mappings** for runtime behavior customization

---

## Structure

```
config/
├── config.example.yaml       # Example pipeline configuration with all available options
├── pipelines.yaml            # Active runtime pipeline definitions (environment-specific)
└── (other deployment configs as needed)
```

---

## Key Files

### config.example.yaml

Template showing all available pipeline options:

- Fast Recall pipeline: deterministic planner, no LLM
- Deep Recall pipeline: explicit LLM planner, KG/timeline integration
- Activity ingestion pipeline: session lifecycle, event normalization
- Degradation fallbacks: behavior when Mentle or LLM is unavailable

Use as reference when setting up new deployments.

### pipelines.yaml

Active runtime configuration. Specifies:

- Enabled pipelines (fast_recall, deep_recall, activity)
- Planner type (deterministic or openai)
- LLM endpoint and model name (if external planner)
- Timeout and retry policies

---

## Usage

Load at server startup:

```bash
export GARDEN_PIPELINE_CONFIG=./config/pipelines.yaml
./garden.exe
```

Without `GARDEN_PIPELINE_CONFIG`, Garden uses built-in defaults with deterministic planner only.

---

## Conventions

- YAML format, 2-space indentation
- Comments document non-obvious settings
- Example values use safe defaults (no API keys in repo)
- Environment variables override file values

---

## MANUAL

When updating:

1. Keep example configs in sync with actual schema
2. Document new pipeline options in config.example.yaml
3. Do not commit sensitive values (.env, API keys)
4. Reference in deployment docs

Parent reference: `../AGENTS.md`
