// Command warmup primes the local Hugging Face cache with the
// all-MiniLM-L6-v2 model that embedder.New("") requests.
//
// It mirrors the call path used by the embedder package so that
// the cache layout matches what hugot expects at runtime:
//   - HF_HOME (default ~/.cache/huggingface) holds blobs/ + snapshots/
//   - onnx/model.onnx is fetched and the ORT session is built
//
// Usage:
//   go run ./cmd/warmup            # default model
//   go run ./cmd/warmup -model X   # custom HF repo id
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/dashimaki/mentle/internal/embedder"
)

func main() {
	model := flag.String("model", "sentence-transformers/all-MiniLM-L6-v2", "HF repo id to warm up")
	flag.Parse()

	start := time.Now()
	log.Printf("warming up embedder model: %s", *model)

	e, err := embedder.New(*model, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "warmup failed after %s: %v\n", time.Since(start).Truncate(time.Millisecond), err)
		os.Exit(1)
	}
	defer e.Close()

	// Exercise the pipeline with a tiny input so ORT actually loads weights.
	vec, err := e.CreateEmbedding(context.Background(), "warmup probe")
	if err != nil {
		fmt.Fprintf(os.Stderr, "embed probe failed after %s: %v\n", time.Since(start).Truncate(time.Millisecond), err)
		os.Exit(1)
	}
	if len(vec) == 0 {
		fmt.Fprintf(os.Stderr, "warmup produced empty vector\n")
		os.Exit(1)
	}

	log.Printf("warmup ok: dim=%d, elapsed=%s", len(vec), time.Since(start).Truncate(time.Millisecond))
}