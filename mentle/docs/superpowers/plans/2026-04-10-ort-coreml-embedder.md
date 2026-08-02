# ORT + CoreML Embedder Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the current XLA/Go hugot session with an ORT session that uses the CoreML execution provider on Apple Silicon, giving GPU/ANE-accelerated embeddings on macOS with a CPU-ORT fallback.

**Architecture:** The session creation in `internal/embedder/hugot.go` is changed to try ORT+CoreML first, fall back to ORT CPU, then fall back to the pure-Go backend. The ORT dylib bundled inside `yalue/onnxruntime_go@v1.27.0` already links `CoreML.framework` and `Metal.framework`, so no extra download is needed. Builds targeting ORT must pass `-tags ORT` to `go build`/`go test`.

**Tech Stack:** Go 1.26, `github.com/knights-analytics/hugot v0.7.0` (ORT backend, build tag `ORT`), `github.com/yalue/onnxruntime_go v1.27.0` (bundled arm64 dylib with CoreML EP), `github.com/knights-analytics/hugot/options` (WithCoreML).

---

## File Map

| File | Change |
|---|---|
| `internal/embedder/hugot.go` | Replace `NewXLASession`→`NewGoSession` cascade with ORT+CoreML→ORT CPU→Go cascade; add `WithOrtDylibPath` helper for darwin |
| `internal/embedder/hugot_test.go` | Add benchmark test measuring single-embed and batch-embed latency (existing file or new) |
| `cmd/benchperf/main.go` | Add `-tags ORT` note in usage comment; no logic changes needed |
| `Makefile` *(new)* | `build`, `build-ort`, `bench-perf`, `test`, `test-ort` targets |

---

## Task 1: Establish baseline perf numbers (ORT CPU, no CoreML)

This validates that the ORT build path works on your machine before adding CoreML.

**Files:**
- Modify: `internal/embedder/hugot.go`

- [ ] **Step 1: Read the current hugot.go to confirm starting state**

  The file is at `internal/embedder/hugot.go`. The `New()` function currently does:
  ```go
  session, err := hugot.NewXLASession()
  if err != nil {
      session, err = hugot.NewGoSession()
      ...
  }
  ```

- [ ] **Step 2: Replace session creation with ORT CPU → Go fallback**

  Replace lines 41-48 in `internal/embedder/hugot.go`:

  ```go
  // Try ORT (ONNX Runtime) first — faster than pure-Go on all hardware.
  // On darwin/arm64 the bundled dylib already links CoreML + Metal, but we
  // start with plain ORT CPU here to validate the build path.
  session, err := hugot.NewORTSession(ortDylibOption())
  if err != nil {
      // Fall back to pure-Go (no CGo required, works everywhere).
      session, err = hugot.NewGoSession()
      if err != nil {
          return nil, fmt.Errorf("create session: %w", err)
      }
  }
  ```

  Add the helper at the bottom of the file (before `Close`):

  ```go
  // ortDylibOption returns the hugot option that points ORT at the directory
  // containing the arm64 dylib bundled inside yalue/onnxruntime_go when running
  // on darwin/arm64. options.WithOnnxLibraryPath expects a directory; it appends
  // the platform-specific filename internally.
  // On other platforms returns a no-op option (ORT searches PATH/LD_LIBRARY_PATH).
  func ortDylibOption() options.WithOption {
      if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
          dir, err := findOrtDylibDir()
          if err == nil {
              return options.WithOnnxLibraryPath(dir)
          }
      }
      return func(o *options.Options) error { return nil }
  }

  // findOrtDylibDir locates the test_data directory inside yalue/onnxruntime_go
  // in the Go module cache. The arm64 dylib there already links CoreML + Metal.
  func findOrtDylibDir() (string, error) {
      gopath := os.Getenv("GOPATH")
      if gopath == "" {
          gopath = filepath.Join(os.Getenv("HOME"), "go")
      }
      dir := filepath.Join(gopath, "pkg", "mod",
          "github.com", "yalue", "onnxruntime_go@v1.27.0",
          "test_data")
      if _, err := os.Stat(dir); err != nil {
          return "", err
      }
      return dir, nil
  }
  ```

  Add missing imports to the import block:
  ```go
  "os"
  "path/filepath"
  "runtime"

  "github.com/knights-analytics/hugot/options"
  ```

- [ ] **Step 3: Verify the build compiles with ORT tag**

  ```bash
  go build -tags ORT ./internal/embedder/
  ```
  Expected: no errors. If you see `undefined: hugot.NewORTSession`, the `ORT` tag is not being passed.

- [ ] **Step 4: Run benchperf with ORT CPU to get baseline numbers**

  ```bash
  go run -tags ORT ./cmd/benchperf/
  ```
  Expected output (rough order of magnitude on M-series, ORT CPU):
  ```
  Embedder creation (model load): ~2-5s
  Single embedding (1 text):      ~20-80ms
  Batch embedding (50 texts):     ~300ms-1s  (~6-20ms per text)
  ```
  Note these numbers — they are your baseline to compare against after adding CoreML.

