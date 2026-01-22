// SPDX-FileCopyrightText: 2026 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

//go:build debug
// +build debug

package stun

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAttrOverflowErr_Error(t *testing.T) {
	err := AttrOverflowErr{
		Got:  100,
		Max:  50,
		Type: AttrLifetime,
	}
	assert.Equal(t, "incorrect length of LIFETIME attribute: 100 exceeds maximum 50", err.Error())
}

func TestAttrLengthErr_Error(t *testing.T) {
	err := AttrLengthErr{
		Attr:     AttrErrorCode,
		Expected: 15,
		Got:      99,
	}
	assert.Equal(t, "incorrect length of ERROR-CODE attribute: got 99, expected 15", err.Error())
}
