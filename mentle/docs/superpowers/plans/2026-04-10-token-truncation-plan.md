# Token-Aware Truncation Using Hugot's Internal Tokenizer — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the broken external-tokenizer truncation with hugot's own internal Go tokenizer, using byte-offset spans to slice text to fit within 512 tokens.

**Architecture:** Access hugot's `hftokenizer.Tokenizer` via `pipeline.Model.Tokenizer.GoTokenizer.Tokenizer`, call `EncodeWithAnnotations` to get token counts and byte spans, then slice the original text at the appropriate span boundary when over-limit. Remove the external `daulet/tokenizers` dependency entirely.

**Tech Stack:** Go, hugot v0.7.0, gomlx/go-huggingface (hugot's internal tokenizer)

---

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/embedder/hugot.go` | Rewrite truncation | Remove external tokenizer, use hugot's internal tokenizer for truncation |
| `internal/embedder/hugot_test.go` | Rewrite tests | Update tests for new truncation approach, test byte-offset slicing |
| `go.mod` | Modify | Remove `github.com/daulet/tokenizers` from direct dependencies |

---

### Task 1: Rewrite `hugot.go` — Remove External Tokenizer, Add Internal Tokenizer Truncation

**Files:**
- Modify: `internal/embedder/hugot.go` (full rewrite of truncation logic)

**Understanding the tokenizer access:**
- `e.pipeline` is `*pipelines.FeatureExtractionPipeline`
- `e.pipeline.Model` is `*backends.Model`
- `e.pipeline.Model.Tokenizer` is `*backends.Tokenizer`
- `e.pipeline.Model.Tokenizer.GoTokenizer` is `*backends.GoTokenizer` (nil if using ORT/Rust backend)
- `e.pipeline.Model.Tokenizer.GoTokenizer.Tokenizer` is `api.Tokenizer` (interface, implemented by `hftokenizer.Tokenizer`)
- `EncodeWithAnnotations(text)` returns `api.AnnotatedEncoding{IDs, Spans, SpecialTokensMask}`
- For BERT-style models with special tokens, the output structure is:
  - IDs: `[CLS_ID, real1, real2, ..., realN, SEP_ID]`
  - Spans: `[{-1,-1}, {s1,e1}, {s2,e2}, ..., {sN,eN}, {-1,-1}]`
  - Special tokens have `Start: -1, End: -1` spans
- Total tokens = real tokens + 2 (CLS + SEP)
- Max allowed = 512, so max real tokens = 510

- [ ] **Step 1: Rewrite `hugot.go`**

Replace the entire file content. Key changes:
1. Remove `github.com/daulet/tokenizers` import
2. Remove `tokenizer *tokenizers.Tokenizer` field from Embedder struct
3. Remove `truncateLimit` constant
4. Remove `loadTokenizer()`, `defaultModelsDir()`, all debug prints
5. Add `truncateText` using hugot's internal tokenizer
6. Keep `truncateByRunes` as fallback
7. Keep `maxTokens = 512`

Here is the new `hugot.go`:

```go
// Package embedder provides text embedding using Hugging Face transformers via hugot.
// It uses ONNX models for efficient, native Go embeddings without external processes.
package embedder

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/knights-analytics/hugot"
	"github.com/knights-analytics/hugot/options"
	"github.com/knights-analytics/hugot/pipelines"
)

// maxTokens is the model's position embedding limit. all-MiniLM-L6-v2
// and most sentence-transformer models support up to 512 subword tokens.
const maxTokens = 512

// Embedder generates text embeddings using hugot (Hugging Face ONNX runtime).
type Embedder struct {
	pipeline    *pipelines.FeatureExtractionPipeline
	session     *hugot.Session
	modelPath   string
	modelsDir   string
	modelName   string
	initialized bool
}

