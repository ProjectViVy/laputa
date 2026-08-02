<!-- Parent: ../../AGENTS.md -->

# mentle/models/onnx — ONNX Embedding Model Files

**Generated:** 2026-08-01  
**Purpose:** Pre-trained embedding model (all-MiniLM-L6-v2) and tokenizer artifacts

---

## Purpose

The `onnx/` directory contains pre-trained model files:

- **model.onnx** — compiled ONNX model (384-dimensional)
- **tokenizer.json** — BPE tokenizer vocabulary
- **config.json** — model configuration
- **special_tokens_map.json** — special token definitions

Model: **sentence-transformers/all-MiniLM-L6-v2**

---

## Structure

```
onnx/
├── model.onnx                       # ONNX compiled model (~30MB)
├── tokenizer.json                   # BPE vocabulary
├── config.json                      # Model config (hidden size, etc.)
├── special_tokens_map.json          # Special tokens ([CLS], [SEP], etc.)
└── tokenizer_config.json            # Tokenizer configuration
```

---

## Model Details

| Property | Value |
|----------|-------|
| Architecture | MiniLM (distilled BERT) |
| Output dimensions | 384 |
| Max sequence length | 256 |
| Vocabulary size | 30,522 |
| License | Apache 2.0 |

---

## Runtime Integration

Loaded by hugot at startup:

```go
embedder, err := embedder.NewEmbedder(ctx, "./models/onnx")
if err != nil {
    log.Fatal(err)
}

vectors, err := embedder.Embed(ctx, texts)
```

No external Python runtime or daemon required.

---

## Updating the Model

To switch to a different embedding model:

1. Export new model to ONNX format
2. Replace files in this directory
3. Update `models/onnx/config.json` with new dimensions
4. Test with `make build && make test`

---

## License

Model licensed under Apache 2.0. No redistribution restrictions for binary artifacts.

---

## MANUAL

Model files are stable. Do not modify without design review (dimension changes break all indexes).

Parent reference: ../../AGENTS.md