- [ ] **Step 5: Run tests without ORT tag to confirm Go fallback still works**

  ```bash
  go test ./internal/embedder/...
  ```
  Expected: PASS (falls back to Go session since ORT tag not present).

- [ ] **Step 6: Commit**

  ```bash
  git add internal/embedder/hugot.go
  git commit -m "feat(embedder): switch to ORT CPU session with Go fallback on darwin/arm64"
  ```

---

## Task 2: Add CoreML execution provider

**Files:**
- Modify: `internal/embedder/hugot.go`

- [ ] **Step 1: Update `New()` to try ORT+CoreML → ORT CPU → Go**

  Replace the session creation block added in Task 1 with:

  ```go
  // Try ORT + CoreML (Apple Silicon GPU/ANE acceleration).
  session, err := hugot.NewORTSession(ortDylibOption(), options.WithCoreML(coreMLFlags()))
  if err != nil {
      // CoreML unavailable (non-darwin, older macOS, or missing EP).
      // Try plain ORT (CPU).
      session, err = hugot.NewORTSession(ortDylibOption())
      if err != nil {
          // Final fallback: pure-Go (no CGo, works everywhere).
          session, err = hugot.NewGoSession()
          if err != nil {
              return nil, fmt.Errorf("create session: %w", err)
          }
      }
  }
  ```

  Add the `coreMLFlags` helper before `ortDylibOption`:

  ```go
  // coreMLFlags returns CoreML execution provider options that allow the ANE,
  // GPU, and CPU to all be used. See:
  // https://onnxruntime.ai/docs/execution-providers/CoreML-ExecutionProvider.html
  func coreMLFlags() map[string]string {
      return map[string]string{
          // MLComputeUnitsCPUAndNE | MLComputeUnitsCPUAndGPU | MLComputeUnitsAll
          // "0" = MLComputeUnitsAll (CPU+GPU+ANE, CoreML decides)
          "MLComputeUnits": "0",
      }
  }
  ```

- [ ] **Step 2: Build with ORT tag to confirm it compiles**

  ```bash
  go build -tags ORT ./...
  ```
  Expected: no errors.

- [ ] **Step 3: Run benchperf with ORT+CoreML to measure acceleration**

  ```bash
  go run -tags ORT ./cmd/benchperf/
  ```
  Expected (M-series with CoreML ANE/GPU):
  ```
  Single embedding (1 text):      ~5-20ms   (vs 20-80ms CPU baseline)
  Batch embedding (50 texts):     ~50-200ms (vs 300ms-1s CPU baseline)
  ```
  If numbers are similar to CPU baseline, CoreML EP may not be applying to all ops. This is normal for small models — CoreML has a compilation/startup cost that amortises over large batches.

- [ ] **Step 4: Run all tests (without ORT tag) to confirm no regressions**

  ```bash
  go test ./...
  ```
  Expected: all PASS.

- [ ] **Step 5: Run tests with ORT tag**

  ```bash
  go test -tags ORT ./...
  ```
  Expected: all PASS.

- [ ] **Step 6: Commit**

  ```bash
  git add internal/embedder/hugot.go
  git commit -m "feat(embedder): add CoreML execution provider for Apple Silicon GPU/ANE acceleration"
  ```

---

## Task 3: Write a Makefile with ORT build targets

Having to remember `-tags ORT` manually is error-prone. A Makefile documents intent and keeps CI/local builds consistent.

**Files:**
- Create: `Makefile`

- [ ] **Step 1: Write the Makefile**

  ```makefile
  .PHONY: build build-ort test test-ort bench-perf clean

  # Default build (pure Go, no CGo required, works everywhere)
  build:
  	go build ./...

  # ORT build — enables ONNX Runtime + CoreML on darwin/arm64
  # Requires CGo. The bundled arm64 dylib inside yalue/onnxruntime_go@v1.27.0
  # already links CoreML.framework and Metal.framework.
  build-ort:
  	go build -tags ORT ./...

  # Run all tests (pure Go backend)
  test:
  	go test ./...

  # Run all tests with ORT backend (darwin/arm64: CoreML accelerated)
  test-ort:
  	go test -tags ORT ./...

  # Run the performance benchmark (pure Go)
  bench-perf:
  	go run ./cmd/benchperf/

  # Run the performance benchmark with ORT + CoreML
  bench-perf-ort:
  	go run -tags ORT ./cmd/benchperf/

  clean:
  	rm -f mempalace mempalace-go mempalace-test coverage.out
  ```

- [ ] **Step 2: Verify all targets run cleanly**

  ```bash
  make build
  make test
  make bench-perf-ort
  ```
  Expected: `build` and `test` succeed; `bench-perf-ort` prints timing table.

- [ ] **Step 3: Commit**

  ```bash
  git add Makefile
  git commit -m "build: add Makefile with ORT/CoreML build targets for Apple Silicon"
  ```

