package router

import (
	"context"
	"testing"
)

func TestRouteSectionPrefix(t *testing.T) {
	gov := &mockBackend{name: "governance"}
	r := &Router{Governance: gov, Mentle: &mockBackend{name: "mentle"}}

	backend, err := r.Route("section:01-identity")
	if err != nil {
		t.Fatalf("route: %v", err)
	}
	if backend != gov {
		t.Error("expected governance backend")
	}
}

func TestRouteMemoryPrefix(t *testing.T) {
	mem := &mockBackend{name: "mentle"}
	r := &Router{Governance: &mockBackend{name: "governance"}, Mentle: mem}

	backend, err := r.Route("memory:test")
	if err != nil {
		t.Fatalf("route: %v", err)
	}
	if backend != mem {
		t.Error("expected mentle backend")
	}
}

func TestRouteUnknownPrefix(t *testing.T) {
	r := &Router{
		Governance: &mockBackend{name: "governance"},
		Mentle:     &mockBackend{name: "mentle"},
	}

	_, err := r.Route("foo:bar")
	if err == nil {
		t.Fatal("expected error for unknown prefix")
	}
}

type mockBackend struct {
	name string
}

func (m *mockBackend) Write(ctx context.Context, key, value string, meta map[string]any) (string, error) {
	return key, nil
}

func (m *mockBackend) Read(ctx context.Context, key string) (map[string]any, error) {
	return map[string]any{"key": key}, nil
}

func (m *mockBackend) List(ctx context.Context, prefix string, limit int) ([]map[string]any, error) {
	return nil, nil
}

func (m *mockBackend) Forget(ctx context.Context, key string) (bool, error) {
	return true, nil
}
