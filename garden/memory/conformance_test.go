package memory_test

import (
	"testing"

	"github.com/dashimaki/garden/memory/memorytest"
)

// The reference fake must satisfy the same conformance suite as real
// backends — otherwise the suite itself can't be trusted.
func TestFakeBackendConformance(t *testing.T) {
	memorytest.BackendConformance(t, memorytest.NewFakeFactory(memorytest.NewFakeStore()))
}
