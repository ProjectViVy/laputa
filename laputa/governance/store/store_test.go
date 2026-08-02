package redis

import (
	"context"
	"testing"

	laputa "github.com/dashimaki/laputa/governance"
)

func TestRedisStore_InitializeAndRoundTrip(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, Options{Addr: "localhost:6379"})
	if err != nil {
		t.Skipf("redis not available: %v", err)
	}
	defer store.Close()

	// Use a unique prefix to avoid cross-test contamination.
	store.prefix = "laputa:test:section:"

	section := laputa.SectionName("01-identity")
	if err := store.Write(ctx, section, map[string]any{"role": "test-agent"}); err != nil {
		t.Fatalf("write: %v", err)
	}

	data, err := store.Read(ctx, section)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if data["role"] != "test-agent" {
		t.Errorf("expected role=test-agent, got %v", data["role"])
	}

	exists, err := store.Exists(ctx, section)
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if !exists {
		t.Errorf("expected section to exist")
	}

	sections, err := store.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	found := false
	for _, s := range sections {
		if s == section {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected section in list, got %v", sections)
	}

	// Cleanup
	_ = store.client.Del(ctx, store.sectionKey(section)).Err()
}

func TestRedisStore_PatchAndDelete(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, Options{Addr: "localhost:6379"})
	if err != nil {
		t.Skipf("redis not available: %v", err)
	}
	defer store.Close()
	store.prefix = "laputa:test:section:"

	section := laputa.SectionName("02-relationship")
	_ = store.Write(ctx, section, map[string]any{"relationships": []string{}})

	if err := store.Patch(ctx, section, "resonance.score", 85); err != nil {
		t.Fatalf("patch: %v", err)
	}
	data, _ := store.Read(ctx, section)
	score := data["resonance"].(map[string]any)["score"]
	// JSON numbers unmarshal to float64 by default; compare numerically.
	scoreFloat, ok := score.(float64)
	if !ok || scoreFloat != 85 {
		t.Errorf("expected score=85, got %v (type %T)", score, score)
	}

	if err := store.Delete(ctx, section, "resonance.score"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	data, _ = store.Read(ctx, section)
	if _, ok := data["resonance"].(map[string]any)["score"]; ok {
		t.Errorf("expected score deleted")
	}

	// Cleanup
	_ = store.client.Del(ctx, store.sectionKey(section)).Err()
}
