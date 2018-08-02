package testutil

import (
	"testing"
)

// ShouldNotAllocate fails if f allocates.
func ShouldNotAllocate(t *testing.T, f func()) {
	if Race {
		t.Skip("skip while running with -race")
		return
	}
	if a := testing.AllocsPerRun(10, f); a > 0 {
		t.Errorf("allocations detected: %f", a)
	}
}
