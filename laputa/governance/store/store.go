// Package redis provides a Redis-backed implementation of Laputa's SectionStore.
// It uses go-redis directly to avoid pulling in the full mempalace-go-redis module
// and its heavy ML/ONNX dependencies.
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	laputa "github.com/dashimaki/laputa/governance"
	goredis "github.com/redis/go-redis/v9"
)

// SectionName is an alias for laputa.SectionName to satisfy the interface locally.
type SectionName = laputa.SectionName

// Store implements a simple JSON-per-section backend on Redis.
type Store struct {
	addr     string
	client   *goredis.Client
	prefix   string
	mu       sync.RWMutex
	ctx      context.Context
}

// Options configures the Redis store.
type Options struct {
	Addr   string // e.g. "localhost:6379"
	DB     int
	Prefix string // key prefix, default "laputa:section:"
}

// New creates a Redis-backed SectionStore.
func New(ctx context.Context, opts Options) (*Store, error) {
	if opts.Prefix == "" {
		opts.Prefix = "laputa:section:"
	}
	client := goredis.NewClient(&goredis.Options{
		Addr: opts.Addr,
		DB:   opts.DB,
	})
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping %s: %w", opts.Addr, err)
	}
	return &Store{
		addr:   opts.Addr,
		client: client,
		prefix: opts.Prefix,
		ctx:    ctx,
	}, nil
}

// sectionKey returns the Redis key for a section.
func (s *Store) sectionKey(section SectionName) string {
	return s.prefix + string(section)
}

// Read returns the current data for a section.
func (s *Store) Read(ctx context.Context, section SectionName) (map[string]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := s.sectionKey(section)
	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == goredis.Nil {
			return make(map[string]any), nil
		}
		return nil, fmt.Errorf("read section %s: %w", section, err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal section %s: %w", section, err)
	}
	return result, nil
}

// Write replaces the entire section data.
func (s *Store) Write(ctx context.Context, section SectionName, data map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal section %s: %w", section, err)
	}
	key := s.sectionKey(section)
	if err := s.client.Set(ctx, key, out, 0).Err(); err != nil {
		return fmt.Errorf("write section %s: %w", section, err)
	}
	return nil
}

// Patch updates a specific path within a section (dot-notation).
func (s *Store) Patch(ctx context.Context, section SectionName, path string, value any) error {
	data, err := s.Read(ctx, section)
	if err != nil {
		return err
	}
	if err := setPath(data, path, value); err != nil {
		return fmt.Errorf("patch section %s path %s: %w", section, path, err)
	}
	return s.Write(ctx, section, data)
}

// Delete removes a specific path within a section.
func (s *Store) Delete(ctx context.Context, section SectionName, path string) error {
	data, err := s.Read(ctx, section)
	if err != nil {
		return err
	}
	if err := deletePath(data, path); err != nil {
		return fmt.Errorf("delete section %s path %s: %w", section, path, err)
	}
	return s.Write(ctx, section, data)
}

// List returns all section names currently stored.
func (s *Store) List(ctx context.Context) ([]SectionName, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sections []SectionName
	iter := s.client.Scan(ctx, 0, s.prefix+"*", 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		name := strings.TrimPrefix(key, s.prefix)
		sections = append(sections, SectionName(name))
	}
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("list sections: %w", err)
	}
	return sections, nil
}

// Exists checks if a section exists.
func (s *Store) Exists(ctx context.Context, section SectionName) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	n, err := s.client.Exists(ctx, s.sectionKey(section)).Result()
	if err != nil {
		return false, fmt.Errorf("check section %s: %w", section, err)
	}
	return n == 1, nil
}

// Close closes the Redis connection.
func (s *Store) Close() error {
	return s.client.Close()
}

// ---- path helpers copied from laputa.go ----

func setPath(data map[string]any, path string, value any) error {
	parts := splitPath(path)
	current := data
	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
			return nil
		}
		next, ok := current[part].(map[string]any)
		if !ok {
			next = make(map[string]any)
			current[part] = next
		}
		current = next
	}
	return nil
}

func deletePath(data map[string]any, path string) error {
	parts := splitPath(path)
	current := data
	for i, part := range parts {
		if i == len(parts)-1 {
			delete(current, part)
			return nil
		}
		next, ok := current[part].(map[string]any)
		if !ok {
			return fmt.Errorf("path not found: %s", path)
		}
		current = next
	}
	return nil
}

func splitPath(path string) []string {
	var parts []string
	var current string
	for _, r := range path {
		if r == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
