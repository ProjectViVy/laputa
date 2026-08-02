package govector

import (
	"os"
	"testing"
)

func setupTestStore(t *testing.T) (*Store, func()) {
	t.Helper()
	dir := t.TempDir()
	store, err := NewStore(dir+"/vectors.db", 4)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Add test data with 4-dim vectors.
	testData := []struct {
		id      string
		vector  []float32
		payload map[string]any
	}{
		{"doc1", []float32{1, 0, 0, 0}, map[string]any{"wing": "backend", "room": "session-notes", "content": "notes about API"}},
		{"doc2", []float32{0, 1, 0, 0}, map[string]any{"wing": "backend", "room": "decisions", "content": "decided on REST"}},
		{"doc3", []float32{0, 0, 1, 0}, map[string]any{"wing": "frontend", "room": "session-notes", "content": "React components"}},
		{"doc4", []float32{0, 0, 0, 1}, map[string]any{"wing": "frontend", "room": "ideas", "content": "UI redesign"}},
		{"doc5", []float32{0.5, 0.5, 0, 0}, map[string]any{"wing": "infra", "room": "archive", "content": "old server config"}},
	}

	for _, td := range testData {
		if err := store.Add(td.id, td.vector, td.payload); err != nil {
			t.Fatalf("failed to add %s: %v", td.id, err)
		}
	}

	return store, func() { store.Close() }
}

func TestStore_Search_InFilter(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	query := []float32{1, 0, 0, 0}

	// No filter — should return all.
	results, err := store.Search(query, 10, nil)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if len(results) != 5 {
		t.Errorf("expected 5 results, got %d", len(results))
	}

	// Exact match filter.
	results, err = store.Search(query, 10, map[string]any{"room": "session-notes"})
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results for room=session-notes, got %d", len(results))
	}
}

func TestStore_Search_InOperator(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	query := []float32{1, 0, 0, 0}

	// $in: search across multiple rooms.
	filter := map[string]any{
		"room": map[string]any{
			"$in": []any{"session-notes", "decisions"},
		},
	}
	results, err := store.Search(query, 10, filter)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}

	// Should return doc1 (session-notes), doc2 (decisions), doc3 (session-notes).
	if len(results) != 3 {
		t.Errorf("expected 3 results for $in rooms, got %d: %v", len(results), resultIDs(results))
	}

	// Verify all returned docs have room in the allowed set.
	allowedRooms := map[string]bool{"session-notes": true, "decisions": true}
	for _, r := range results {
		room, _ := r.Payload["room"].(string)
		if !allowedRooms[room] {
			t.Errorf("unexpected room %q in $in results", room)
		}
	}
}

func TestStore_Search_InOperator_MultipleFields(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	query := []float32{1, 0, 0, 0}

	// $in on wing AND $in on room — both must match.
	filter := map[string]any{
		"wing": map[string]any{
			"$in": []any{"backend", "frontend"},
		},
		"room": map[string]any{
			"$in": []any{"session-notes"},
		},
	}
	results, err := store.Search(query, 10, filter)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}

	// Should return doc1 (backend + session-notes) and doc3 (frontend + session-notes).
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d: %v", len(results), resultIDs(results))
	}
}

func TestStore_Search_NinOperator(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	query := []float32{1, 0, 0, 0}

	// $nin: exclude archive and ideas.
	filter := map[string]any{
		"room": map[string]any{
			"$nin": []any{"archive", "ideas"},
		},
	}
	results, err := store.Search(query, 10, filter)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}

	// Should exclude doc4 (ideas) and doc5 (archive).
	if len(results) != 3 {
		t.Errorf("expected 3 results (excluding archive+ideas), got %d: %v", len(results), resultIDs(results))
	}

	excludedRooms := map[string]bool{"archive": true, "ideas": true}
	for _, r := range results {
		room, _ := r.Payload["room"].(string)
		if excludedRooms[room] {
			t.Errorf("excluded room %q found in $nin results", room)
		}
	}
}

func TestStore_Search_InAndNinCombined(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	query := []float32{1, 0, 0, 0}

	// Include backend + frontend, but exclude archive.
	filter := map[string]any{
		"wing": map[string]any{
			"$in": []any{"backend", "frontend"},
		},
		"room": map[string]any{
			"$nin": []any{"archive"},
		},
	}
	results, err := store.Search(query, 10, filter)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}

	// doc1 (backend, session-notes) ✓
	// doc2 (backend, decisions) ✓
	// doc3 (frontend, session-notes) ✓
	// doc4 (frontend, ideas) ✓
	// doc5 (infra, archive) ✗ — wing not in $in, room in $nin
	if len(results) != 4 {
		t.Errorf("expected 4 results, got %d: %v", len(results), resultIDs(results))
	}
}