// New creates a new Embedder. It downloads (if needed) and loads the model.
func New(modelName string, modelsDir string) (*Embedder, error) {
	if modelName == "" {
		modelName = "sentence-transformers/all-MiniLM-L6-v2"
	}

	downloadOpts := hugot.NewDownloadOptions()
	downloadOpts.OnnxFilePath = "onnx/model.onnx"
	downloadOpts.Verbose = false

	modelPath, err := hugot.DownloadModel(modelName, modelsDir, downloadOpts)
	if err != nil {
		return nil, fmt.Errorf("download model: %w", err)
	}

	e := &Embedder{
		modelPath: modelPath,
		modelsDir: modelsDir,
		modelName: modelName,
	}

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

	config := hugot.FeatureExtractionConfig{
		ModelPath: modelPath,
		Name:      "mempalace-embeddings",
	}

	pipeline, err := hugot.NewPipeline[*pipelines.FeatureExtractionPipeline](session, config)
	if err != nil {
		session.Destroy()
		return nil, fmt.Errorf("create pipeline: %w", err)
	}

	e.pipeline = pipeline
	e.session = session
	e.initialized = true

	return e, nil
}

// truncateText limits text to fit within the model's token limit.
// Uses hugot's own internal tokenizer to count tokens exactly,
// then slices the original text at the appropriate byte boundary.
// Falls back to rune-based truncation if the internal tokenizer is unavailable.
func (e *Embedder) truncateText(text string) string {
	tok := e.getHugotTokenizer()
	if tok == nil {
		return truncateByRunes(text)
	}

	result := tok.EncodeWithAnnotations(text)
	if len(result.IDs) <= maxTokens {
		return text
	}

	// IDs layout with special tokens: [CLS, real1, ..., realN, SEP]
	// Spans layout: [{-1,-1}, {s1,e1}, ..., {sN,eN}, {-1,-1}]
	// Max real tokens = maxTokens - 2 (reserve 2 for CLS + SEP).
	maxRealTokens := maxTokens - 2

	// Find the byte boundary: spans[0]=CLS (special), spans[1]=first real token.
	// The last allowed real token is at index maxRealTokens in the spans array
	// (index 0 is CLS, so real token i is at spans[i]).
	if maxRealTokens < len(result.Spans) {
		cutPoint := result.Spans[maxRealTokens].End
		if cutPoint > 0 && cutPoint <= len(text) {
			return text[:cutPoint]
		}
	}

	return truncateByRunes(text)
}

// getHugotTokenizer returns the internal Go tokenizer from hugot's pipeline.
// Returns nil if the pipeline is not initialized or the Go tokenizer is unavailable
// (e.g., when using the ORT/Rust backend).
func (e *Embedder) getHugotTokenizer() interface {
	EncodeWithAnnotations(string) struct {
		IDs   []int
		Spans []struct{ Start, End int }
	}
} {
	if e.pipeline == nil || e.pipeline.Model == nil {
		return nil
	}
	tk := e.pipeline.Model.Tokenizer
	if tk == nil || tk.GoTokenizer == nil {
		return nil
	}
	return tk.GoTokenizer.Tokenizer
}

// truncateByRunes is the conservative fallback: limits text to ~400 Unicode code-points.
func truncateByRunes(text string) string {
	runes := []rune(text)
	if len(runes) <= 400 {
		return text
	}
	return string(runes[:400])
}

// CreateEmbedding generates a float32 vector for the given text.
func (e *Embedder) CreateEmbedding(ctx context.Context, text string) ([]float32, error) {
	output, err := e.pipeline.RunPipeline([]string{e.truncateText(text)})
	if err != nil {
		return nil, err
	}

	if len(output.Embeddings) == 0 {
		return nil, fmt.Errorf("empty embedding output")
	}

	return output.Embeddings[0], nil
}

