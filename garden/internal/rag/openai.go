package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type OpenAIPlanner struct {
	BaseURL, APIKey, Model string
	Client                 *http.Client
}

func (p *OpenAIPlanner) Plan(ctx context.Context, input PlannerInput) (RetrievalPlan, error) {
	return p.plan(ctx, "Create a retrieval plan", input)
}
func (p *OpenAIPlanner) Refine(ctx context.Context, input PlannerInput) (RetrievalPlan, error) {
	return p.plan(ctx, "Judge sufficiency and create only genuinely new follow-up queries, or set stop=true", input)
}
func (p *OpenAIPlanner) plan(ctx context.Context, instruction string, input PlannerInput) (RetrievalPlan, error) {
	payload, _ := json.Marshal(input)
	prompt := instruction + ". Return JSON only with queries (max 4), entities (max 4), temporal, stop. Input: " + string(payload)
	content, err := p.complete(ctx, prompt)
	if err != nil {
		return RetrievalPlan{}, err
	}
	plan, err := decodePlan([]byte(stripFence(content)))
	if err != nil {
		return RetrievalPlan{}, fmt.Errorf("parse planner response: %w", err)
	}
	return plan, nil
}
func (p *OpenAIPlanner) Summarize(ctx context.Context, intent string, evidence []Evidence) (string, error) {
	payload, _ := json.Marshal(evidence)
	return p.complete(ctx, "Build a concise context for the intent. Use only supplied evidence and cite every claim as [evidence_id]. Intent: "+intent+" Evidence: "+string(payload))
}
func (p *OpenAIPlanner) complete(ctx context.Context, prompt string) (string, error) {
	if p.BaseURL == "" || p.APIKey == "" || p.Model == "" {
		return "", errors.New("planner is not configured")
	}
	body, _ := json.Marshal(map[string]any{"model": p.Model, "messages": []map[string]string{{"role": "system", "content": "You are a governed retrieval planner. Never invent evidence or tools."}, {"role": "user", "content": prompt}}, "temperature": 0})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(p.BaseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	client := p.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("planner HTTP %d", resp.StatusCode)
	}
	var decoded struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", err
	}
	if len(decoded.Choices) == 0 {
		return "", errors.New("planner returned no choices")
	}
	return decoded.Choices[0].Message.Content, nil
}
func stripFence(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "```json")
	value = strings.TrimPrefix(value, "```")
	value = strings.TrimSuffix(value, "```")
	return strings.TrimSpace(value)
}

func decodePlan(raw []byte) (RetrievalPlan, error) {
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return RetrievalPlan{}, err
	}
	return RetrievalPlan{Queries: coerceStrings(value["queries"]), Entities: coerceStrings(value["entities"]), Temporal: coerceBool(value["temporal"]), Stop: coerceBool(value["stop"])}, nil
}
func coerceStrings(value any) []string {
	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) != "" {
			return []string{v}
		}
	case []any:
		out := []string{}
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}
func coerceBool(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(strings.TrimSpace(v), "true")
	}
	return false
}
