package stun

import (
	"hash/crc32"

	"github.com/pkg/errors"
)

// A Fingerprint attribute MAY be present in all STUN messages.  The
// value of the attribute is computed as the CRC-32 of the STUN message
// up to (but excluding) the FINGERPRINT attribute itself, XOR'ed with
// the 32-bit value 0x5354554e (the XOR helps in cases where an
// application packet is also using CRC-32 in it)
type Fingerprint struct {
	Fingerprint uint32
}

const (
	fingerprintXOR    uint32 = 0x5354554e
	fingerprintLength uint16 = 4
)

func calculateFingerprint(b []byte) uint32 {
	return crc32.ChecksumIEEE(b) ^ fingerprintXOR
}

// Pack with Fingerprint
func (s *Fingerprint) Pack(message *Message) error {
	prevLen := message.Length
	message.Length += attrHeaderLength + fingerprintLength
	message.CommitLength()
	v := make([]byte, fingerprintLength)
	enc.PutUint32(v, calculateFingerprint(message.Raw))
	message.Length = prevLen

	message.AddAttribute(AttrFingerprint, v)
	return nil
}

//Unpack with Fingerprint
func (s *Fingerprint) Unpack(message *Message, rawAttribute *RawAttribute) error {

	s.Fingerprint = enc.Uint32(rawAttribute.Value)

	expected := calculateFingerprint(message.Raw[:rawAttribute.Offset])

	if expected != s.Fingerprint {
		return errors.Errorf("fingerprint mismatch %v != %v (expected)", s.Fingerprint, expected)
	}
	return nil
}
