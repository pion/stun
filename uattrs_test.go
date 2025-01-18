// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package stun

import (
	"testing"
)

func TestUnknownAttributes(t *testing.T) {
	msg := new(Message)
	attr := &UnknownAttributes{
		AttrDontFragment,
		AttrChannelNumber,
	}
	if attr.String() != "DONT-FRAGMENT, CHANNEL-NUMBER" {
		t.Error("bad String:", attr)
	}
	if (UnknownAttributes{}).String() != "<nil>" {
		t.Error("bad blank string")
	}
	if err := attr.AddTo(msg); err != nil {
		t.Error(err)
	}
	t.Run("GetFrom", func(t *testing.T) {
		attrs := make(UnknownAttributes, 10)
		if err := attrs.GetFrom(msg); err != nil {
			t.Error(err)
		}
		for i, at := range *attr {
			if at != attrs[i] {
				t.Error("expected", at, "!=", attrs[i])
			}
		}
		mBlank := new(Message)
		if err := attrs.GetFrom(mBlank); err == nil {
			t.Error("should error")
		}
		mBlank.Add(AttrUnknownAttributes, []byte{1, 2, 3})
		if err := attrs.GetFrom(mBlank); err == nil {
			t.Error("should error")
		}
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
