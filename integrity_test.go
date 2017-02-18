package stun

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestMessageIntegrity_AddTo_Simple(t *testing.T) {
	i := NewLongtermIntegrity("user", "realm", "pass")
	expected, err := hex.DecodeString("8493fbc53ba582fb4c044c456bdc40eb")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(expected, i) {
		t.Error(&IntegrityErr{
			Expected: expected,
			Actual:   i,
		})
	}
	t.Run("Check", func(t *testing.T) {
		m := new(Message)
		m.WriteHeader()
		if err := i.AddTo(m); err != nil {
			t.Error(err)
		}
		m.WriteHeader()
		dM := new(Message)
		dM.Raw = m.Raw
		if err := dM.Decode(); err != nil {
			t.Error(err)
		}
		if err := i.Check(dM); err != nil {
			t.Error(err)
		}
		m.Raw[3] = m.Raw[3] + 12 // HMAC now invalid
		if err, ok := i.Check(dM).(*IntegrityErr); !ok {
			t.Error(err, "should be *IntegrityErr")
		}
	})
}

func TestMessageIntegrity(t *testing.T) {
	m := new(Message)
	//NewSoftware("software")
	i := MessageIntegrity("password")
	m.WriteHeader()
	if err := i.AddTo(m); err != nil {
		t.Error(err)
	}
	_, err := m.Get(AttrMessageIntegrity)
	if err != nil {
		t.Error(err)
	}
}

func BenchmarkMessageIntegrity_AddTo(b *testing.B) {
	m := new(Message)
	integrity := MessageIntegrity("password")
	m.WriteHeader()
	b.ReportAllocs()
	b.SetBytes(int64(len(m.Raw)))
	for i := 0; i < b.N; i++ {
		m.WriteHeader()
		if err := integrity.AddTo(m); err != nil {
			b.Error(err)
		}
		m.Reset()
	}
}
func BenchmarkMessageIntegrity_Check(b *testing.B) {
	m := new(Message)
	NewSoftware("software").AddTo(m)
	integrity := MessageIntegrity("password")
	b.ReportAllocs()
	m.WriteHeader()
	b.SetBytes(int64(len(m.Raw)))
	if err := integrity.AddTo(m); err != nil {
		b.Error(err)
	}
	m.WriteLength()
	for i := 0; i < b.N; i++ {
		if err := integrity.Check(m); err != nil {
			b.Fatal(err)
		}
	}
}
