package crud

import (
	"context"
	"testing"

	"github.com/dashimaki/garden/internal/router"
)

type mockBackend struct {
	name string
}

func (m *mockBackend) Write(ctx context.Context, key, value string, meta map[string]any) (string, error) {
	return m.name + ":" + key, nil
}

func (m *mockBackend) Read(ctx context.Context, key string) (map[string]any, error) {
	return map[string]any{"backend": m.name, "key": key}, nil
}

func (m *mockBackend) List(ctx context.Context, prefix string, limit int) ([]map[string]any, error) {
	return []map[string]any{{"backend": m.name, "prefix": prefix}}, nil
}

func (m *mockBackend) Forget(ctx context.Context, key string) (bool, error) {
	return true, nil
}

func TestHandlerRoutesSectionKeys(t *testing.T) {
	gov := &mockBackend{name: "governance"}
	mem := &mockBackend{name: "mentle"}
	h := &Handler{
		Router: &router.Router{Governance: gov, Mentle: mem},
	}
	ctx := context.Background()

	id, err := h.Write(ctx, "section:01-identity", `{"role":"agent"}`, nil)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if id != "governance:section:01-identity" {
		t.Errorf("write id = %q, want governance:section:01-identity", id)
	}

	read, err := h.Read(ctx, "section:01-identity")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if read["backend"] != "governance" {
		t.Errorf("read backend = %v, want governance", read["backend"])
	}

	list, err := h.List(ctx, "section:", 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0]["backend"] != "governance" {
		t.Errorf("list = %v, want governance backend", list)
	}

	ok, err := h.Forget(ctx, "section:01-identity")
	if err != nil || !ok {
		t.Fatalf("forget: ok=%v err=%v", ok, err)
	}
}

func TestHandlerRoutesMemoryKeys(t *testing.T) {
	gov := &mockBackend{name: "governance"}
	mem := &mockBackend{name: "mentle"}
	h := &Handler{
		Router: &router.Router{Governance: gov, Mentle: mem},
	}
	ctx := context.Background()

	id, err := h.Write(ctx, "memory:diary:1", "hello", nil)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if id != "mentle:memory:diary:1" {
		t.Errorf("write id = %q, want mentle:memory:diary:1", id)
	}

	read, err := h.Read(ctx, "memory:diary:1")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if read["backend"] != "mentle" {
		t.Errorf("read backend = %v, want mentle", read["backend"])
	}
}

func TestHandlerUnknownPrefix(t *testing.T) {
	h := &Handler{
		Router: &router.Router{
			Governance: &mockBackend{name: "governance"},
			Mentle:     &mockBackend{name: "mentle"},
		},
	}
	ctx := context.Background()

	_, err := h.Write(ctx, "unknown:key", "value", nil)
	if err == nil {
		t.Fatal("expected error for unknown prefix")
	}
}
