package stun

import (
	"strings"
	"testing"
)

func TestRealm_GetFrom(t *testing.T) {
	m := New()
	v := "realm"
	m.Add(AttrRealm, []byte(v))
	m.WriteHeader()

	m2 := &Message{
		Raw: make([]byte, 0, 256),
	}
	r := new(Realm)
	if err := r.GetFrom(m2); err != ErrAttributeNotFound {
		t.Errorf("GetFrom should return %q, got: %v", ErrAttributeNotFound, err)
	}
	if _, err := m2.ReadFrom(m.reader()); err != nil {
		t.Error(err)
	}
	if err := r.GetFrom(m); err != nil {
		t.Fatal(err)
	}
	if r.String() != v {
		t.Errorf("Expected %q, got %q.", v, r)
	}

	rAttr, ok := m.Attributes.Get(AttrRealm)
	if !ok {
		t.Error("realm attribute should be found")
	}
	s := rAttr.String()
	if !strings.HasPrefix(s, "REALM:") {
		t.Error("bad string representation", s)
	}
}

func TestRealm_AddTo_Invalid(t *testing.T) {
	m := New()
	r := &Realm{
		Raw: make([]byte, 1024),
	}
	if err := r.AddTo(m); err != ErrRealmTooBig {
		t.Errorf("AddTo should return %q, got: %v", ErrRealmTooBig, err)
	}
	if err := r.GetFrom(m); err != ErrAttributeNotFound {
		t.Errorf("GetFrom should return %q, got: %v", ErrAttributeNotFound, err)
	}
}
