package router

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dashimaki/laputa/governance"
)

type actorKey struct{}

func WithActor(ctx context.Context, actor governance.ActorRole) context.Context {
	return context.WithValue(ctx, actorKey{}, actor)
}

func actorFrom(ctx context.Context) governance.ActorRole {
	if v, ok := ctx.Value(actorKey{}).(governance.ActorRole); ok {
		return v
	}
	return governance.ActorUser
}

// GovernanceBackend adapts governance.Engine to the unified CRUD Backend.
type GovernanceBackend struct {
	Engine   *governance.Engine
	Governed *governance.GovernedService
}

// NewGovernanceBackend wraps a governance engine as a CRUD backend.
func NewGovernanceBackend(engine *governance.Engine) *GovernanceBackend {
	return &GovernanceBackend{Engine: engine}
}

// Write stores section data. Keys use the form section:01-identity.
func (g *GovernanceBackend) Write(ctx context.Context, key, value string, meta map[string]any) (string, error) {
	section, err := parseSectionKey(key)
	if err != nil {
		return "", err
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(value), &data); err != nil {
		return "", fmt.Errorf("parse section value: %w", err)
	}
	if meta != nil {
		data["_meta"] = meta
	}

	if g.Governed != nil {
		err = g.Governed.Mutate(ctx, governance.MutationRequest{
			Section: section,
			Action:  "write",
			Actor:   actorFrom(ctx),
			Reason:  "v1 compatibility write",
			Data:    data,
		})
	} else {
		err = g.Engine.SetSection(ctx, section, data)
	}
	if err != nil {
		return "", err
	}
	return key, nil
}

// Read returns section data for a section: key.
func (g *GovernanceBackend) Read(ctx context.Context, key string) (map[string]any, error) {
	section, err := parseSectionKey(key)
	if err != nil {
		return nil, err
	}

	data, err := g.Engine.GetSection(ctx, section)
	if err != nil {
		return nil, err
	}

	out := map[string]any{
		"key":   key,
		"value": data,
	}
	if meta, ok := data["_meta"].(map[string]any); ok {
		out["meta"] = meta
	}
	return out, nil
}

// List returns sections whose keys match prefix (default section:).
func (g *GovernanceBackend) List(ctx context.Context, prefix string, limit int) ([]map[string]any, error) {
	if prefix == "" {
		prefix = "section:"
	}
	if !strings.HasPrefix(prefix, "section:") {
		return nil, fmt.Errorf("governance list prefix must start with section:")
	}

	sectionPrefix := strings.TrimPrefix(prefix, "section:")
	sections, err := g.Engine.ListSections(ctx)
	if err != nil {
		return nil, err
	}

	var records []map[string]any
	for _, section := range sections {
		key := "section:" + string(section)
		if sectionPrefix != "" && !strings.HasPrefix(string(section), sectionPrefix) {
			continue
		}
		data, err := g.Engine.GetSection(ctx, section)
		if err != nil {
			return nil, err
		}
		record := map[string]any{
			"key":   key,
			"value": data,
		}
		if meta, ok := data["_meta"].(map[string]any); ok {
			record["meta"] = meta
		}
		records = append(records, record)
		if limit > 0 && len(records) >= limit {
			break
		}
	}
	return records, nil
}

// Forget clears a section by resetting it to empty data.
func (g *GovernanceBackend) Forget(ctx context.Context, key string) (bool, error) {
	section, err := parseSectionKey(key)
	if err != nil {
		return false, err
	}

	if g.Governed != nil {
		err = g.Governed.Mutate(ctx, governance.MutationRequest{
			Section: section,
			Action:  "write",
			Actor:   actorFrom(ctx),
			Reason:  "v1 compatibility delete",
			Data:    map[string]any{},
		})
	} else {
		err = g.Engine.SetSection(ctx, section, map[string]any{})
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func parseSectionKey(key string) (governance.SectionName, error) {
	if !strings.HasPrefix(key, "section:") {
		return "", fmt.Errorf("invalid governance key: %s", key)
	}
	name := strings.TrimPrefix(key, "section:")
	if _, ok := governance.SectionRegistry[governance.SectionName(name)]; !ok {
		return "", fmt.Errorf("unknown section: %s", name)
	}
	return governance.SectionName(name), nil
}
