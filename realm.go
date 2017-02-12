package stun

import "errors"

// NewRealm returns *Realm with provided value.
// Must be SASL-prepared.
func NewRealm(nonce string) *Realm {
	// TODO: use sasl
	return &Realm{
		Raw: []byte(nonce),
	}
}

// Realm represents REALM attribute.
//
// https://tools.ietf.org/html/rfc5389#section-15.8
type Realm struct {
	Raw []byte
}

func (n Realm) String() string {
	return string(n.Raw)
}

const maxRealmB = 763

// ErrRealmTooBig means that REALM value is bigger that 763 bytes.
var ErrRealmTooBig = errors.New("REALM value bigger than 763 bytes")

// AddTo adds NONCE to message.
func (n *Realm) AddTo(m *Message) error {
	if len(n.Raw) > maxRealmB {
		return ErrRealmTooBig
	}
	m.Add(AttrRealm, n.Raw)
	return nil
}

// GetFrom gets REALM from message.
func (n *Realm) GetFrom(m *Message) error {
	v, err := m.Get(AttrRealm)
	if err != nil {
		return err
	}
	n.Raw = v
	return nil
}
