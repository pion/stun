package stun

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"
)

func TestMessageIntegrity_AddTo_Simple(t *testing.T) {
	i := NewLongTermIntegrity("user", "realm", "pass")
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
		NewSoftware("software").AddTo(m)
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
		if err, ok := i.Check(dM).(*IntegrityErr); !ok {
			t.Error(err, "should be *IntegrityErr")
		}
	})
}

func TestMessageIntegrityWithFingerprint(t *testing.T) {
	m := new(Message)
	m.TransactionID = [TransactionIDSize]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	m.WriteHeader()
	NewSoftware("software").AddTo(m)
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
	errStr := fmt.Sprintf("Integrity check failed: 0x%s (expected) !- 0x%s (actual)",
		"19985afb819c098acfe1c2771881227f14c70eaf",
		"ef9da0e0caf0b0e4ff321e7b56f1e114c802cb7e",
	)
	if err := i.Check(m); err.Error() != errStr {
		t.Fatal(err, "!=", errStr)
	}
}

func TestMessageIntegrity(t *testing.T) {
	m := new(Message)
	//NewSoftware("software")
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
	//NewSoftware("software")
	m.WriteHeader()
	Fingerprint.AddTo(m)
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
	NewSoftware("software").AddTo(m)
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
