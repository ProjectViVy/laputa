// Package embedder provides text embedding using Hugging Face transformers via hugot.
// It uses ONNX models for efficient, native Go embeddings without external processes.
package embedder

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
// If modelsDir is empty, it uses the bundled model in ./models/onnx/.
func New(modelName string, modelsDir string) (*Embedder, error) {
	if modelName == "" {
		modelName = "sentence-transformers/all-MiniLM-L6-v2"
	}

	// Check for bundled model first (avoid network download).
	// Probe order: 1) modelsDir/model.onnx  2) modelsDir/onnx/model.onnx
	//              3) cwd ./models/onnx/model.onnx
	bundledPath := filepath.Join("models", "onnx")
	if modelsDir != "" {
		for _, candidate := range []string{
			filepath.Join(modelsDir, "model.onnx"),
			filepath.Join(modelsDir, "onnx", "model.onnx"),
		} {
			if _, err := os.Stat(candidate); err == nil {
				return newFromPath(filepath.Dir(candidate), modelName)
			}
		}
	}
	if _, err := os.Stat(filepath.Join(bundledPath, "model.onnx")); err == nil {
		return newFromPath(bundledPath, modelName)
	}

	downloadOpts := hugot.NewDownloadOptions()
	downloadOpts.OnnxFilePath = "onnx/model.onnx"
	downloadOpts.Verbose = false

	modelPath, err := hugot.DownloadModel(context.Background(), modelName, modelsDir, downloadOpts)
	if err != nil {
		return nil, fmt.Errorf("download model: %w", err)
	}

	e := &Embedder{
		modelPath: modelPath,
		modelsDir: modelsDir,
		modelName: modelName,
	}

	// Try ORT + CoreML (Apple Silicon GPU/ANE acceleration).
	session, err := hugot.NewORTSession(context.Background(), ortDylibOption(), options.WithCoreML(coreMLFlags()))
	if err != nil {
		// CoreML unavailable (non-darwin, older macOS, or missing EP).
		// Try plain ORT (CPU).
		session, err = hugot.NewORTSession(context.Background(), ortDylibOption())
		if err != nil {
			// Final fallback: pure-Go (no CGo, works everywhere).
			session, err = hugot.NewGoSession(context.Background())
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

// newFromPath creates an Embedder from a local model path (no download).
func newFromPath(modelPath string, modelName string) (*Embedder, error) {
	e := &Embedder{
		modelPath: modelPath,
		modelName: modelName,
	}

	// Try ORT + CoreML (Apple Silicon GPU/ANE acceleration).
	session, err := hugot.NewORTSession(context.Background(), ortDylibOption(), options.WithCoreML(coreMLFlags()))
	if err != nil {
		// CoreML unavailable (non-darwin, older macOS, or missing EP).
		// Try plain ORT (CPU).
		session, err = hugot.NewORTSession(context.Background(), ortDylibOption())
		if err != nil {
			// Final fallback: pure-Go (no CGo, works everywhere).
			session, err = hugot.NewGoSession(context.Background())
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
// then slices original text at byte-offset boundary when over-limit.
// Falls back to rune-based truncation if internal tokenizer is unavailable.
func (e *Embedder) truncateText(text string) string {
	tok := e.getHugotTokenizer()
	if tok == nil {
		return truncateByRunes(text)
	}

	result := tok.EncodeWithAnnotations(text)
	if len(result.IDs) <= maxTokens {
		return text
	}

	// IDs layout with special tokens: [CLS_ID, real1, ..., realN, SEP_ID]
	// Spans layout: [{-1,-1}, {s1,e1}, ..., {sN,eN}, {-1,-1}]
	// Max real tokens = maxTokens - 2 (reserve 2 for CLS + SEP).
	maxRealTokens := maxTokens - 2

	// Find byte boundary: spans[0]=CLS (special), spans[1]=first real token.
	// The last allowed real token is at index maxRealTokens in spans array
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
// Returns nil if pipeline is not initialized or Go tokenizer is unavailable
// (e.g., when using ORT/Rust backend instead of Go/XLA).
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

// truncateByRunes is conservative fallback: limits text to ~400 Unicode code-points.
func truncateByRunes(text string) string {
	runes := []rune(text)
	if len(runes) <= 400 {
		return text
	}
	return string(runes[:400])
}

// CreateEmbedding generates a float32 vector for given text.
func (e *Embedder) CreateEmbedding(ctx context.Context, text string) ([]float32, error) {
	output, err := e.pipeline.RunPipeline(ctx, []string{e.truncateText(text)})
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

		output, err := e.pipeline.RunPipeline(ctx, chunk)
		if err != nil {
			for _, text := range chunk {
				single, err2 := e.pipeline.RunPipeline(ctx, []string{text})
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

// coreMLFlags returns CoreML execution provider options that allow ANE,
// GPU, and CPU to all be used. See:
// https://onnxruntime.ai/docs/execution-providers/CoreML-ExecutionProvider.html
func coreMLFlags() map[string]string {
	return map[string]string{
		// MLComputeUnitsAll (0) = CPU+GPU+ANE, CoreML decides optimal placement.
		"MLComputeUnits": "0",
	}
}

// ortDylibOption returns hugot option that points ORT at the directory
// containing the arm64 dylib bundled inside yalue/onnxruntime_go when running
// on darwin/arm64. options.WithOnnxLibraryPath expects a directory; it appends
// platform-specific filename internally.
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

// Close releases hugot session resources.
func (e *Embedder) Close() {
	if e.session != nil {
		e.session.Destroy()
	}
}