// CreateEmbeddings batch-embeds multiple texts in a single forward pass.
// It processes texts in chunks of 64 (recommended by hugot) and handles
// shape mismatches by falling back to single embeddings when needed.
func (e *Embedder) CreateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	truncated := make([]string, len(texts))
	for i, t := range texts {
		truncated[i] = e.truncateText(t)
	}

	const chunkSize = 64
	allEmbeddings := make([][]float32, 0, len(texts))

	for i := 0; i < len(truncated); i += chunkSize {
		end := min(i+chunkSize, len(truncated))
		chunk := truncated[i:end]

		output, err := e.pipeline.RunPipeline(chunk)
		if err != nil {
			for _, text := range chunk {
				single, err2 := e.pipeline.RunPipeline([]string{text})
				if err2 != nil {
					return nil, fmt.Errorf("batch+fallback embed: %w (single: %v)", err, err2)
				}
				if len(single.Embeddings) > 0 {
					allEmbeddings = append(allEmbeddings, single.Embeddings[0])
				}
			}
			continue
		}
		allEmbeddings = append(allEmbeddings, output.Embeddings...)
	}
	return allEmbeddings, nil
}

// coreMLFlags returns CoreML execution provider options that allow the ANE,
// GPU, and CPU to all be used. See:
// https://onnxruntime.ai/docs/execution-providers/CoreML-ExecutionProvider.html
func coreMLFlags() map[string]string {
	return map[string]string{
		// MLComputeUnitsAll (0) = CPU+GPU+ANE, CoreML decides optimal placement.
		"MLComputeUnits": "0",
	}
}

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

// Close releases the hugot session resources.
func (e *Embedder) Close() {
	if e.session != nil {
		e.session.Destroy()
	}
}
```

**IMPORTANT NOTE about `getHugotTokenizer`:** The return type above uses an anonymous interface which is verbose. A cleaner approach is to use the actual type from hugot's dependency. The real implementation should import `"github.com/gomlx/go-huggingface/tokenizers/api"` and return `api.Tokenizer`. Here is the corrected signature:

```go
import "github.com/gomlx/go-huggingface/tokenizers/api"

func (e *Embedder) getHugotTokenizer() api.Tokenizer {
	if e.pipeline == nil || e.pipeline.Model == nil {
		return nil
	}
	tk := e.pipeline.Model.Tokenizer
	if tk == nil || tk.GoTokenizer == nil {
		return nil
	}
	return tk.GoTokenizer.Tokenizer
}
```

And `truncateText` uses `api.Tokenizer` directly:

```go
func (e *Embedder) truncateText(text string) string {
	tok := e.getHugotTokenizer()
	if tok == nil {
		return truncateByRunes(text)
	}

	result := tok.EncodeWithAnnotations(text)
	if len(result.IDs) <= maxTokens {
		return text
	}

	maxRealTokens := maxTokens - 2

	if maxRealTokens < len(result.Spans) {
		cutPoint := result.Spans[maxRealTokens].End
		if cutPoint > 0 && cutPoint <= len(text) {
			return text[:cutPoint]
		}
	}

	return truncateByRunes(text)
}
```

Add the import for `api` in the import block. The full import block should be:

```go
import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/gomlx/go-huggingface/tokenizers/api"
	"github.com/knights-analytics/hugot"
	"github.com/knights-analytics/hugot/options"
	"github.com/knights-analytics/hugot/pipelines"
)
```

Note: `github.com/gomlx/go-huggingface` is already an indirect dependency via hugot, so no new dependency needed.

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/embedder/`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/embedder/hugot.go
git commit -m "refactor(embedder): use hugot's internal tokenizer for token-aware truncation

Replace the broken external tokenizer (daulet/tokenizers) approach with
hugot's own internal Go tokenizer. EncodeWithAnnotations returns byte-offset
spans, so we can slice the original text at the exact boundary that produces
<= 510 real tokens (512 minus [CLS] + [SEP]).

