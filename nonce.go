package stun

import "errors"

// NewNonce returns *Nonce with provided value.
func NewNonce(nonce string) *Nonce {
	return &Nonce{
		Raw: []byte(nonce),
	}
}

// Nonce represents NONCE attribute.
//
// https://tools.ietf.org/html/rfc5389#section-15.8
type Nonce struct {
	Raw []byte
}

func (n Nonce) String() string {
	return string(n.Raw)
}

const maxNonceB = 763

// ErrNonceTooBig means that NONCE value is bigger that 763 bytes.
var ErrNonceTooBig = errors.New("NONCE value bigger than 763 bytes")

// AddTo adds NONCE to message.
func (n *Nonce) AddTo(m *Message) error {
	if len(n.Raw) > maxNonceB {
		return ErrNonceTooBig
	}
	m.Add(AttrNonce, n.Raw)
	return nil
}

// GetFrom gets NONCE from message.
func (n *Nonce) GetFrom(m *Message) error {
	v, err := m.Get(AttrNonce)
	if err != nil {
		return err
	}
	n.Raw = v
	return nil
}
