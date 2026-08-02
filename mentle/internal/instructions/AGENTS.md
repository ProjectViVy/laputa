<!-- Parent: ../../AGENTS.md -->

# mentle/internal/instructions — Usage Instructions & Prompts

**Generated:** 2026-08-01  
**Purpose:** Store and manage usage instructions and LLM prompts

---

## Purpose

The `instructions/` package manages structured instructions:

- **Usage instructions** — how to use memories and features
- **LLM prompts** — templates for LLM interactions
- **Prompt engineering** — tune prompts for specific tasks
- **Versioning** — track instruction changes
- **Localization** — support multiple languages

---

## Structure

```
instructions/
├── instructions.go   # Instruction management
└── instructions_test.go
```

---

## Key Concepts

### Instruction Types

```go
const (
    TypeUsage      = "usage"
    TypePrompt     = "prompt"
    TypeSystemPrompt = "system_prompt"
)
```

### Instruction

```go
type Instruction struct {
    ID       string
    Type     string
    Name     string
    Content  string
    Version  int
    Tags     []string
    Language string   // "en", "zh", etc.
}
```

### Usage

**Store(ctx, instruction):**
- Store instruction
- Track version

**Get(ctx, name):**
- Retrieve instruction
- Return latest version

**List(ctx, tag):**
- List instructions with tag
- Return all versions

---

## Testing

```bash
cd mentle
GOSUMDB=off go test -v ./internal/instructions/...
```

**Behavioral tests:**

- Instructions stored correctly
- Versioning works
- Retrieval returns latest
- Tags filter correctly

---

## Conventions

- All content is text
- Version starts at 1
- Language defaults to "en"

---

## MANUAL

Keep instructions focused on storage. Prompt execution goes elsewhere.

Parent reference: ../AGENTS.md
