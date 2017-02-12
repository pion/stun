package stun

import (
	"strings"
	"testing"
)

func TestNonce_GetFrom(t *testing.T) {
	m := New()
	v := "example.org"
	m.Add(AttrNonce, []byte(v))
	m.WriteHeader()

	m2 := &Message{
		Raw: make([]byte, 0, 256),
	}
	nonce := new(Nonce)
	if _, err := m2.ReadFrom(m.reader()); err != nil {
		t.Error(err)
	}
	if err := nonce.GetFrom(m); err != nil {
		t.Fatal(err)
	}
	if nonce.String() != v {
		t.Errorf("Expected %q, got %q.", v, nonce)
	}

	nAttr, ok := m.Attributes.Get(AttrNonce)
	if !ok {
		t.Error("nonce attribute should be found")
	}
	s := nAttr.String()
	if !strings.HasPrefix(s, "NONCE:") {
		t.Error("bad string representation", s)
	}
}

func TestNonce_AddTo_Invalid(t *testing.T) {
	m := New()
	n := &Nonce{
		Raw: make([]byte, 1024),
	}
	if err := n.AddTo(m); err != ErrNonceTooBig {
		t.Errorf("AddTo should return %q, got: %v", ErrNonceTooBig, err)
	}
	if err := n.GetFrom(m); err != ErrAttributeNotFound {
		t.Errorf("GetFrom should return %q, got: %v", ErrAttributeNotFound, err)
	}
}

func TestNonce_AddTo(t *testing.T) {
	m := New()
	n := NewNonce("example.org")
	if err := n.AddTo(m); err != nil {
		t.Error(err)
	}
	v, err := m.Get(AttrNonce)
	if err != nil {
		t.Error(err)
	}
	if string(v) != "example.org" {
		t.Errorf("bad nonce %q", v)
	}
}
