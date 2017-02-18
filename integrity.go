package stun

import (
	"crypto/hmac"
	"crypto/md5" // #nosec
	"crypto/sha1"
	"errors"
	"fmt"
	"strings"
)

// separator for credentials.
const credentialsSep = ":"

// NewLongTermIntegrity returns new MessageIntegrity with key for long-term
// credentials. Password, username, and realm must be SASL-prepared.
func NewLongTermIntegrity(username, realm, password string) MessageIntegrity {
	// TODO: perform sasl prep.
	k := strings.Join(
		[]string{
			username,
			realm,
			password,
		},
		credentialsSep,
	)
	// #nosec
	h := md5.New()
	fmt.Fprint(h, k)
	return MessageIntegrity(h.Sum(nil))
}

// NewShortTermIntegrity returns new MessageIntegrity with key for short-term
// credentials. Password must be SASL-prepared.
func NewShortTermIntegrity(password string) MessageIntegrity {
	// TODO: perform sasl prep.
	return MessageIntegrity(password)
}

// MessageIntegrity represents MESSAGE-INTEGRITY attribute. AddTo and GetFrom
// methods will allocate memory for cryptographic functions. Zero-allocation
// version of MessageIntegrity is not implemented. Implementation and changes
// to it is subject to security review.
//
// https://tools.ietf.org/html/rfc5389#section-15.4
type MessageIntegrity []byte

// ErrFingerprintBeforeIntegrity means that FINGEPRINT attribute is already in
// message, so MESSAGE-INTEGRITY attribute cannot be added.
var ErrFingerprintBeforeIntegrity = errors.New(
	"FINGERPRINT before MESSAGE-INTEGRITY attribute",
)

func (i MessageIntegrity) String() string {
	return fmt.Sprintf("KEY: 0x%x", []byte(i))
}

const messageIntegritySize = 20

// AddTo adds MESSAGE-INTEGRITY attribute to message. Be advised, CPU
// and allocations costly, can be cause of DOS.
func (i MessageIntegrity) AddTo(m *Message) error {
	for _, a := range m.Attributes {
		// Message should not contain FINGERPRINT attribute
		// before MESSAGE-INTEGRITY.
		if a.Type == AttrFingerprint {
			return ErrFingerprintBeforeIntegrity
		}
	}
	// The text used as input to HMAC is the STUN message,
	// including the header, up to and including the attribute preceding the
	// MESSAGE-INTEGRITY attribute.
	l := m.Length
	// Adjusting m.Length to contain MESSAGE-INTEGRITY TLV.
	m.Length += messageIntegritySize + attributeHeaderSize
	m.WriteLength()        // writing length to m.Raw
	v := newHMAC(i, m.Raw) // calculating HMAC for adjusted m.Raw
	m.Length = l           // changing m.Length back
	m.Add(AttrMessageIntegrity, v)
	return nil
}

// IntegrityErr occurs when computed HMAC differs from expected.
type IntegrityErr struct {
	Expected []byte
	Actual   []byte
}

func (i *IntegrityErr) Error() string {
	return fmt.Sprintf(
		"Integrity check failed: 0x%x (expected) !- 0x%x (actual)",
		i.Expected, i.Actual,
	)
}

func newHMAC(key, message []byte) []byte {
	mac := hmac.New(sha1.New, key)
	_, err := mac.Write(message)
	if err != nil {
		panic(err)
	}
	return mac.Sum(nil)
}

// Check checks MESSAGE-INTEGRITY attribute. Be advised, CPU and allocations
// costly, can be cause of DOS.
func (i MessageIntegrity) Check(m *Message) error {
	v, err := m.Get(AttrMessageIntegrity)
	if err != nil {
		return err
	}

	// Adjusting length in header to match m.Raw that was
	// used when computing HMAC.
	l := m.Length
	afterIntegrity := false
	sizeReduced := int(messageIntegritySize + attributeHeaderSize)
	for _, a := range m.Attributes {
		if afterIntegrity {
			sizeReduced += nearestPaddedValueLength(int(a.Length))
			sizeReduced += attributeHeaderSize
		}
		if a.Type == AttrMessageIntegrity {
			afterIntegrity = true
		}
	}
	m.Length -= uint32(sizeReduced)
	expected := newHMAC(i, m.Raw[:m.Length+messageHeaderSize])
	m.Length = l
	m.WriteLength() // writing length back
	if !hmac.Equal(v, expected) {
		return &IntegrityErr{
			Expected: expected,
			Actual:   v,
		}
	}
	return nil
}
