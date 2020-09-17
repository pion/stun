// +build !js

package stun

import (
	"net"
	"testing"
)

func BenchmarkFingerprint_AddTo(b *testing.B) {
	b.ReportAllocs()
	m := new(Message)
	s := NewSoftware("software")
	addr := &XORMappedAddress{
		IP: net.IPv4(213, 1, 223, 5),
	}
	addAttr(b, m, addr)
	addAttr(b, m, s)
	b.SetBytes(int64(len(m.Raw)))
	for i := 0; i < b.N; i++ {
		Fingerprint.AddTo(m) // nolint:errcheck
		m.WriteLength()
		m.Length -= attributeHeaderSize + fingerprintSize
		m.Raw = m.Raw[:m.Length+messageHeaderSize]
		m.Attributes = m.Attributes[:len(m.Attributes)-1]
	}
}

func TestFingerprint_Check(t *testing.T) {
	m := new(Message)
	addAttr(t, m, NewSoftware("software"))
	m.WriteHeader()
	Fingerprint.AddTo(m) // nolint:errcheck
	m.WriteHeader()
	if err := Fingerprint.Check(m); err != nil {
		t.Error(err)
	}
	m.Raw[3]++
	if err := Fingerprint.Check(m); err == nil {
		t.Error("should error")
	}
}

func TestFingerprint_CheckBad(t *testing.T) {
	m := new(Message)
	addAttr(t, m, NewSoftware("software"))
	m.WriteHeader()
	if err := Fingerprint.Check(m); err == nil {
		t.Error("should error")
	}
	m.Add(AttrFingerprint, []byte{1, 2, 3})
	if !IsAttrSizeInvalid(Fingerprint.Check(m)) {
		t.Error("IsAttrSizeInvalid should be true")
	}
}

func BenchmarkFingerprint_Check(b *testing.B) {
	b.ReportAllocs()
	m := new(Message)
	s := NewSoftware("software")
	addr := &XORMappedAddress{
		IP: net.IPv4(213, 1, 223, 5),
	}
	addAttr(b, m, addr)
	addAttr(b, m, s)
	m.WriteHeader()
	Fingerprint.AddTo(m) // nolint:errcheck
	m.WriteHeader()
	b.SetBytes(int64(len(m.Raw)))
	for i := 0; i < b.N; i++ {
		if err := Fingerprint.Check(m); err != nil {
			b.Fatal(err)
		}
	}
}
