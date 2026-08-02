<!-- Parent: ../AGENTS.md -->

# mentle/docs — Internal Documentation and Design Specs

**Generated:** 2026-08-01  
**Purpose:** Design documents, algorithm specs, and superpowers roadmap

---

## Purpose

The `docs/` directory contains internal design documentation and specifications:

- **Algorithm design** — hybrid search, BM25, vector retrieval strategies
- **Superpowers roadmap** — planned enhancements and research directions
- **Integration guides** — how to use Mentle as a library or MCP server
- **Architecture notes** — design decisions and trade-offs

---

## Structure

```
docs/
├── hybrid-search.md                 # Hybrid search algorithm design
├── superpowers/                     # Future capabilities roadmap
│   ├── plans/                       # High-level planning documents
│   │   └── (design docs)
│   └── specs/                       # Detailed specifications
│       └── (technical specs)
└── (additional docs as needed)
```

### Subdirectories (Depth 2)

- **[superpowers/AGENTS.md](./superpowers/AGENTS.md)** — roadmap planning and technical specifications

---

## Key Documents

### hybrid-search.md

Comprehensive design of Mentle's hybrid search approach:

- **Semantic search** — vector similarity (cosine distance)
- **Lexical search** — BM25 ranking
- **Temporal filtering** — knowledge graph validity constraints
- **Fusion strategy** — combining scores from multiple rankers
- **Performance characteristics** — benchmarks and trade-offs

Topics covered:
- Algorithm overview
- Score normalization
- Rank aggregation
- Edge cases and degradation
- Performance tuning

---

## Superpowers

The `superpowers/` directory tracks planned enhancements and research directions:

- **plans/** — high-level planning documents
- **specs/** — detailed technical specifications

See [superpowers/AGENTS.md](./superpowers/AGENTS.md) for details on specific initiatives.

---

## Architecture Decisions

When documenting new architecture decisions:

1. Create a new Markdown file with clear title
2. Include sections: Problem, Approach, Trade-offs, Alternatives
3. Add date and author information
4. Link from relevant AGENTS.md files
5. Archive old decisions with date prefix

---

## Integration Guides

Documentation on integrating Mentle:

- **Library mode** — embedding Mentle in Go projects
- **MCP server mode** — running as Claude Desktop resource server
- **CLI mode** — command-line usage
- **Benchmarking** — using LongMemEval harness

---

## Conventions

- **Markdown files** — single source of truth for algorithm/design
- **Dated decisions** — prefix with YYYY-MM-DD if archiving
- **Code examples** — tested and runnable where possible
- **Links** — relative paths to code and subdirectory AGENTS.md files
- **Terminology** — consistent with parent AGENTS.md and module README

---

## Build & Test

### Verify Documentation Examples

If documentation includes code examples:

```bash
cd mentle
go test ./...  # Verify any included examples compile
```

### Check Links

Ensure all relative links in docs are valid:

```bash
find docs -name "*.md" -exec grep -l "](\..*md)" {} \;
```

---

## MANUAL

When updating documentation:

1. **Keep this file as single source of truth** for docs directory organization
2. **Link to superpowers/AGENTS.md** for roadmap details
3. **Do not duplicate** algorithm descriptions (reference hybrid-search.md instead)
4. **Update when:**
   - New design documents are added
   - Algorithm specs are revised
   - Integration guides are updated
   - Superpowers roadmap changes
5. **Do not update for:**
   - Implementation details (use subdirectory AGENTS.md instead)
   - Performance benchmarks (document in benchmarks/AGENTS.md)
   - Archive material (preserve in docs/archive/)

Parent reference: ../AGENTS.md