---

## Task 4: Add a Go benchmark for embedding latency

A repeatable `go test -bench` benchmark lets you measure regression/improvement precisely across backend changes.

**Files:**
- Modify: `internal/embedder/hugot_test.go` (create if it doesn't exist)

- [ ] **Step 1: Check if hugot_test.go exists**

  ```bash
  ls internal/embedder/
  ```

- [ ] **Step 2: Create/add the benchmark**

  If `internal/embedder/hugot_test.go` does not exist, create it. If it exists, append the functions below.

  ```go
  package embedder_test

  import (
      "context"
      "testing"
  )

  // BenchmarkSingleEmbed measures single-text embedding latency.
  // Run with: go test -bench=BenchmarkSingleEmbed -benchtime=10s ./internal/embedder/
  // With ORT: go test -tags ORT -bench=BenchmarkSingleEmbed -benchtime=10s ./internal/embedder/
  func BenchmarkSingleEmbed(b *testing.B) {
      emb, err := New("", "")
      if err != nil {
          b.Fatalf("create embedder: %v", err)
      }
      defer emb.Close()

      ctx := context.Background()
      text := "the quick brown fox jumps over the lazy dog"

      b.ResetTimer()
      for i := 0; i < b.N; i++ {
          _, err := emb.CreateEmbedding(ctx, text)
          if err != nil {
              b.Fatalf("embed: %v", err)
          }
      }
  }

  // BenchmarkBatchEmbed measures batch embedding throughput (50 texts).
  // Run with: go test -bench=BenchmarkBatchEmbed -benchtime=10s ./internal/embedder/
  // With ORT: go test -tags ORT -bench=BenchmarkBatchEmbed -benchtime=10s ./internal/embedder/
  func BenchmarkBatchEmbed(b *testing.B) {
      emb, err := New("", "")
      if err != nil {
          b.Fatalf("create embedder: %v", err)
      }
      defer emb.Close()

      ctx := context.Background()
      texts := make([]string, 50)
      for i := range texts {
          texts[i] = "the quick brown fox jumps over the lazy dog"
      }

      b.ResetTimer()
      for i := 0; i < b.N; i++ {
          _, err := emb.CreateEmbeddings(ctx, texts)
          if err != nil {
              b.Fatalf("batch embed: %v", err)
          }
      }
  }
  ```

- [ ] **Step 3: Run benchmarks without ORT to get pure-Go baseline**

  ```bash
  go test -bench=. -benchtime=30s ./internal/embedder/
  ```
  Expected output:
  ```
  BenchmarkSingleEmbed-N   <N>   <ns/op>
  BenchmarkBatchEmbed-N    <N>   <ns/op>
  ```

- [ ] **Step 4: Run benchmarks with ORT+CoreML to compare**

  ```bash
  go test -tags ORT -bench=. -benchtime=30s ./internal/embedder/
  ```
  Record both sets of numbers. Typical improvement on M-series is 3-10x on batch.

- [ ] **Step 5: Commit**

  ```bash
  git add internal/embedder/hugot_test.go
  git commit -m "test(embedder): add Go benchmark for single and batch embedding latency"
  ```

---

## Notes and Troubleshooting

**"cannot find -lonnxruntime" or CGo linker errors:**
The directory returned by `findOrtDylibDir()` must exist and contain `onnxruntime_arm64.dylib`. Run:
```bash
ls ~/go/pkg/mod/github.com/yalue/onnxruntime_go@v1.27.0/test_data/onnxruntime_arm64.dylib
```
If missing, `go mod download github.com/yalue/onnxruntime_go@v1.27.0` will populate it.

**"image not found" at runtime on darwin:**
The dylib has `@rpath/libonnxruntime.1.24.1.dylib` as its install name. Set `DYLD_LIBRARY_PATH` to the directory containing it, or use `install_name_tool -change` to rewrite the rpath. The easiest workaround is to copy the dylib to `/usr/local/lib/` (standard search path):
```bash
sudo cp ~/go/pkg/mod/github.com/yalue/onnxruntime_go@v1.27.0/test_data/onnxruntime_arm64.dylib \
    /usr/local/lib/libonnxruntime.dylib
```
hugot's darwin fallback (`ort.SetSharedLibraryPath("libonnxruntime.dylib")`) will then find it automatically without the module-cache path resolution.

**CoreML not accelerating (same speed as CPU):**
CoreML compiles a model on first use and caches it in `~/Library/Caches/com.apple.e5rt.e5modelcache`. The first run will be slow; subsequent runs use the cached compiled model. If still slow after second run, the model may fall back to CPU because CoreML doesn't support all ONNX ops. Check the ORT log by temporarily setting `hugot.WithEnvLoggingLevel(1)` in `NewORTSession`.

**"undefined: hugot.NewORTSession" at compile time:**
You forgot `-tags ORT`. The ORT session is gated behind `//go:build cgo && (ORT || ALL)`.
