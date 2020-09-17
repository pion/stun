package stun

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestMessageIntegrity_AddTo_Simple(t *testing.T) {
	i := NewLongTermIntegrity("user", "realm", "pass")
	expected, err := hex.DecodeString("8493fbc53ba582fb4c044c456bdc40eb")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(expected, i) {
		t.Error(ErrIntegrityMismatch)
	}
	t.Run("Check", func(t *testing.T) {
		m := new(Message)
		m.WriteHeader()
		if err := i.AddTo(m); err != nil {
			t.Error(err)
		}
		NewSoftware("software").AddTo(m) // nolint:errcheck
		m.WriteHeader()
		dM := new(Message)
		dM.Raw = m.Raw
		if err := dM.Decode(); err != nil {
			t.Error(err)
		}
		if err := i.Check(dM); err != nil {
			t.Error(err)
		}
		dM.Raw[24] += 12 // HMAC now invalid
		if i.Check(dM) == nil {
			t.Error("should be invalid")
		}
	})
}

func TestMessageIntegrityWithFingerprint(t *testing.T) {
	m := new(Message)
	m.TransactionID = [TransactionIDSize]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	m.WriteHeader()
	NewSoftware("software").AddTo(m) // nolint:errcheck
	i := NewShortTermIntegrity("pwd")
	if i.String() != "KEY: 0x707764" {
		t.Error("bad string", i)
	}
	if err := i.Check(m); err == nil {
		t.Error("should error")
	}
	if err := i.AddTo(m); err != nil {
		t.Fatal(err)
	}
	if err := Fingerprint.AddTo(m); err != nil {
		t.Fatal(err)
	}
	if err := i.Check(m); err != nil {
		t.Fatal(err)
	}
	m.Raw[24] = 33
	if err := i.Check(m); err == nil {
		t.Fatal("mismatch expected")
	}
}

func TestMessageIntegrity(t *testing.T) {
	m := new(Message)
	i := NewShortTermIntegrity("password")
	m.WriteHeader()
	if err := i.AddTo(m); err != nil {
		t.Error(err)
	}
	_, err := m.Get(AttrMessageIntegrity)
	if err != nil {
		t.Error(err)
	}
}

func TestMessageIntegrityBeforeFingerprint(t *testing.T) {
	m := new(Message)
	m.WriteHeader()
	Fingerprint.AddTo(m) // nolint:errcheck
	i := NewShortTermIntegrity("password")
	if err := i.AddTo(m); err == nil {
		t.Error("should error")
	}
}

func BenchmarkMessageIntegrity_AddTo(b *testing.B) {
	m := new(Message)
	integrity := NewShortTermIntegrity("password")
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
	m.Raw = make([]byte, 0, 1024)
	NewSoftware("software").AddTo(m) // nolint:errcheck
	integrity := NewShortTermIntegrity("password")
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