This eliminates the token-count mismatch between two different tokenizer
implementations and removes the dependency on daulet/tokenizers."
```

---

### Task 2: Rewrite `hugot_test.go` — Update Tests for New Truncation

**Files:**
- Modify: `internal/embedder/hugot_test.go`

The old tests reference `loadTokenizer()`, `e.tokenizer`, `truncateByTokens()`, and `maxRunes` — all removed. Tests must be rewritten.

The new test approach:
1. `TestTruncateByRunes` — unchanged (function still exists with same signature)
2. `TestTruncateText_Method` — test that a bare Embedder (no pipeline) falls back to rune truncation
3. `TestTruncateText_WithHugotTokenizer` — requires a real embedder (skip if unavailable), test that long text is truncated to <= 512 tokens
4. Keep `TestCreateEmbeddings_LargeBatch` and benchmarks as-is (they create real embedders)

- [ ] **Step 1: Rewrite the test file**

```go
package embedder

import (
	"context"
	"strings"
	"testing"
)

// TestTruncateByRunes verifies the fallback rune truncation handles
// pathological inputs (long URLs, Unicode math symbols).
func TestTruncateByRunes(t *testing.T) {
	short := "hello world"
	if got := truncateByRunes(short); got != short {
		t.Errorf("short text modified: got %q, want %q", got, short)
	}

	atLimit := strings.Repeat("a", 400)
	if got := truncateByRunes(atLimit); got != atLimit {
		t.Errorf("text at limit was modified")
	}

	long := strings.Repeat("a", 500)
	got := truncateByRunes(long)
	if len([]rune(got)) != 400 {
		t.Errorf("truncated text has %d runes, want %d", len([]rune(got)), 400)
	}

	longURL := "https://" + strings.Repeat("x", 450) + ".com/path/to/image.jpg"
	gotURL := truncateByRunes(longURL)
	if len([]rune(gotURL)) > 400 {
		t.Errorf("URL not truncated: %d runes > 400", len([]rune(gotURL)))
	}

	mathText := strings.Repeat("𝐴", 500)
	gotMath := truncateByRunes(mathText)
	if len([]rune(gotMath)) > 400 {
		t.Errorf("unicode math not truncated: %d runes > 400", len([]rune(gotMath)))
	}
}

// TestTruncateText_Fallback verifies that truncateText falls back to rune
// truncation when the internal tokenizer is unavailable (nil pipeline).
func TestTruncateText_Fallback(t *testing.T) {
	e := &Embedder{}

	short := "hello world"
	if got := e.truncateText(short); got != short {
		t.Errorf("short text modified: got %q, want %q", got, short)
	}

	long := strings.Repeat("a", 500)
	got := e.truncateText(long)
	if len([]rune(got)) != 400 {
		t.Errorf("fallback truncation: got %d runes, want 400", len([]rune(got)))
	}
}

// TestTruncateText_WithHugotTokenizer tests token-aware truncation using
// hugot's internal tokenizer. Requires the tokenizers C library.
func TestTruncateText_WithHugotTokenizer(t *testing.T) {
	emb, err := New("", "")
	if err != nil {
		t.Skipf("embedder unavailable, skipping: %v", err)
	}
	defer emb.Close()

	short := "hello world"
	if got := emb.truncateText(short); got != short {
		t.Errorf("short text modified: got %q, want %q", got, short)
	}

	veryLong := strings.Repeat("the quick brown fox jumps over the lazy dog ", 200)
	got := emb.truncateText(veryLong)
	if got == "" {
		t.Fatal("truncation returned empty string")
	}
	if got == veryLong {
		t.Error("very long text was not truncated")
	}

	// Verify the truncated text fits within maxTokens when re-encoded.
	tok := emb.getHugotTokenizer()
	if tok == nil {
		t.Skip("internal tokenizer not available")
	}
	reEncoded := tok.EncodeWithAnnotations(got)
	if len(reEncoded.IDs) > maxTokens {
		t.Errorf("truncated text produces %d tokens, want <= %d", len(reEncoded.IDs), maxTokens)
	}

	// Long URL test.
	longURL := "https://" + strings.Repeat("abcdef", 200) + ".com/" + strings.Repeat("path", 100)
	gotURL := emb.truncateText(longURL)
	reEncURL := tok.EncodeWithAnnotations(gotURL)
	if len(reEncURL.IDs) > maxTokens {
		t.Errorf("URL truncation: %d tokens > %d", len(reEncURL.IDs), maxTokens)
	}
}

