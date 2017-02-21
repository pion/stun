package stun

import (
	"strings"
	"testing"
)

func TestSoftware_GetFrom(t *testing.T) {
	m := New()
	v := "Client v0.0.1"
	m.Add(AttrSoftware, []byte(v))
	m.WriteHeader()

	m2 := &Message{
		Raw: make([]byte, 0, 256),
	}
	software := new(Software)
	if _, err := m2.ReadFrom(m.reader()); err != nil {
		t.Error(err)
	}
	if err := software.GetFrom(m); err != nil {
		t.Fatal(err)
	}
	if software.String() != v {
		t.Errorf("Expected %q, got %q.", v, software)
	}

	sAttr, ok := m.Attributes.Get(AttrSoftware)
	if !ok {
		t.Error("sowfware attribute should be found")
	}
	s := sAttr.String()
	if !strings.HasPrefix(s, "SOFTWARE:") {
		t.Error("bad string representation", s)
	}
}

func TestSoftware_AddTo_Invalid(t *testing.T) {
	m := New()
	s :=  make(Software, 1024)
	if err := s.AddTo(m); err != ErrSoftwareTooBig {
		t.Errorf("AddTo should return %q, got: %v", ErrSoftwareTooBig, err)
	}
	if err := s.GetFrom(m); err != ErrAttributeNotFound {
		t.Errorf("GetFrom should return %q, got: %v", ErrAttributeNotFound, err)
	}
}

func TestSoftware_AddTo_Regression(t *testing.T) {
	// s.AddTo checked len(m.Raw) instead of len(s.Raw).
	m := &Message{Raw: make([]byte, 2048)}
	s := make(Software, 100)
	if err := s.AddTo(m); err != nil {
		t.Errorf("AddTo should return <nil>, got: %v", err)
	}
}
