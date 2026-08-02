package facade

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dashimaki/mentle/internal/palace"
)

var ErrUnavailable = errors.New("mentle service unavailable")

// Write stores a memory record. Phase 1 will route memory: keys here.
func (s *Service) Write(ctx context.Context, key, content string, meta map[string]any) (string, error) {
	if s.Hybrid == nil {
		return "", ErrUnavailable
	}
	id := strings.TrimPrefix(key, "memory:")
	if id == "" {
		return "", errors.New("memory key is required")
	}
	drawer := palace.Drawer{ID: id, Content: content, Metadata: map[string]string{}}
	if value, ok := meta["wing"].(string); ok {
		drawer.Wing = value
	}
	if value, ok := meta["room"].(string); ok {
		drawer.Room = value
	}
	if value, ok := meta["source"].(string); ok {
		drawer.SourceFile = value
	}
	for k, value := range meta {
		if str, ok := value.(string); ok {
			drawer.Metadata[k] = str
		}
	}
	if err := s.Hybrid.Store(ctx, drawer); err != nil {
		return "", fmt.Errorf("store memory: %w", err)
	}
	return "memory:" + id, nil
}

// Read retrieves a memory record by key.
func (s *Service) Read(ctx context.Context, key string) (map[string]any, error) {
	if s.Searcher == nil {
		return nil, ErrUnavailable
	}
	id := strings.TrimPrefix(key, "memory:")
	drawers, err := s.Searcher.ListAll(ctx, 50000)
	if err != nil {
		return nil, err
	}
	for _, drawer := range drawers {
		if drawer.ID == id {
			return drawerRecord(drawer.ID, drawer.Content, drawer.Wing, drawer.Room, drawer.Metadata), nil
		}
	}
	return nil, fmt.Errorf("memory %q not found", id)
}

// List returns memory records matching a prefix.
func (s *Service) List(ctx context.Context, prefix string, limit int) ([]map[string]any, error) {
	if s.Searcher == nil {
		return nil, ErrUnavailable
	}
	drawers, err := s.Searcher.ListAll(ctx, limit)
	if err != nil {
		return nil, err
	}
	records := make([]map[string]any, 0, len(drawers))
	for _, drawer := range drawers {
		key := "memory:" + drawer.ID
		if strings.HasPrefix(key, prefix) {
			records = append(records, drawerRecord(drawer.ID, drawer.Content, drawer.Wing, drawer.Room, drawer.Metadata))
		}
	}
	return records, nil
}

// Forget removes a memory record by key.
func (s *Service) Forget(ctx context.Context, key string) (bool, error) {
	if s.Catalog != nil && strings.HasPrefix(key, "memory:") {
		var id string
		err := s.Catalog.db.QueryRowContext(ctx, `SELECT id FROM memories WHERE legacy_key=?`, key).Scan(&id)
		if err == nil {
			_, err = s.DeleteMemory(ctx, id, "legacy", "")
			return err == nil, err
		}
	}
	if s.Hybrid == nil {
		return false, ErrUnavailable
	}
	id := strings.TrimPrefix(key, "memory:")
	if err := s.Hybrid.Delete(ctx, id); err != nil {
		return false, err
	}
	return true, nil
}

func drawerRecord(id, content, wing, room string, metadata map[string]string) map[string]any {
	return map[string]any{"key": "memory:" + id, "value": content, "wing": wing, "room": room, "meta": metadata}
}
