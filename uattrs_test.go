package stun

import (
	"testing"
)

func TestUnknownAttributes(t *testing.T) {
	m := new(Message)
	a := &UnknownAttributes{
		AttrDontFragment,
		AttrChannelNumber,
	}
	if a.String() != "DONT-FRAGMENT, CHANNEL-NUMBER" {
		t.Error("bad String:", a)
	}
	if err := a.AddTo(m); err != nil {
		t.Error(err)
	}
	t.Run("AppendFrom", func(t *testing.T) {
		attrs := make(UnknownAttributes, 10)
		if err := attrs.GetFrom(m); err != nil {
			t.Error(err)
		}
		for i, at := range *a {
			if at != attrs[i] {
				t.Error("expected", at, "!=", attrs[i])
			}
		}
	})
}

func BenchmarkUnknownAttributes(b *testing.B) {
	m := new(Message)
	a := UnknownAttributes{
		AttrDontFragment,
		AttrChannelNumber,
		AttrRealm,
		AttrMessageIntegrity,
	}
	b.Run("AddTo", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if err := a.AddTo(m); err != nil {
				b.Fatal(err)
			}
			m.Reset()
		}
	})
	b.Run("GetFrom", func(b *testing.B) {
		b.ReportAllocs()
		if err := a.AddTo(m); err != nil {
			b.Fatal(err)
		}
		attrs := make(UnknownAttributes, 0, 10)
		for i := 0; i < b.N; i++ {
			if err := attrs.GetFrom(m); err != nil {
				b.Fatal(err)
			}
			attrs = attrs[:0]
		}
	})
}
