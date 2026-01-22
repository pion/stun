// SPDX-FileCopyrightText: 2026 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package stun

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkMessage_GetNotFound(b *testing.B) {
	m := New()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m.Get(AttrRealm) //nolint:errcheck,gosec
	}
}

func BenchmarkMessage_Get(b *testing.B) {
	m := New()
	m.Add(AttrUsername, []byte{1, 2, 3, 4, 5, 6, 7})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m.Get(AttrUsername) //nolint:errcheck,gosec
	}
}

func TestRawAttribute_AddTo(t *testing.T) {
	v := []byte{1, 2, 3, 4}
	m, err := Build(RawAttribute{
		Type:  AttrData,
		Value: v,
	})
	assert.NoError(t, err)
	gotV, gotErr := m.Get(AttrData)
	assert.NoError(t, gotErr)
	assert.True(t, bytes.Equal(gotV, v), "value mismatch")
}

func TestMessage_GetNoAllocs(t *testing.T) {
	msg := New()
	NewSoftware("c").AddTo(msg) //nolint:errcheck,gosec
	msg.WriteHeader()

	t.Run("Default", func(t *testing.T) {
		allocs := testing.AllocsPerRun(10, func() {
			msg.Get(AttrSoftware) //nolint:errcheck,gosec
		})
		assert.Zero(t, allocs, "allocated memory, but should not")
	})
	t.Run("Not found", func(t *testing.T) {
		allocs := testing.AllocsPerRun(10, func() {
			msg.Get(AttrOrigin) //nolint:errcheck,gosec
		})
		assert.Zero(t, allocs, "allocated memory, but should not")
	})
}

func TestPadding(t *testing.T) {
	tt := []struct {
		in, out int
	}{
		{4, 4},   // 0
		{2, 4},   // 1
		{5, 8},   // 2
		{8, 8},   // 3
		{11, 12}, // 4
		{1, 4},   // 5
		{3, 4},   // 6
		{6, 8},   // 7
		{7, 8},   // 8
		{0, 0},   // 9
		{40, 40}, // 10
	}
	for i, c := range tt {
		got := nearestPaddedValueLength(c.in)
		assert.Equal(t, c.out, got, "[%d]: padd(%d)", i, c.in)
	}
}

func TestAttrTypeRange(t *testing.T) {
	for _, a := range []AttrType{
		AttrPriority,
		AttrErrorCode,
		AttrUseCandidate,
		AttrEvenPort,
		AttrRequestedAddressFamily,
	} {
		a := a
		t.Run(a.String(), func(t *testing.T) {
			a := a
			assert.True(t, a.Required(), "should be required")
			assert.False(t, a.Optional(), "should be required")
		})
	}
	for _, a := range []AttrType{
		AttrSoftware,
		AttrICEControlled,
		AttrOrigin,
	} {
		a := a
		t.Run(a.String(), func(t *testing.T) {
			assert.False(t, a.Required(), "should be optional")
			assert.True(t, a.Optional(), "should be optional")
		})
	}
}

func TestAttrTypeKnown(t *testing.T) {
	// All Attributes in attrNames should be known
	for attr := range attrNames() {
		assert.True(t, attr.Known())
	}

	assert.False(t, AttrType(0xFFFF).Known()) // Known
}
