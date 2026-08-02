// Package govector provides a GoVector-based vector store implementation.
// GoVector is a pure Go HNSW vector database for embeddings storage.
package govector

import (
	"github.com/DotNetAge/govector/core"
)

type Store struct {
	collection *core.Collection
	storage    *core.Storage
}

type SearchResult struct {
	ID      string
	Score   float32
	Payload map[string]any
}

func NewStore(dbPath string, dimension int) (*Store, error) {
	storage, err := core.NewStorage(dbPath)
	if err != nil {
		return nil, err
	}
	col, err := core.NewCollection("mempalace_drawers", dimension, core.Cosine, storage, true)
	if err != nil {
		return nil, err
	}
	return &Store{collection: col, storage: storage}, nil
}

func (s *Store) Add(id string, vector []float32, payload map[string]any) error {
	return s.collection.Upsert([]core.PointStruct{{
		ID:      id,
		Vector:  vector,
		Payload: core.Payload(payload),
	}})
}

// Point is a pre-constructed vector with its payload.
type Point struct {
	ID      string
	Vector  []float32
	Payload map[string]any
}

// AddBatch stores multiple points in a single upsert call.
func (s *Store) AddBatch(points []Point) error {
	ps := make([]core.PointStruct, len(points))
	for i, p := range points {
		ps[i] = core.PointStruct{
			ID:      p.ID,
			Vector:  p.Vector,
			Payload: core.Payload(p.Payload),
		}
	}
	return s.collection.Upsert(ps)
}

