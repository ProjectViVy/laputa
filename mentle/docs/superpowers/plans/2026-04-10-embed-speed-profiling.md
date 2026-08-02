# Embed Speed: Profile-then-Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Find the real bottleneck inside Go's hugot `RunPipeline` call (using pprof), then implement the appropriate speed-up so LongMemEval embed time matches or beats Python/ChromaDB (~3x faster than current baseline).

**Architecture:** We profile first with `go test -cpuprofile`, identify the top frames, then apply the fix (likely: concurrent embedding workers, larger chunk sizes, or cross-question session caching). ORT is deprioritised — pure-Go (GoMLX) is the target backend.

**Tech Stack:** Go 1.21+, `github.com/knights-analytics/hugot`, `runtime/pprof`, `go tool pprof`, `sync.WaitGroup` / `errgroup` for concurrency if that path is taken.

---

## File Map

| File | Action | Role |
|------|--------|------|
| `internal/embedder/hugot.go` | Modify | `chunkSize` constant lives here; concurrency changes go here if needed |
| `internal/embedder/hugot_test.go` | Modify | Add `BenchmarkCorpusEmbed` profiling benchmark |
| `benchmarks/longmemeval/longmemeval.go` | Modify | Add cross-question cache and/or worker-pool call site |
| `benchmarks/longmemeval/cache.go` | Create (if cache path chosen) | `precomputeCache` function |
| `benchmarks/longmemeval/worker.go` | Create (if concurrent path chosen) | Worker-pool `embedCorpusConcurrent` |

---

## Task 1: Commit the in-flight chunkSize=64 changes

These changes are already written and tested; they just need to land.

**Files:**
- Modify: `internal/embedder/hugot.go` (already changed, `chunkSize` 32→64)
- Modify: `internal/embedder/hugot_test.go` (already has `TestCreateEmbeddings_LargeBatch`)

- [ ] **Step 1: Verify tests pass**

```bash
go test ./internal/embedder/ -v -run TestCreateEmbeddings_LargeBatch
```

Expected output ends with `PASS`.

- [ ] **Step 2: Commit**

```bash
git add internal/embedder/hugot.go internal/embedder/hugot_test.go
git commit -m "feat(embedder): increase chunk size 32→64 and add large-batch test"
```

---

## Task 2: Add a realistic-corpus profiling benchmark

Add a benchmark that mimics the real LongMemEval workload: ~50 texts, each ~80 words (typical session length). This is what we'll profile.

**Files:**
- Modify: `internal/embedder/hugot_test.go`

- [ ] **Step 1: Write the benchmark**

Add this function to the *bottom* of `internal/embedder/hugot_test.go`:

```go
// BenchmarkCorpusEmbed mirrors the real LongMemEval workload: one question's
// worth of corpus (~50 sessions, each ~80 words). Profile with:
//
//	go test -bench=BenchmarkCorpusEmbed -benchtime=5x -cpuprofile=cpu.prof ./internal/embedder/
//	go tool pprof -http=:6060 cpu.prof
func BenchmarkCorpusEmbed(b *testing.B) {
	emb, err := New("", "")
	if err != nil {
		b.Fatalf("create embedder: %v", err)
	}
	defer emb.Close()

	ctx := context.Background()

	// Build 50 texts that are each ~80 words — realistic session length.
	word := "Lorem ipsum dolor sit amet consectetur adipiscing elit sed do eiusmod"
	single := ""
	for i := 0; i < 80; i++ {
		if i > 0 {
			single += " "
		}
		single += word[i%len(word) : i%len(word)+1]
	}
	// Use a realistic 80-word sentence instead of the char-by-char above.
	single = "the quick brown fox jumped over the lazy dog near the river bank on a warm summer afternoon while birds sang in the trees and children played nearby along the winding path through the ancient forest full of tall oaks and whispering pines under a clear blue sky"
	texts := make([]string, 50)
	for i := range texts {
		texts[i] = single
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := emb.CreateEmbeddings(ctx, texts)
		if err != nil {
			b.Fatalf("corpus embed: %v", err)
		}
	}
}
```

- [ ] **Step 2: Verify the benchmark compiles and runs one iteration**

```bash
go test -bench=BenchmarkCorpusEmbed -benchtime=1x ./internal/embedder/
```

Expected: one iteration completes without error, prints `ns/op` line.

