<!-- Parent: ../AGENTS.md -->

# laputa/.hermes — Hermes Planning Artifacts

**Generated:** 2026-08-01  
**Purpose:** Runtime planning and autodream state for Laputa agent

---

## Purpose

The `.hermes/` directory stores Hermes-driven planning artifacts and automated scheduling outputs for Laputa:

- **plans/** — Autodream plan files with step-by-step task sequences
- **Runtime state** — Hermes scheduler integration (if active)

---

## Structure

```
.hermes/
└── plans/
    └── 2026-06-30_010300-laputa-eino-rhythm-autodream.md
```

### Subdirectories (Depth 2)

- **[plans/AGENTS.md](./plans/AGENTS.md)** — autodream plan index and execution tracking

---

## Key Concepts

### Autodream Plans

Hermes-generated plan files follow naming convention:

```
{YYYY-MM-DD}_{HHMMSS}-{module}-{feature}-autodream.md
```

Each plan is a markdown task list with:
- Goal statement
- Step-by-step tasks (numbered)
- Test/validation steps
- Risk and tradeoff analysis
- Open questions

---

## Current Plans

| Plan | Date | Feature | Status |
|------|------|---------|--------|
| laputa-eino-rhythm-autodream | 2026-06-30 | Rhythm engine + Eino integration | reference |

---

## Build & Test

No build artifacts. Plans are documentation only.

---

## Architecture Boundaries

**Ownership rule (immutable):**

```text
Hermes may generate plans.
Hermes may not execute without governance approval.
Laputa may execute approved plans.
```

---

## Conventions

- **Plan naming:** `{date}-{module}-{feature}-autodream.md`
- **Task format:** numbered steps, each with success criteria
- **Version control:** plans are committed for audit trail
- **Archival:** completed or superseded plans move to `docs/archive/`

---

## MANUAL

This document is maintained by the oh-my-claudecode writer agent.

When updating:

1. Link to plans/ subdirectory for plan details
2. Update when new plan files are added
3. Do not duplicate plan content

Parent reference: `../AGENTS.md`
