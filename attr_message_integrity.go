package stun

import (
	"crypto/hmac"
	"crypto/sha1" // #nosec

	"github.com/pkg/errors"
)

// https://tools.ietf.org/html/rfc5389#section-15.4
// The MESSAGE-INTEGRITY attribute contains an HMAC-SHA1 [RFC2104] of
// the STUN message.  The MESSAGE-INTEGRITY attribute can be present in
// any STUN message type.  Since it uses the SHA1 hash, the HMAC will be
// 20 bytes.  The text used as input to HMAC is the STUN message,
// including the header, up to and including the attribute preceding the
// MESSAGE-INTEGRITY attribute.  With the exception of the FINGERPRINT
// attribute, which appears after MESSAGE-INTEGRITY, agents MUST ignore
// all other attributes that follow MESSAGE-INTEGRITY.

// Look into this
// https://tools.ietf.org/html/rfc7635#appendix-B

const (
	messageIntegrityLength = 20
)

// MessageIntegrity is struct represented MESSAGE-INTEGRITY attribute rfc5389#section-15.4
type MessageIntegrity struct {
	Key []byte
}

//MessageIntegrityCalculateHMAC returns hmac checksum
func MessageIntegrityCalculateHMAC(key, message []byte) ([]byte, error) {
	/* #nosec */
	mac := hmac.New(sha1.New, key)
	if _, err := mac.Write(message); err != nil {
		// Can we recover from this failure?
		return nil, errors.Wrap(err, "unable to create message integrity HMAC")
	}
	return mac.Sum(nil), nil
}

//Pack message with MessageIntegrity
func (m *MessageIntegrity) Pack(message *Message) error {
	prevLen := message.Length
	message.Length += attrHeaderLength + messageIntegrityLength
	message.CommitLength()
	v, err := MessageIntegrityCalculateHMAC(m.Key[:], message.Raw)
	if err != nil {
		return err
	}
	message.Length = prevLen

	message.AddAttribute(AttrMessageIntegrity, v)
	return nil
}

//Unpack copy from Key to rawAttribute.Value
func (m *MessageIntegrity) Unpack(message *Message, rawAttribute *RawAttribute) error {
	if len(rawAttribute.Value) != messageIntegrityLength {
		return errors.Errorf("MessageIntegrity bad key length %d", len(rawAttribute.Value))
	}
	copy(m.Key[:], rawAttribute.Value[:])
	return nil
}