- [ ] **Step 3: Commit**

```bash
git add internal/embedder/hugot_test.go
git commit -m "test(embedder): add BenchmarkCorpusEmbed for profiling realistic workload"
```

---

## Task 3: Capture a CPU profile and read the flame graph

**Files:** none (read-only analysis step)

- [ ] **Step 1: Run 5 iterations with CPU profile**

```bash
go test -bench=BenchmarkCorpusEmbed -benchtime=5x -cpuprofile=cpu.prof ./internal/embedder/
```

This writes `cpu.prof` in the current directory (repo root when run from there, or `internal/embedder/` when run from that dir). Run from the repo root:

```bash
cd /path/to/mempalace-go
go test -bench=BenchmarkCorpusEmbed -benchtime=5x \
    -cpuprofile=cpu.prof \
    github.com/argylelabcoat/mempalace-go/internal/embedder
```

Expected: `cpu.prof` file created, benchmark output printed.

- [ ] **Step 2: Open the interactive flame graph**

```bash
go tool pprof -http=:6060 cpu.prof
```

Open `http://localhost:6060/ui/flamegraph` in a browser.

- [ ] **Step 3: Identify the top hotspot**

Look for the widest bars. Key questions to answer:
- Is time dominated by `RunPipeline` internals (GoMLX graph tracing/compilation)?
- Is it dominated by tokenisation?
- Is it dominated by memory allocation (`runtime.mallocgc`)?
- Is it dominated by something else (e.g., BLAS, single-threaded matrix multiply)?

Record the top 3 frames here before continuing.

- [ ] **Step 4: Choose the optimisation path**

Based on the profile, pick **one** of the following tasks to execute next:
- If GoMLX graph tracing/JIT is the hotspot → **Task 4A** (concurrent workers)
- If tokenisation is the hotspot → **Task 4B** (pre-tokenise outside pipeline)
- If matrix multiply is single-threaded → **Task 4A** (concurrent workers, different cores)
- If memory allocation is the hotspot → **Task 4C** (cross-question session cache)

---

## Task 4A: Concurrent embedding worker pool (if concurrency is the fix)

Use multiple hugot pipeline instances (one per worker) to embed several questions' corpora in parallel. Goroutines are cheap; the limiting factor is RAM (each hugot session holds model weights ≈ 90 MB).

**Files:**
- Create: `benchmarks/longmemeval/worker.go`
- Modify: `benchmarks/longmemeval/longmemeval.go`

- [ ] **Step 1: Write a failing test for the worker pool**

Create `benchmarks/longmemeval/worker_test.go`:

```go
package longmemeval

import (
	"sync/atomic"
	"testing"
)

// TestWorkerPool_CallsWorkFnN verifies that the pool calls the work function
// exactly once for each input item, and that the results come back in the
// same order as the input.
func TestWorkerPool_CallsWorkFnN(t *testing.T) {
	inputs := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	var callCount int64

	results, err := runWorkerPool(inputs, 3, func(i int) (int, error) {
		atomic.AddInt64(&callCount, 1)
		return i * 2, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if int(callCount) != len(inputs) {
		t.Errorf("work fn called %d times, want %d", callCount, len(inputs))
	}
	for i, r := range results {
		if r != inputs[i]*2 {
			t.Errorf("results[%d] = %d, want %d", i, r, inputs[i]*2)
		}
	}
}
```

- [ ] **Step 2: Run the test to confirm it fails**

```bash
go test ./benchmarks/longmemeval/ -run TestWorkerPool_CallsWorkFnN -v
```

Expected: compile error — `runWorkerPool` undefined.

- [ ] **Step 3: Implement the worker pool**

Create `benchmarks/longmemeval/worker.go`:

```go
package longmemeval

import (
	"sync"
)

// runWorkerPool fans out work(items[i]) across up to `workers` goroutines,
// preserving order in the returned slice. If any invocation returns an error,
// the first error is returned and remaining results are zero-valued.
func runWorkerPool[T, R any](items []T, workers int, work func(T) (R, error)) ([]R, error) {
	results := make([]R, len(items))
	errs := make([]error, len(items))

	type job struct {
		idx  int
		item T
	}

	jobs := make(chan job, len(items))
	for i, item := range items {
		jobs <- job{idx: i, item: item}
	}
	close(jobs)

	var wg sync.WaitGroup
	for w := 0; w < workers && w < len(items); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				r, err := work(j.item)
				results[j.idx] = r
				errs[j.idx] = err
			}
		}()
	}
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return results, nil
}
```

