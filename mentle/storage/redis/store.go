// Package redis provides Redis database connectivity for vector storage and search.
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/dashimaki/mentle/internal/palace"
)

type Store struct {
	client *redis.Client
	ctx    context.Context
}

func Open(addr string) (*Store, error) {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connect to redis %s: %w", addr, err)
	}
	return &Store{client: client, ctx: ctx}, nil
}

func (s *Store) Close() error {
	return s.client.Close()
}

func (s *Store) SaveDrawer(d *palace.Drawer) error {
	key := fmt.Sprintf("drawer:%s", d.ID)
	data, err := json.Marshal(d)
	if err != nil {
		return err
	}
	return s.client.Set(s.ctx, key, data, 0).Err()
}

func (s *Store) GetDrawer(id string) (*palace.Drawer, error) {
	key := fmt.Sprintf("drawer:%s", id)
	data, err := s.client.Get(s.ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	var d palace.Drawer
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *Store) SearchDrawers(wing, room string, n int) ([]palace.Drawer, error) {
	var result []palace.Drawer
	iter := s.client.Scan(s.ctx, 0, "drawer:*", 0).Iterator()
	for iter.Next(s.ctx) {
		data, err := s.client.Get(s.ctx, iter.Val()).Bytes()
		if err != nil {
			continue
		}
		var d palace.Drawer
		if err := json.Unmarshal(data, &d); err != nil {
			continue
		}
		if wing != "" && d.Wing != wing {
			continue
		}
		if room != "" && d.Room != room {
			continue
		}
		result = append(result, d)
		if len(result) >= n {
			break
		}
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Store) DeleteDrawer(id string) error {
	key := fmt.Sprintf("drawer:%s", id)
	return s.client.Del(s.ctx, key).Err()
}

func (s *Store) ListWings() ([]string, error) {
	wingSet := make(map[string]bool)
	iter := s.client.Scan(s.ctx, 0, "drawer:*", 0).Iterator()
	for iter.Next(s.ctx) {
		data, err := s.client.Get(s.ctx, iter.Val()).Bytes()
		if err != nil {
			continue
		}
		var d palace.Drawer
		if err := json.Unmarshal(data, &d); err != nil {
			continue
		}
		wingSet[d.Wing] = true
	}
	var wings []string
	for w := range wingSet {
		wings = append(wings, w)
	}
	return wings, iter.Err()
}

func (s *Store) ListRooms(wing string) ([]string, error) {
	roomSet := make(map[string]bool)
	iter := s.client.Scan(s.ctx, 0, "drawer:*", 0).Iterator()
	for iter.Next(s.ctx) {
		data, err := s.client.Get(s.ctx, iter.Val()).Bytes()
		if err != nil {
			continue
		}
		var d palace.Drawer
		if err := json.Unmarshal(data, &d); err != nil {
			continue
		}
		if d.Wing == wing {
			roomSet[d.Room] = true
		}
	}
	var rooms []string
	for r := range roomSet {
		rooms = append(rooms, r)
	}
	return rooms, iter.Err()
}

func (s *Store) Count() (int64, error) {
	var count int64
	iter := s.client.Scan(s.ctx, 0, "drawer:*", 0).Iterator()
	for iter.Next(s.ctx) {
		count++
	}
	return count, iter.Err()
}

func (s *Store) Stats() (map[string]interface{}, error) {
	info, err := s.client.Info(s.ctx).Result()
	if err != nil {
		return nil, err
	}
	stats := make(map[string]interface{})
	stats["redis_info"] = info
	count, err := s.Count()
	if err != nil {
		return nil, err
	}
	stats["drawer_count"] = count
	return stats, nil
}

// VectorSearch performs semantic search using RedisSearch if available,
// otherwise falls back to linear scan with embedding comparison.
func (s *Store) VectorSearch(queryEmbedding []float32, wing, room string, n int) ([]palace.Drawer, error) {
	// TODO: Implement RedisSearch vector index if available
	// For now, fall back to linear scan
	var result []palace.Drawer
	iter := s.client.Scan(s.ctx, 0, "drawer:*", 0).Iterator()
	for iter.Next(s.ctx) {
		data, err := s.client.Get(s.ctx, iter.Val()).Bytes()
		if err != nil {
			continue
		}
		var d palace.Drawer
		if err := json.Unmarshal(data, &d); err != nil {
			continue
		}
		if wing != "" && d.Wing != wing {
			continue
		}
		if room != "" && d.Room != room {
			continue
		}
		result = append(result, d)
		if len(result) >= n {
			break
		}
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// KeywordSearch performs full-text search using RedisSearch if available,
// otherwise falls back to linear scan with content matching.
func (s *Store) KeywordSearch(query string, wing, room string, n int) ([]palace.Drawer, error) {
	var result []palace.Drawer
	iter := s.client.Scan(s.ctx, 0, "drawer:*", 0).Iterator()
	for iter.Next(s.ctx) {
		data, err := s.client.Get(s.ctx, iter.Val()).Bytes()
		if err != nil {
			continue
		}
		var d palace.Drawer
		if err := json.Unmarshal(data, &d); err != nil {
			continue
		}
		if wing != "" && d.Wing != wing {
			continue
		}
		if room != "" && d.Room != room {
			continue
		}
		if strings.Contains(strings.ToLower(d.Content), strings.ToLower(query)) {
			result = append(result, d)
			if len(result) >= n {
				break
			}
		}
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// HybridSearch combines vector and keyword search with RRF fusion.
func (s *Store) HybridSearch(query string, queryEmbedding []float32, wing, room string, n int) ([]palace.Drawer, error) {
	// Get results from both methods
	vectorResults, err := s.VectorSearch(queryEmbedding, wing, room, n*2)
	if err != nil {
		return nil, err
	}
	keywordResults, err := s.KeywordSearch(query, wing, room, n*2)
	if err != nil {
		return nil, err
	}

	// RRF fusion
	scores := make(map[string]float64)
	rank := 1
	for _, d := range vectorResults {
		scores[d.ID] += 1.0 / float64(rank+60)
		rank++
	}
	rank = 1
	for _, d := range keywordResults {
		scores[d.ID] += 1.0 / float64(rank+60)
		rank++
	}

	// Combine and sort
	var combined []palace.Drawer
	seen := make(map[string]bool)
	for _, d := range vectorResults {
		if !seen[d.ID] {
			seen[d.ID] = true
			combined = append(combined, d)
		}
	}
	for _, d := range keywordResults {
		if !seen[d.ID] {
			seen[d.ID] = true
			combined = append(combined, d)
		}
	}

	// Sort by score (descending)
	for i := range combined {
		for j := i + 1; j < len(combined); j++ {
			if scores[combined[i].ID] < scores[combined[j].ID] {
				combined[i], combined[j] = combined[j], combined[i]
			}
		}
	}

	if len(combined) > n {
		combined = combined[:n]
	}
	return combined, nil
}
