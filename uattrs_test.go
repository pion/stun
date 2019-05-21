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
	if (UnknownAttributes{}).String() != "<nil>" {
		t.Error("bad blank stirng")
	}
	if err := a.AddTo(m); err != nil {
		t.Error(err)
	}
	t.Run("GetFrom", func(t *testing.T) {
		attrs := make(UnknownAttributes, 10)
		if err := attrs.GetFrom(m); err != nil {
			t.Error(err)
		}
		for i, at := range *a {
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
