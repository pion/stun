// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

// Package testutil contains helpers and utilities for writing tests
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
		t.Errorf("Allocations detected: %f", a)
	}
}
