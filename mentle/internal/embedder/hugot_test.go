package embedder

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMain relocates the working directory to the module root so that the
// relative "models/onnx/model.onnx" probe inside New("","") resolves against
// the bundled model instead of falling through to a network download.
//
// Without this, `go test ./internal/embedder/...` runs with cwd = the package
// directory, where models/onnx/ does not exist, so New() falls back to
// hugot.DownloadModel and the test fails on machines without a working
// huggingface.co connection.
func TestMain(m *testing.M) {
	if _, err := os.Stat(filepath.Join("models", "onnx", "model.onnx")); err != nil {
		// cwd is not the module root; walk up to it.
		if root, err := findModuleRoot(); err == nil {
			_ = os.Chdir(root)
		}
	}
	os.Exit(m.Run())
}

// findModuleRoot walks upward from the current directory until it finds
// go.mod. Falls back to the parent of the "internal" directory when the
// search walks past the filesystem root.
func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", os.ErrNotExist
}

// TestTruncateByRunes verifies fallback rune truncation handles
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
// truncation when internal tokenizer is unavailable (nil pipeline).
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
// hugot's internal tokenizer. Requires tokenizers C library.
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

	veryLong := strings.Repeat("the quick brown fox jumps over lazy dog ", 200)
	got := emb.truncateText(veryLong)
	if got == "" {
		t.Fatal("truncation returned empty string")
	}
	if got == veryLong {
		t.Error("very long text was not truncated")
	}

	tok := emb.getHugotTokenizer()
	if tok == nil {
		t.Skip("internal tokenizer not available")
	}

	// Verify truncated text fits within maxTokens when re-encoded.
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
// chunk boundary are handled correctly — all texts get embeddings, none are dropped.
// This exercises the chunk-splitting logic across boundary (currently 64).
func TestCreateEmbeddings_LargeBatch(t *testing.T) {
	emb, err := New("", "")
	if err != nil {
		t.Fatalf("create embedder: %v", err)
	}
	defer emb.Close()

	ctx := context.Background()
	texts := make([]string, 65)
	for i := range texts {
		texts[i] = "the quick brown fox jumps over lazy dog"
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
	text := "the quick brown fox jumps over lazy dog"

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
		texts[i] = "the quick brown fox jumps over lazy dog"
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
	single := "the quick brown fox jumped over lazy dog near river bank on a warm summer afternoon while birds sang in trees and children played nearby along winding path through ancient forest full of tall oaks and whispering pines under a clear blue sky"
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
