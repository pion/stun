// SPDX-FileCopyrightText: 2026 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package stun

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnknownAttributes(t *testing.T) {
	msg := new(Message)
	attr := &UnknownAttributes{
		AttrDontFragment,
		AttrChannelNumber,
	}
	assert.Equal(t, "DONT-FRAGMENT, CHANNEL-NUMBER", attr.String())
	assert.Equal(t, "<nil>", (UnknownAttributes{}).String())
	assert.NoError(t, attr.AddTo(msg))
	t.Run("GetFrom", func(t *testing.T) {
		attrs := make(UnknownAttributes, 10)
		assert.NoError(t, attrs.GetFrom(msg))
		for i, at := range *attr {
			assert.Equal(t, at, attrs[i])
		}
		mBlank := new(Message)
		assert.Error(t, attrs.GetFrom(mBlank))
		mBlank.Add(AttrUnknownAttributes, []byte{1, 2, 3})
		assert.Error(t, attrs.GetFrom(mBlank))
	})
}

func BenchmarkUnknownAttributes(b *testing.B) {
	msg := new(Message)
	attr := UnknownAttributes{
		AttrDontFragment,
		AttrChannelNumber,
		AttrRealm,
		AttrMessageIntegrity,
	}
	b.Run("AddTo", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if err := attr.AddTo(msg); err != nil {
				b.Fatal(err)
			}
			msg.Reset()
		}
	})
	b.Run("GetFrom", func(b *testing.B) {
		b.ReportAllocs()
		if err := attr.AddTo(msg); err != nil {
			b.Fatal(err)
		}
		attrs := make(UnknownAttributes, 0, 10)
		for i := 0; i < b.N; i++ {
			if err := attrs.GetFrom(msg); err != nil {
				b.Fatal(err)
			}
			attrs = attrs[:0]
		}
	})
}
