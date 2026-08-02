// Package vectorstore defines the common interface for all vector database backends.
// Implementations include: govector (local HNSW), Redis, Qdrant, ChromaDB, LanceDB.
package vectorstore

import (
	"context"
	"fmt"
)

// Point represents a vector with metadata payload.
type Point struct {
	ID      string
	Vector  []float32
	Payload map[string]any
}

// SearchResult represents a single vector search result.
type SearchResult struct {
	ID      string
	Score   float32
	Payload map[string]any
}

// Store is the unified interface for all vector database backends.
type Store interface {
	// Search performs vector similarity search with optional filtering.
	Search(ctx context.Context, query []float32, limit int, filter map[string]any) ([]SearchResult, error)

	// Add inserts or updates a single vector point.
	Add(ctx context.Context, id string, vector []float32, payload map[string]any) error

	// AddBatch inserts or updates multiple vector points.
	AddBatch(ctx context.Context, points []Point) error

	// Delete removes a vector by ID.
	Delete(ctx context.Context, id string) error

	// ListAll returns all stored points up to limit.
	ListAll(ctx context.Context, limit int) ([]SearchResult, error)

	// Close releases resources.
	Close() error
}

// BackendType identifies the vector store backend.
type BackendType string

const (
	BackendGoVector BackendType = "govector"
	BackendRedis    BackendType = "redis"
	BackendQdrant   BackendType = "qdrant"
	BackendChroma   BackendType = "chroma"
	BackendLanceDB  BackendType = "lancedb"
)

// Config holds connection parameters for any backend.
type Config struct {
	Type       BackendType
	Addr       string // host:port or URL
	Collection string // collection/table name
	Dimension  int    // vector dimension
	APIKey     string // optional auth
}

// Open creates a Store based on Config.Type.
func Open(cfg Config) (Store, error) {
	switch cfg.Type {
	case BackendGoVector:
		return nil, fmt.Errorf("govector: use storage/govector.NewStore directly")
	case BackendRedis:
		return nil, fmt.Errorf("redis: use storage/redis.Open directly")
	case BackendQdrant:
		return nil, fmt.Errorf("qdrant: not yet implemented")
	case BackendChroma:
		return nil, fmt.Errorf("chroma: not yet implemented")
	case BackendLanceDB:
		return nil, fmt.Errorf("lancedb: not yet implemented")
	default:
		return nil, fmt.Errorf("unknown backend type: %s", cfg.Type)
	}
}