// TestCreateEmbeddings_LargeBatch verifies that batches larger than the internal
// chunk boundary are handled correctly.
func TestCreateEmbeddings_LargeBatch(t *testing.T) {
	emb, err := New("", "")
	if err != nil {
		t.Fatalf("create embedder: %v", err)
	}
	defer emb.Close()

	ctx := context.Background()
	texts := make([]string, 65)
	for i := range texts {
		texts[i] = "the quick brown fox jumps over the lazy dog"
	}

	vecs, err := emb.CreateEmbeddings(ctx, texts)
	if err != nil {
		t.Fatalf("CreateEmbeddings: %v", err)
	}
	if len(vecs) != len(texts) {
		t.Errorf("expected %d embeddings, got %d", len(texts), len(vecs))
	}
	for i, v := range vecs {
		if len(v) == 0 {
			t.Errorf("embedding %d is empty", i)
		}
	}
}

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

func BenchmarkCorpusEmbed(b *testing.B) {
	emb, err := New("", "")
	if err != nil {
		b.Fatalf("create embedder: %v", err)
	}
	defer emb.Close()

	ctx := context.Background()
	single := "the quick brown fox jumped over the lazy dog near the river bank on a warm summer afternoon while birds sang in the trees and children played nearby along the winding path through the ancient forest full of tall oaks and whispering pines under a clear blue sky"
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

- [ ] **Step 2: Verify compilation**

Run: `go vet ./internal/embedder/`
Expected: No errors

- [ ] **Step 3: Run tests (may skip if tokenizers C library unavailable)**

Run: `go test ./internal/embedder/ -v -run 'TestTruncate|TestCreateEmbeddings' -count=1`
Expected: `TestTruncateByRunes` and `TestTruncateText_Fallback` pass. Other tests may skip.

- [ ] **Step 4: Commit**

```bash
git add internal/embedder/hugot_test.go
git commit -m "test(embedder): update tests for hugot internal tokenizer truncation"
```

---

### Task 3: Remove `daulet/tokenizers` from `go.mod`

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Remove the direct dependency**

Run: `go mod tidy`
Expected: `github.com/daulet/tokenizers` moves from `require` block to `require (// indirect)` block or is removed entirely (it's still pulled in transitively by hugot, but no longer a direct dependency).

Verify with: `grep "daulet/tokenizers" go.mod`
Expected: It should appear only in the indirect block (or not at all if hugot doesn't need it directly).

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: No errors (note: embedder binary may fail to link due to missing tokenizers C library on this machine, but `go build ./internal/embedder/` should succeed since it only compiles, doesn't link the C library until runtime)

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: remove direct dependency on daulet/tokenizers"
```

---

### Task 4: Final Validation

**Files:** None modified

- [ ] **Step 1: Run go vet**

Run: `go vet ./internal/embedder/`
Expected: No errors

- [ ] **Step 2: Run gofmt/go fmt**

Run: `gofmt -l internal/embedder/`
Expected: No output (all files formatted)

- [ ] **Step 3: Run golangci-lint on changed files**

Run: `golangci-lint run ./internal/embedder/`
Expected: No new issues (pre-existing issues from other packages are acceptable)

- [ ] **Step 4: Run all embedder tests**

Run: `go test -v -count=1 ./internal/embedder/ -run 'Test'`
Expected: `TestTruncateByRunes` and `TestTruncateText_Fallback` pass. Integration tests (`TestTruncateText_WithHugotTokenizer`, `TestCreateEmbeddings_LargeBatch`) may skip if tokenizers C library is unavailable.

- [ ] **Step 5: Verify go.mod is clean**

Run: `go mod tidy && git diff go.mod go.sum`
Expected: No diff (already tidy)
