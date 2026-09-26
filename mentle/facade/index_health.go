package facade

import govector "github.com/dashimaki/mentle/storage/govector"

// unavailableVectorStore is an inert diagnostic adapter for lexical-only
// services. It makes the existing IndexHealth vector probe report unavailable
// instead of dereferencing a nil store; it never creates a vector index.
type unavailableVectorStore struct{}

func (unavailableVectorStore) Search([]float32, int, map[string]any) ([]govector.SearchResult, error) {
	return nil, ErrUnavailable
}
func (unavailableVectorStore) Add(string, []float32, map[string]any) error { return ErrReadOnly }
func (unavailableVectorStore) AddBatch([]govector.Point) error             { return ErrReadOnly }
func (unavailableVectorStore) Delete(string) error                         { return ErrReadOnly }
func (unavailableVectorStore) ListAll(int) ([]govector.SearchResult, error) {
	return nil, ErrUnavailable
}
func (unavailableVectorStore) Close() error { return nil }