func (s *Store) Search(query []float32, limit int, filter map[string]any) ([]SearchResult, error) {
	hasIn, hasNin := hasInOrNinFilter(filter)

	// If we have $in filters, we need to fetch more results since govector
	// can't do OR natively — we fetch a larger set and post-filter.
	searchLimit := limit
	if hasIn {
		searchLimit = max(
			// Fetch more to account for OR filtering.
			limit*10, 1000)
	}

	var cf *core.Filter
	if len(filter) > 0 {
		cf = &core.Filter{
			Must: buildConditions(filter),
		}
	}
	points, err := s.collection.Search(query, cf, searchLimit)
	if err != nil {
		return nil, err
	}

	// Convert to our SearchResult type first.
	results := make([]SearchResult, len(points))
	for i, p := range points {
		results[i] = SearchResult{
			ID:      p.ID,
			Score:   p.Score,
			Payload: p.Payload,
		}
	}

	// Apply $in post-filter (govector can't do OR natively).
	if hasIn {
		results = applyInFilter(results, filter)
	}

	// Apply $nin post-filter.
	if hasNin {
		results = applyNinFilter(results, filter)
	}

	// Trim to requested limit.
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// buildConditions converts a filter map into govector conditions.
// Supports: exact match, range (gt/gte/lt/lte/eq), $in (OR of exact).
// $nin is applied as a post-filter in Search().
func buildConditions(filter map[string]any) []core.Condition {
	var conditions []core.Condition
	for key, val := range filter {
		if key == "$and" || key == "$or" {
			continue // Handled separately if needed.
		}
		if condMap, ok := val.(map[string]any); ok {
			conditions = append(conditions, buildCondition(key, condMap)...)
		} else {
			conditions = append(conditions, core.Condition{
				Key:   key,
				Type:  core.MatchTypeExact,
				Match: core.MatchValue{Value: val},
			})
		}
	}
	return conditions
}

// buildCondition handles structured filter values like $in, $nin, range.
func buildCondition(key string, filter map[string]any) []core.Condition {
	var conditions []core.Condition

	// $in → expand to multiple exact conditions (OR logic via MustNot on negation).
	// govector's Filter.Must uses AND, so we handle $in by checking each value
	// as separate conditions combined with OR at the collection level.
	// Since govector doesn't support OR natively, we expand $in into a single
	// condition that checks all values and let the collection handle it.
	if inVals, ok := filter["$in"].([]any); ok && len(inVals) > 0 {
		// Build individual exact-match conditions.
		// govector's collection.Search handles multiple Must conditions as AND,
		// so for $in we need a different approach: pass through as-is and
		// handle OR logic in the wrapper's post-filter.
		// For now, we create a special marker — the actual OR filtering
		// is done via post-filter in Search().
	}

	// $nin → post-filter only (handled in Search).

	// Range conditions.
	rangeVal := &core.RangeValue{}
	hasRange := false

	if val, ok := filter["gt"]; ok {
		rangeVal.GT = val
		hasRange = true
	}
	if val, ok := filter["gte"]; ok {
		rangeVal.GTE = val
		hasRange = true
	}
	if val, ok := filter["lt"]; ok {
		rangeVal.LT = val
		hasRange = true
	}
	if val, ok := filter["lte"]; ok {
		rangeVal.LTE = val
		hasRange = true
	}

	if hasRange {
		conditions = append(conditions, core.Condition{
			Key:   key,
			Type:  core.MatchTypeRange,
			Range: rangeVal,
		})
	}

	if val, ok := filter["eq"]; ok {
		conditions = append(conditions, core.Condition{
			Key:   key,
			Type:  core.MatchTypeExact,
			Match: core.MatchValue{Value: val},
		})
	}

	return conditions
}

// hasInOrNinFilter checks if the filter contains $in or $nin operators.
func hasInOrNinFilter(filter map[string]any) (hasIn, hasNin bool) {
	for _, val := range filter {
		if condMap, ok := val.(map[string]any); ok {
			if _, has := condMap["$in"]; has {
				hasIn = true
			}
			if _, has := condMap["$nin"]; has {
				hasNin = true
			}
		}
	}
	return hasIn, hasNin
}

// applyInFilter keeps only results where the field matches ANY of the $in values.
func applyInFilter(results []SearchResult, filter map[string]any) []SearchResult {
	// Collect all $in rules: field → set of allowed values.
	inRules := make(map[string]map[string]struct{})
	hasEmptyIn := false
	for key, val := range filter {
		if condMap, ok := val.(map[string]any); ok {
			if inVals, ok := condMap["$in"].([]any); ok {
				if len(inVals) == 0 {
					hasEmptyIn = true
				}
				allowed := make(map[string]struct{}, len(inVals))
				for _, v := range inVals {
					if s, ok := v.(string); ok {
						allowed[s] = struct{}{}
					}
				}
				inRules[key] = allowed
			}
		}
	}

	// Empty $in = match nothing.
	if hasEmptyIn || len(inRules) == 0 {
		return nil
	}

	var filtered []SearchResult
	for _, r := range results {
		match := true
		for field, allowedSet := range inRules {
			if fieldVal, ok := r.Payload[field].(string); ok {
				if _, found := allowedSet[fieldVal]; !found {
					match = false
					break
				}
			}
		}
		if match {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// applyNinFilter removes results that match any $nin values.
func applyNinFilter(results []SearchResult, filter map[string]any) []SearchResult {
	ninRules := make(map[string]map[string]struct{})
	for key, val := range filter {
		if condMap, ok := val.(map[string]any); ok {
			if ninVals, ok := condMap["$nin"].([]any); ok {
				exclude := make(map[string]struct{}, len(ninVals))
				for _, v := range ninVals {
					if s, ok := v.(string); ok {
						exclude[s] = struct{}{}
					}
				}
				ninRules[key] = exclude
			}
		}
	}

	if len(ninRules) == 0 {
		return results
	}

	var filtered []SearchResult
	for _, r := range results {
		excluded := false
		for field, excludeSet := range ninRules {
			if fieldVal, ok := r.Payload[field].(string); ok {
				if _, found := excludeSet[fieldVal]; found {
					excluded = true
					break
				}
			}
		}
		if !excluded {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func (s *Store) Close() error {
	if s.storage != nil {
		return s.storage.Close()
	}
	return nil
}

func (s *Store) Delete(id string) error {
	_, err := s.collection.Delete([]string{id}, nil)
	return err
}

func (s *Store) ListAll(limit int) ([]SearchResult, error) {
	zeroVector := make([]float32, s.collection.VectorLen)
	results, err := s.collection.Search(zeroVector, nil, limit)
	if err != nil {
		return nil, err
	}
	var searchResults []SearchResult
	for _, r := range results {
		searchResults = append(searchResults, SearchResult{
			ID:      r.ID,
			Score:   r.Score,
			Payload: r.Payload,
		})
	}
	return searchResults, nil
}
