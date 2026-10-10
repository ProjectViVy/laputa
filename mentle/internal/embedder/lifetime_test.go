package embedder

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// Native ONNX execution must be joined before session destruction even when
// the caller's deadline expires. No model download or replacement backend.
func TestCanceledNativeEmbeddingCanCloseSafely(t *testing.T) {
	models, err := filepath.Abs("models/onnx")
	if err != nil {
		t.Fatal(err)
	}
	e, err := NewLocal("", models)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	_, err = e.CreateEmbedding(ctx, "synthetic native lifecycle probe")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expired embedding: %v", err)
	}
	e.Close()
	e.Close()
	if _, err := e.CreateEmbedding(context.Background(), "closed"); !errors.Is(err, ErrClosed) {
		t.Fatalf("closed single embedding: %v", err)
	}
	if _, err := e.CreateEmbeddings(context.Background(), []string{"closed"}); !errors.Is(err, ErrClosed) {
		t.Fatalf("closed batch embedding: %v", err)
	}
}
