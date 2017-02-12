package stun

import "testing"

func TestUnknownAttributes(t *testing.T) {
	m := new(Message)
	a := &UnknownAttributes{
		Types: []AttrType{
			AttrDontFragment,
			AttrChannelNumber,
		},
	}
	if a.String() != "DONT-FRAGMENT, CHANNEL-NUMBER" {
		t.Error("bad String:", a)
	}
	if err := a.AddTo(m); err != nil {
		t.Error(err)
	}
	t.Run("AppendFrom", func(t *testing.T) {
		attrs := new(UnknownAttributes)
		if err := attrs.GetFrom(m); err != nil {
			t.Error(err)
		}
		for i, at := range a.Types {
			if at != attrs.Types[i] {
				t.Error("expected", at, "!=", attrs.Types[i])
			}
		}
	})
}

func BenchmarkUnknownAttributes(b *testing.B) {
	m := new(Message)
	a := &UnknownAttributes{
		Types: []AttrType{
			AttrDontFragment,
			AttrChannelNumber,
			AttrRealm,
			AttrMessageIntegrity,
		},
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
		attrs := new(UnknownAttributes)
		for i := 0; i < b.N; i++ {
			if err := attrs.GetFrom(m); err != nil {
				b.Fatal(err)
			}
			attrs.Types = attrs.Types[:0]
		}
	})
}