func TestStore_Search_NinOnly(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	query := []float32{1, 0, 0, 0}

	// Just $nin, no $in.
	filter := map[string]any{
		"wing": map[string]any{
			"$nin": []any{"infra"},
		},
	}
	results, err := store.Search(query, 10, filter)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}

	// Should exclude doc5 (infra).
	if len(results) != 4 {
		t.Errorf("expected 4 results (excluding infra), got %d: %v", len(results), resultIDs(results))
	}
}

func TestStore_Search_LimitRespected(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	query := []float32{1, 0, 0, 0}

	filter := map[string]any{
		"wing": map[string]any{
			"$in": []any{"backend", "frontend", "infra"},
		},
	}
	results, err := store.Search(query, 2, filter)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}

	if len(results) > 2 {
		t.Errorf("limit not respected: got %d results, expected ≤ 2", len(results))
	}
}

func TestStore_Search_EmptyInValues(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	query := []float32{1, 0, 0, 0}

	// $in with no values should return nothing.
	filter := map[string]any{
		"room": map[string]any{
			"$in": []any{},
		},
	}
	results, err := store.Search(query, 10, filter)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}

	// Empty $in = no matches.
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty $in, got %d", len(results))
	}
}

func resultIDs(results []SearchResult) []string {
	ids := make([]string, len(results))
	for i, r := range results {
		ids[i] = r.ID
	}
	return ids
}

func TestApplyInFilter(t *testing.T) {
	results := []SearchResult{
		{ID: "1", Payload: map[string]any{"room": "alpha"}},
		{ID: "2", Payload: map[string]any{"room": "beta"}},
		{ID: "3", Payload: map[string]any{"room": "gamma"}},
	}

	filter := map[string]any{
		"room": map[string]any{"$in": []any{"alpha", "gamma"}},
	}

	filtered := applyInFilter(results, filter)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 results, got %d", len(filtered))
	}

	ids := map[string]bool{}
	for _, r := range filtered {
		ids[r.ID] = true
	}
	if !ids["1"] || !ids["3"] {
		t.Errorf("expected IDs 1 and 3, got %v", ids)
	}
}

func TestApplyNinFilter(t *testing.T) {
	results := []SearchResult{
		{ID: "1", Payload: map[string]any{"wing": "A"}},
		{ID: "2", Payload: map[string]any{"wing": "B"}},
		{ID: "3", Payload: map[string]any{"wing": "C"}},
	}

	filter := map[string]any{
		"wing": map[string]any{"$nin": []any{"B"}},
	}

	filtered := applyNinFilter(results, filter)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 results, got %d", len(filtered))
	}

	ids := map[string]bool{}
	for _, r := range filtered {
		ids[r.ID] = true
	}
	if !ids["1"] || !ids["3"] {
		t.Errorf("expected IDs 1 and 3, got %v", ids)
	}
	if ids["2"] {
		t.Error("ID 2 should be excluded by $nin")
	}
}

func TestHasInOrNinFilter(t *testing.T) {
	tests := []struct {
		name    string
		filter  map[string]any
		wantIn  bool
		wantNin bool
	}{
		{
			name:    "$in present",
			filter:  map[string]any{"room": map[string]any{"$in": []any{"a", "b"}}},
			wantIn:  true,
			wantNin: false,
		},
		{
			name:    "$nin present",
			filter:  map[string]any{"wing": map[string]any{"$nin": []any{"x"}}},
			wantIn:  false,
			wantNin: true,
		},
		{
			name:    "both present",
			filter:  map[string]any{"room": map[string]any{"$in": []any{"a"}}, "wing": map[string]any{"$nin": []any{"x"}}},
			wantIn:  true,
			wantNin: true,
		},
		{
			name:    "exact match only",
			filter:  map[string]any{"room": "alpha"},
			wantIn:  false,
			wantNin: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIn, gotNin := hasInOrNinFilter(tt.filter)
			if gotIn != tt.wantIn {
				t.Errorf("hasIn = %v, want %v", gotIn, tt.wantIn)
			}
			if gotNin != tt.wantNin {
				t.Errorf("hasNin = %v, want %v", gotNin, tt.wantNin)
			}
		})
	}
}

// Ensure store cleanup removes temp files.
func TestStore_Cleanup(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir+"/vectors.db", 4)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	store.Close()

	// Verify file exists.
	if _, err := os.Stat(dir + "/vectors.db"); os.IsNotExist(err) {
		t.Error("expected vectors.db to exist after close")
	}
}
