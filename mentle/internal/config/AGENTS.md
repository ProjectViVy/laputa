<!-- Parent: ../../AGENTS.md -->

# mentle/internal/config — Configuration Management

**Generated:** 2026-08-01  
**Purpose:** Runtime configuration and sensible defaults

---

## Purpose

The `config/` package provides configuration management:

- **Environment variable parsing** — read from os.Getenv
- **YAML/JSON configuration** — structured config files
- **Defaults** — sensible fallbacks for all options
- **Validation** — type checking and bounds enforcement
- **Hot reload** — optional configuration refresh

---

## Structure

```
config/
├── config.go         # Config loading and parsing
└── config_test.go
```

---

## Key Concepts

### Configuration Schema

```go
type Config struct {
    Search SearchConfig
    Mining MiningConfig
    Palace PalaceConfig
    Storage StorageConfig
}

type SearchConfig struct {
    BM25K1              float64 // default: 1.5
    BM25B               float64 // default: 0.75
    MaxResults          int     // default: 100
    TermExpandTimeout   time.Duration
}
```

### Loading Priority

1. Environment variables (highest priority)
2. YAML file (if specified via MENTLE_CONFIG_PATH)
3. Built-in defaults (lowest priority)

### Environment Variables

- `MENTLE_SEARCH_MAX_RESULTS` — max search results
- `MENTLE_MINING_TIMEOUT` — mining operation timeout
- `MENTLE_PALACE_PATH` — palace storage directory

---

## Testing

```bash
cd mentle
GOSUMDB=off go test -v ./internal/config/...
```

**Behavioral tests:**

- Environment variables override defaults
- YAML parses correctly
- Validation rejects invalid values
- Defaults are sensible

---

## Conventions

- All timeouts are positive durations
- All limits are positive integers
- Configuration is read-only after load
- No mutation during runtime

---

## MANUAL

Keep config focused on loading and validation. Business logic goes elsewhere.

Parent reference: ../AGENTS.md
