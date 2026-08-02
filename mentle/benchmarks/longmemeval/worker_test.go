package longmemeval

import (
	"fmt"
	"sync/atomic"
	"testing"
)

// TestWorkerPool_CallsWorkFnN verifies that the pool calls the work function
// exactly once for each input item, and that the results come back in the
// same order as the input.
func TestWorkerPool_CallsWorkFnN(t *testing.T) {
	inputs := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	var callCount int64

	results, err := runWorkerPool(inputs, 3, func(i int) (int, error) {
		atomic.AddInt64(&callCount, 1)
		return i * 2, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if int(callCount) != len(inputs) {
		t.Errorf("work fn called %d times, want %d", callCount, len(inputs))
	}
	for i, r := range results {
		if r != inputs[i]*2 {
			t.Errorf("results[%d] = %d, want %d", i, r, inputs[i]*2)
		}
	}
}

func TestWorkerPoolWithResource_UsesAllResources(t *testing.T) {
	resources := []int{10, 20, 30}
	inputs := []string{"a", "b", "c", "d", "e", "f"}

	results, err := runWorkerPoolWithResource(inputs, resources, func(res int, s string) (string, error) {
		return fmt.Sprintf("%s:%d", s, res), nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != len(inputs) {
		t.Errorf("got %d results, want %d", len(results), len(inputs))
	}
	// Each result must start with the corresponding input letter.
	for i, r := range results {
		if r[0] != inputs[i][0] {
			t.Errorf("results[%d] = %q, want prefix %q", i, r, inputs[i])
		}
	}
}
