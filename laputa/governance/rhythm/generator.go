package rhythm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	openai "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

// Generator generates rhythm reports from prompts.
type Generator interface {
	Generate(ctx context.Context, kind RhythmKind, prompt string) (*ReportResult, error)
}

// EinoGenerator uses a ChatModel to generate reports.
type EinoGenerator struct {
	chatModel *openai.ChatModel
}

// NewEinoGenerator creates a generator from an Eino ChatModel.
func NewEinoGenerator(cm *openai.ChatModel) *EinoGenerator {
	return &EinoGenerator{chatModel: cm}
}

// NewOpenAIGenerator creates a generator backed by an OpenAI-compatible API.
func NewOpenAIGenerator(ctx context.Context, baseURL, apiKey, model string) (*EinoGenerator, error) {
	cm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
	})
	if err != nil {
		return nil, fmt.Errorf("create openai chat model: %w", err)
	}
	return NewEinoGenerator(cm), nil
}

// Generate calls the LLM and parses the JSON response.
func (g *EinoGenerator) Generate(ctx context.Context, kind RhythmKind, prompt string) (*ReportResult, error) {
	resp, err := g.chatModel.Generate(ctx, []*schema.Message{
		{Role: schema.User, Content: prompt},
	})
	if err != nil {
		return nil, fmt.Errorf("llm generate: %w", err)
	}

	var result ReportResult
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		return nil, fmt.Errorf("parse report: %w", err)
	}
	result.GeneratedAt = time.Now().UTC()
	return &result, nil
}

// MockGenerator is a deterministic generator for tests.
type MockGenerator struct {
	Result *ReportResult
}

func NewMockGenerator() *MockGenerator {
	return &MockGenerator{
		Result: &ReportResult{
			Title:      "Daily Rhythm Report",
			Summary:    "All systems nominal.",
			Highlights: []string{"completed task A"},
		},
	}
}

func (m *MockGenerator) Generate(ctx context.Context, kind RhythmKind, prompt string) (*ReportResult, error) {
	r := *m.Result
	r.GeneratedAt = time.Now().UTC()
	return &r, nil
}
