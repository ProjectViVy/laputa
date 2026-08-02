<!-- Parent: ../../../AGENTS.md -->

# mentle/docs/superpowers/specs — Technical Specifications

**Generated:** 2026-08-01  
**Purpose:** Detailed technical specifications for approved superpowers

---

## Purpose

Specification documents provide detailed technical design for approved features:

- **API contracts** — request/response formats
- **Data structures** — Go types and database schema
- **Algorithms** — step-by-step procedures
- **Edge cases** — failure modes and recovery
- **Performance** — target latency and throughput

---

## Structure

Each specification should include:

1. **Title** — superpower name
2. **Status** — draft/approved/implementing/done
3. **Overview** — 1-paragraph summary
4. **Motivation** — why is this important?
5. **Design** — detailed approach
6. **API** — request/response examples
7. **Schema** — data structure changes
8. **Testing** — test strategy and cases
9. **Rollout** — deployment and migration plan

---

## Creating a Spec

```markdown
# Specification: [Name]

## Status

draft / approved / implementing / done

## Overview

[One paragraph]

## Motivation

[Why this matters]

## Design

[Detailed approach]

### Algorithm

[Step-by-step procedure]

### Data Structures

[Go types and schema]

## API

### Request

\`\`\`go
type Request struct { ... }
\`\`\`

### Response

\`\`\`go
type Response struct { ... }
\`\`\`

## Testing

[Test strategy]

## Rollout

[Deployment plan]
```

---

## MANUAL

Specs should be detailed enough for implementation (2-5 pages typical).

Parent reference: ../../AGENTS.md
