# Token-Aware Truncation Using Hugot's Internal Tokenizer

**Date:** 2026-04-10
**Status:** Approved

## Problem

Hugot's Go tokenizer backend (`gomlx/go-huggingface`) does not implement truncation. When input text produces more than 512 tokens, the model's tensor shape (e.g., `[batch, 559, 384]`) clashes with position embeddings `[1, 512, 384]`, causing a runtime crash:

```
dimension of axis #1 doesn't match and cannot be broadcast for BinaryOp (Add),
got shapes (Float32)[14 559 384] and (Float32)[1 512 384]
```

A previous attempt used an external tokenizer (`daulet/tokenizers`) to pre-truncate, but it failed because the two tokenizer implementations produce different token counts for the same text (e.g., 440 vs 559 tokens). The decode-then-re-encode approach was also lossy.

## Root Cause

The Rust tokenizer backend in hugot truncates token arrays to `MaxAllowedTokens` after encoding. The Go tokenizer backend (`tokenizeInputsGo` in `backends/tokenizer_go.go`) calls `EncodeWithAnnotations` but never truncates the output — it passes all tokens through to the model.

## Solution

Use **hugot's own internal Go tokenizer** to pre-truncate text before passing it to the pipeline. Since it's the same tokenizer instance that hugot uses internally during `Preprocess()`, the token counts are guaranteed to match.

### Access Path

```
e.pipeline (FeatureExtractionPipeline)
  .Model (backends.Model)
  .Tokenizer (backends.Tokenizer)
  .GoTokenizer (backends.GoTokenizer)
  .Tokenizer (api.Tokenizer interface, implemented by hftokenizer.Tokenizer)
```

### Algorithm

```go
func (e *Embedder) truncateText(text string) string {
    tok := e.pipeline.Model.Tokenizer.GoTokenizer.Tokenizer
    result := tok.EncodeWithAnnotations(text)
    limit := maxTokens - 2 // 510, leaving room for [CLS] + [SEP]

    if len(result.IDs) <= limit {
        return text
    }

    // Slice original text at the byte boundary of the last allowed token.
    // The spans map back to original text byte offsets, so no re-encoding needed.
    cutPoint := result.Spans[limit-1].End
    return text[:cutPoint]
}
```

`EncodeWithAnnotations` returns `Spans []TokenSpan` where each span has `Start`/`End` byte offsets into the original text. Slicing at `spans[limit-1].End` produces a substring whose first `limit` tokens are identical to the original text's first `limit` tokens — no decode/re-encode cycle needed.

### Fallback

If the pipeline or tokenizer is not accessible (shouldn't happen in practice), fall back to rune-based truncation at 400 runes.

## Changes

### `internal/embedder/hugot.go`

1. **Remove** `daulet/tokenizers` import
2. **Remove** `tokenizer *tokenizers.Tokenizer` field from `Embedder` struct
3. **Remove** `loadTokenizer()`, `defaultModelsDir()`, all debug `fmt.Fprintf` statements
4. **Remove** `truncateByTokens()` (old external-tokenizer version)
5. **Remove** round-trip re-encode verification in `truncateByTokens` and `CreateEmbeddings`
6. **Remove** `truncateLimit` constant (no longer needed)
7. **Rewrite** `truncateText()` to use hugot's internal tokenizer with byte-offset slicing
8. **Keep** `truncateByRunes()` as fallback
9. **Keep** `maxTokens = 512`
10. **Simplify** `CreateEmbeddings` — just `e.truncateText(t)`, no extra verification

### `go.mod`

- **Remove** `github.com/daulet/tokenizers` from direct dependencies

### Tests

Update `hugot_test.go` to work without the external tokenizer, test the new truncation with hugot's internal tokenizer.

## Dependency Impact

- **Remove:** `github.com/daulet/tokenizers` (no longer needed)
- **No new dependencies**

## Risk Assessment

- **Low risk:** Using the same tokenizer instance eliminates the token-count mismatch entirely
- **Edge case:** Very short texts where post-processor adds [CLS]/[SEP] could still exceed 512 — mitigated by the `limit = 510` headroom
- **Edge case:** Empty `Spans` array — fall back to rune truncation
