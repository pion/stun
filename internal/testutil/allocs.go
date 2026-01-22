// SPDX-FileCopyrightText: 2026 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

// Package testutil contains helpers and utilities for writing tests
package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ShouldNotAllocate fails if f allocates.
func ShouldNotAllocate(t *testing.T, f func()) {
	t.Helper()

	if Race {
		t.Skip("skip while running with -race")

		return
	}
	assert.Zero(t, testing.AllocsPerRun(10, f))
}
