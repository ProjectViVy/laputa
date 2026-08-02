<!-- Parent: ../../AGENTS.md -->

# mentle/storage/redis — Redis Cache and Async Indexing Backend

**Generated:** 2026-08-01  
**Purpose:** Optional Redis backend for caching and asynchronous indexing operations

---

## Purpose

The `redis/` package provides optional performance enhancement:

- **Query result caching** — cache hot searches
- **Async indexing** — background vector embedding jobs
- **Session state** — temporary session-specific data
- **Rate limiting** — distribute load

---

## Structure

```
redis/
├── cache.go                         # Caching layer
├── indexing.go                      # Async indexing operations
├── cache_test.go                    # Test suite
└── (supporting utilities)
```

---

## Configuration

Via environment variables:

```bash
MENTLE_REDIS_ADDR=localhost:6379
MENTLE_REDIS_DB=0
MENTLE_REDIS_PASSWORD=
```

If not configured, Redis operations are skipped gracefully.

---

## Key Operations

```go
cache, err := redis.NewCache(addr)

// Cache a search result
err := cache.CacheSearch(ctx, query, results, ttl)

// Get cached result
results, found, err := cache.GetSearch(ctx, query)

// Queue indexing job
jobID, err := cache.QueueIndexing(ctx, cardID, embedding)
```

---

## Build & Test

```bash
cd mentle
GOSUMDB=off go test ./storage/redis/...
```

---

## MANUAL

Redis is optional. System remains functional without it (with performance degradation). Document fallback behavior.

Parent reference: ../AGENTS.md
