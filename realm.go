package stun

import "errors"

// NewRealm returns Realm with provided value.
// Must be SASL-prepared.
func NewRealm(realm string) Realm {
	// TODO: use sasl
	return Realm(realm)
}

// Realm represents REALM attribute.
//
// https://tools.ietf.org/html/rfc5389#section-15.7
type Realm []byte

func (n Realm) String() string {
	return string(n)
}

const maxRealmB = 763

// ErrRealmTooBig means that REALM value is bigger that 763 bytes.
var ErrRealmTooBig = errors.New("REALM value bigger than 763 bytes")

// AddTo adds NONCE to message.
func (n Realm) AddTo(m *Message) error {
	if len(n) > maxRealmB {
		return ErrRealmTooBig
	}
	m.Add(AttrRealm, n)
	return nil
}

// GetFrom gets REALM from message.
func (n *Realm) GetFrom(m *Message) error {
	v, err := m.Get(AttrRealm)
	if err != nil {
		return err
	}
	*n = v
	return nil
}
