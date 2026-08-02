<!-- Parent: ../AGENTS.md -->

# mentle/models — ONNX Embedding Model Files

**Generated:** 2026-08-01  
**Purpose:** Pre-trained embedding model and tokenizer configuration

---

## Purpose

The `models/` directory contains the ONNX embedding model used for vector search:

- **Model file** — all-MiniLM-L6-v2 compiled to ONNX format
- **Tokenizer** — vocabulary and tokenizer configuration
- **Configuration** — model metadata and hyperparameters

No Python dependencies; ONNX runtime is pure Go via hugot.

---

## Structure

```
models/
└── onnx/
    ├── model.onnx                 # Model weights (ONNX format)
    ├── config.json                # Model configuration
    ├── tokenizer.json             # Tokenizer with vocabulary
    ├── special_tokens_map.json    # Special token mappings
    └── tokenizer_config.json      # Tokenizer hyperparameters
```

---

## Model Details

### all-MiniLM-L6-v2

- **Model:** sentence-transformers/all-MiniLM-L6-v2
- **Architecture:** DistilBERT-based, 22M parameters
- **Embedding dimension:** 384
- **Max sequence length:** 512 tokens (truncated to 400 runes in queries)
- **License:** Apache 2.0

### Tokenization

- **Vocabulary size:** ~30,000 tokens
- **Special tokens:** [CLS], [SEP], [PAD], [UNK], [MASK]
- **Padding:** applied to max sequence length
- **Truncation:** queries truncated to 400 Unicode runes before tokenization

---

## Usage

Load model with hugot (ONNX Go runtime):

```go
import "github.com/knights-analytics/hugot"

// Load model
model, err := hugot.NewDistilBertForSentenceTransformers("./models/onnx")
if err != nil {
    log.Fatal(err)
}

// Embed text
embedding, err := model.Embeddings(ctx, []string{"query text"})
if err != nil {
    log.Fatal(err)
}
```

---

## Download

If model files are missing, rebuild from source:

```bash
cd mentle
# Model files are embedded in the build or downloaded on first run
# Alternatively, download from HuggingFace:
# https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2
```

---

## Performance

- **Batch embedding (64 texts):** P95 ≤ 50ms
- **Single embedding:** P95 ≤ 5ms (after warmup)
- **Memory:** ~100MB resident (model + cache)

---

## Conventions

- Model files are **not committed** to git (too large)
- Embedded in Docker image or downloaded at runtime
- Version pinning: use specific model version, not latest
- License compliance: Apache 2.0, include in binary distribution

---

## MANUAL

When updating:

1. Do not replace model without performance comparison
2. Verify new model has compatible tokenizer
3. Update embedding dimension in code if changed (currently 384)
4. Test with LongMemEval benchmark before deploying

Parent reference: ../AGENTS.md
