<!-- Parent: ../AGENTS.md -->

# laputa/cmd — CLI Entry Points & Smoke Tests

**Generated:** 2026-08-01  
**Purpose:** Command-line tools and quick-start executables for Laputa

---

## Purpose

The `cmd/` directory contains entry points for Laputa utilities:

- **CLI commands** for governance operations (if any)
- **Smoke tests** to verify core functionality
- **Setup and initialization** tools
- **Development utilities** for local testing

---

## Structure

```
cmd/
└── eino_smoke/                    # Smoke test for eino LLM orchestration
    ├── main.go                    # Entry point
    └── (supporting files as needed)
```

---

## eino_smoke

Quick smoke test for Laputa's Rhythm reporting engine integration with eino (LLM orchestration framework).

### Purpose

- Verify eino + OpenAI integration works
- Test ReportResult generation (mock and real paths)
- Validate Rhythm kinds (daily, weekly, monthly)

### Build

```bash
cd laputa/cmd/eino_smoke
go build -o eino_smoke .
```

### Run (Mock Path)

No API key needed:

```bash
./eino_smoke -kind daily
```

### Run (Real LLM)

Requires OpenAI API key:

```bash
export OPENAI_API_KEY=sk-...
./eino_smoke -kind daily
```

### Output

Produces ReportResult JSON with:

- Title
- Summary
- Highlights (list)
- OpenQuestions (optional)
- GeneratedAt timestamp

---

## Future Commands

Potential CLI commands (not yet implemented):

- `laputa init` — initialize governance sections
- `laputa backup` — snapshot all sections to a tarball
- `laputa restore` — restore from backup
- `laputa audit` — view mutation history

---

## Conventions

- Go CLI using `flag` or `cobra` (if added)
- Entry point: `main()` in `main.go`
- Error handling: explicit, exit codes reflect status
- Output: JSON or text to stdout; errors to stderr
- No interactive prompts (use flags/env vars instead)

---

## MANUAL

When adding new commands:

1. Create a subdirectory under `cmd/`
2. Include `main.go` as entry point
3. Document purpose and usage here
4. Add build/run examples

Parent reference: ../AGENTS.md