- [ ] **Step 4: Run the test to confirm it passes**

```bash
go test ./benchmarks/longmemeval/ -run TestWorkerPool_CallsWorkFnN -v
```

Expected: `PASS`.

- [ ] **Step 5: Wire the worker pool into the benchmark Run loop**

In `benchmarks/longmemeval/longmemeval.go`, the `Run` function currently creates one `*embedder.Embedder` and loops serially. Replace the embedder creation and the per-entry loop with a pool approach.

The number of workers is controlled by a constant at the top of `longmemeval.go`:

```go
// embedWorkers is the number of parallel corpus-embedding goroutines.
// Each worker loads its own hugot session (~90 MB RAM). Set to 1 to disable.
const embedWorkers = 4
```

Change the `Run` function so that `buildCorpus` calls are dispatched through `runWorkerPool`. The search and scoring steps remain serial (they're fast):

Replace the single-embedder block:
```go
emb, err := embedder.New("", "")
if err != nil {
    return nil, fmt.Errorf("create embedder: %w", err)
}
defer emb.Close()
```

With a pool of embedders:
```go
embs := make([]*embedder.Embedder, embedWorkers)
for i := range embs {
    e, err := embedder.New("", "")
    if err != nil {
        // close already-created ones
        for j := 0; j < i; j++ {
            embs[j].Close()
        }
        return nil, fmt.Errorf("create embedder %d: %w", i, err)
    }
    embs[i] = e
}
defer func() {
    for _, e := range embs {
        e.Close()
    }
}()
```

Then change `buildCorpus` to accept `emb *embedder.Embedder` as a parameter (it already does — no signature change needed). Dispatch via `runWorkerPool`:

```go
type corpusResult struct {
    store     *govector.Store
    corpusIDs []string
}

corpusResults, err := runWorkerPool(entries, embedWorkers, func(entry Entry) (corpusResult, error) {
    // Each goroutine gets its own embedder from the pool.
    // We use a channel to distribute embedders across workers.
    // Simple approach: assign by goroutine index — but runWorkerPool
    // doesn't expose worker index. Use a buffered channel instead.
    panic("see note below")
})
```

**Note:** The simple `runWorkerPool` above doesn't expose a per-worker resource (the embedder). Extend `worker.go` to support a resource pool:

```go
// runWorkerPoolWithResource is like runWorkerPool but each worker gets its own
// resource of type Res (e.g., a dedicated embedder instance).
func runWorkerPoolWithResource[T, R, Res any](
    items []T,
    resources []Res,
    work func(Res, T) (R, error),
) ([]R, error) {
    results := make([]R, len(items))
    errs := make([]error, len(items))

    type job struct {
        idx  int
        item T
    }

    jobs := make(chan job, len(items))
    for i, item := range items {
        jobs <- job{idx: i, item: item}
    }
    close(jobs)

    var wg sync.WaitGroup
    for _, res := range resources {
        res := res // capture
        wg.Add(1)
        go func() {
            defer wg.Done()
            for j := range jobs {
                r, err := work(res, j.item)
                results[j.idx] = r
                errs[j.idx] = err
            }
        }()
    }
    wg.Wait()

    for _, err := range errs {
        if err != nil {
            return nil, err
        }
    }
    return results, nil
}
```

Use it in `Run`:

```go
type corpusResult struct {
    store     *govector.Store
    corpusIDs []string
}

cResults, err := runWorkerPoolWithResource(entries, embs,
    func(emb *embedder.Embedder, entry Entry) (corpusResult, error) {
        store, ids, err := buildCorpus(ctx, emb, entry, mode, encoder)
        return corpusResult{store: store, corpusIDs: ids}, err
    },
)
if err != nil {
    return nil, fmt.Errorf("parallel corpus build: %w", err)
}
```

Then iterate `cResults` serially for search + scoring (same logic as before, just read `store` and `corpusIDs` from `cResults[i]`).

- [ ] **Step 6: Update the test for `runWorkerPoolWithResource`**

Add to `benchmarks/longmemeval/worker_test.go`:

```go
func TestWorkerPoolWithResource_UsesAllResources(t *testing.T) {
    resources := []int{10, 20, 30}
    inputs := []string{"a", "b", "c", "d", "e", "f"}

    results, err := runWorkerPoolWithResource(inputs, resources, func(res int, s string) (string, error) {
        return fmt.Sprintf("%s:%d", s, res), nil
    })
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(results) != len(inputs) {
        t.Errorf("got %d results, want %d", len(results), len(inputs))
    }
    // Each result must start with the corresponding input letter.
    for i, r := range results {
        if r[0] != inputs[i][0] {
            t.Errorf("results[%d] = %q, want prefix %q", i, r, inputs[i])
        }
    }
}
```

- [ ] **Step 7: Run full test suite**

```bash
go test ./...
```

Expected: all tests pass.

- [ ] **Step 8: Run benchmark — measure improvement**

```bash
go run . bench longmemeval --limit 10 data/longmemeval_s_cleaned.json
```

Record embed times from progress lines. Compare to pre-patch baseline (~8.3s avg per question). Target: wall-clock total time reduces by ~`embedWorkers`x (diminishing returns possible if GoMLX is itself single-threaded internally).

- [ ] **Step 9: Commit**

```bash
git add benchmarks/longmemeval/worker.go benchmarks/longmemeval/worker_test.go benchmarks/longmemeval/longmemeval.go
git commit -m "feat(bench): parallel corpus embedding with per-worker embedder pool"
```

---

## Task 4C: Cross-question session cache (if memory/allocation is the fix)

If profiling shows the bottleneck is re-embedding sessions that appear in multiple questions, pre-compute all unique sessions once.

**Files:**
- Create: `benchmarks/longmemeval/cache.go`
- Modify: `benchmarks/longmemeval/longmemeval.go`

- [ ] **Step 1: Write a failing test for the cache builder**

Create `benchmarks/longmemeval/cache_test.go`:

```go
package longmemeval

import (
	"testing"
)

// TestBuildSessionCache_DeduplicatesTexts verifies that identical session texts
// produce a single cache entry (not N copies).
func TestBuildSessionCache_DeduplicatesTexts(t *testing.T) {
	entries := []Entry{
		{
			HaystackSessionIDs: []string{"s1", "s2"},
			HaystackSessions: [][]any{
				{map[string]any{"role": "user", "content": "hello world"}},
				{map[string]any{"role": "user", "content": "hello world"}}, // duplicate
			},
		},
	}

	texts := collectUniqueSessionTexts(entries, "default", nil)
	if len(texts) != 1 {
		t.Errorf("expected 1 unique text, got %d", len(texts))
	}
}
```

- [ ] **Step 2: Run the test to confirm it fails**

```bash
go test ./benchmarks/longmemeval/ -run TestBuildSessionCache_DeduplicatesTexts -v
```

Expected: compile error — `collectUniqueSessionTexts` undefined.

- [ ] **Step 3: Implement the cache builder**

Create `benchmarks/longmemeval/cache.go`:

```go
package longmemeval

import (
	"context"
	"fmt"
	"strings"

	"github.com/argylelabcoat/mempalace-go/internal/dialect"
	"github.com/argylelabcoat/mempalace-go/internal/embedder"
)

// collectUniqueSessionTexts scans all entries and returns the deduplicated set
// of session texts that will need embedding. The returned slice is deterministic
// (insertion-order of first occurrence).
func collectUniqueSessionTexts(entries []Entry, mode string, encoder *dialect.Encoder) []string {
	seen := make(map[string]bool)
	var unique []string
	for _, entry := range entries {
		for sessIdx, session := range entry.HaystackSessions {
			if sessIdx >= len(entry.HaystackSessionIDs) {
				continue
			}
			var userTurns []string
			for _, turnAny := range session {
				turn, _ := turnAny.(map[string]any)
				if turn == nil {
					continue
				}
				if role, ok := turn["role"].(string); ok && role == "user" {
					if content, ok := turn["content"].(string); ok {
						userTurns = append(userTurns, content)
					}
				}
			}
			if len(userTurns) == 0 {
				continue
			}
			text := strings.Join(userTurns, " ")
			if mode == "aaak" && encoder != nil {
				text = encoder.Compress(text, map[string]string{})
			}
			if !seen[text] {
				seen[text] = true
				unique = append(unique, text)
			}
		}
	}
	return unique
}

// BuildSessionCache embeds every unique session text across all entries and
// returns a map from session text → embedding vector. Call once before the
// main benchmark loop to avoid re-embedding repeated sessions.
func BuildSessionCache(ctx context.Context, emb *embedder.Embedder, entries []Entry, mode string, encoder *dialect.Encoder) (map[string][]float32, error) {
	texts := collectUniqueSessionTexts(entries, mode, encoder)
	if len(texts) == 0 {
		return nil, nil
	}

	vecs, err := emb.CreateEmbeddings(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("cache embed: %w", err)
	}

	cache := make(map[string][]float32, len(texts))
	for i, t := range texts {
		cache[t] = vecs[i]
	}
	return cache, nil
}
```

- [ ] **Step 4: Run the test to confirm it passes**

```bash
go test ./benchmarks/longmemeval/ -run TestBuildSessionCache_DeduplicatesTexts -v
```

Expected: `PASS`.

- [ ] **Step 5: Thread the cache through buildCorpus**

Change `buildCorpus` signature in `longmemeval.go` to accept an optional cache:

```go
func buildCorpus(ctx context.Context, emb *embedder.Embedder, entry Entry, mode string, encoder *dialect.Encoder, cache map[string][]float32) (*govector.Store, []string, error) {
```

Inside `buildCorpus`, replace the `emb.CreateEmbeddings(ctx, texts)` call with:

```go
vectors := make([][]float32, len(sessions))
var toEmbed []int // indices that need embedding
for i, s := range sessions {
    if cache != nil {
        if v, ok := cache[s.text]; ok {
            vectors[i] = v
            continue
        }
    }
    toEmbed = append(toEmbed, i)
}

if len(toEmbed) > 0 {
    missingTexts := make([]string, len(toEmbed))
    for j, idx := range toEmbed {
        missingTexts[j] = sessions[idx].text
    }
    vecs, err := emb.CreateEmbeddings(ctx, missingTexts)
    if err != nil {
        return nil, nil, fmt.Errorf("batch embed: %w", err)
    }
    for j, idx := range toEmbed {
        vectors[idx] = vecs[j]
    }
}
```

Update the call site in `Run` to pass the cache:

```go
// Before the loop, build the cache:
cache, err := BuildSessionCache(ctx, emb, entries, mode, encoder)
if err != nil {
    return nil, fmt.Errorf("build session cache: %w", err)
}

// In the loop:
store, corpusIDs, err := buildCorpus(ctx, emb, entry, mode, encoder, cache)
```

- [ ] **Step 6: Run full test suite**

```bash
go test ./...
```

Expected: all pass.

- [ ] **Step 7: Measure improvement**

```bash
go run . bench longmemeval --limit 10 data/longmemeval_s_cleaned.json
```

Then run the full 500:

```bash
go run . bench longmemeval data/longmemeval_s_cleaned.json
```

Note total wall time. If cross-question overlap is meaningful, the savings compound across 500 questions.

- [ ] **Step 8: Commit**

```bash
git add benchmarks/longmemeval/cache.go benchmarks/longmemeval/cache_test.go benchmarks/longmemeval/longmemeval.go
git commit -m "feat(bench): pre-compute cross-question session embedding cache"
```

---

## Task 5: Document findings and update README

After implementing and measuring:

- [ ] **Step 1: Update `README.md`** with the measured embed times before and after the optimization (the `go run . bench longmemeval --limit 10 ...` output with timing lines).

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add embed speed benchmark results before/after optimization"
```

---

## Self-Review Notes

- Task 4A and 4C are independent; either or both can be applied depending on profile results.
- Task 4A (`runWorkerPoolWithResource`) supersedes the simpler `runWorkerPool` — the test for `runWorkerPool` is still valid as a unit test for the simpler helper.
- The `fmt` import is required in `worker_test.go` for `TestWorkerPoolWithResource_UsesAllResources` — add `"fmt"` to imports in that file.
- `collectUniqueSessionTexts` in `cache.go` passes `nil` for `encoder` in tests — the production call always passes the real encoder. The nil guard (`encoder != nil`) handles this safely.
- The `buildCorpus` signature change (adding `cache` param) requires updating all call sites — there is only one call site in `Run()`.
