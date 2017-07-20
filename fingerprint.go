package stun

import (
	"fmt"
	"hash/crc32"
)

// FingerprintAttr represents FINGERPRINT attribute.
//
// https://tools.ietf.org/html/rfc5389#section-15.5
type FingerprintAttr byte

// CRCMismatch represents CRC check error.
type CRCMismatch struct {
	Expected uint32
	Actual   uint32
}

func (m CRCMismatch) Error() string {
	return fmt.Sprintf("CRC mismatch: %x (expected) != %x (actual)",
		m.Expected,
		m.Actual,
	)
}

// Fingerprint is shorthand for FingerprintAttr.
//
// Example:
//
//  m := New()
//  Fingerprint.AddTo(m)
var Fingerprint FingerprintAttr

const (
	fingerprintXORValue uint32 = 0x5354554e
	fingerprintSize            = 4 // 32 bit
)

// FingerprintValue returns CRC-32 of b XOR-ed by 0x5354554e.
//
// The value of the attribute is computed as the CRC-32 of the STUN message
// up to (but excluding) the FINGERPRINT attribute itself, XOR'ed with
// the 32-bit value 0x5354554e (the XOR helps in cases where an
// application packet is also using CRC-32 in it).
func FingerprintValue(b []byte) uint32 {
	return crc32.ChecksumIEEE(b) ^ fingerprintXORValue // XOR
}

// AddTo adds fingerprint to message.
func (FingerprintAttr) AddTo(m *Message) error {
	l := m.Length
	// length in header should include size of fingerprint attribute
	m.Length += fingerprintSize + attributeHeaderSize // increasing length
	m.WriteLength()                                   // writing Length to Raw
	b := make([]byte, fingerprintSize)
	val := FingerprintValue(m.Raw)
	bin.PutUint32(b, val)
	m.Length = l
	m.Add(AttrFingerprint, b)
	return nil
}

// Check reads fingerprint value from m and checks it, returning error if any.
// Can return *AttrLengthErr, ErrAttributeNotFound, and *CRCMismatch.
func (FingerprintAttr) Check(m *Message) error {
	b, err := m.Get(AttrFingerprint)
	if err != nil {
		return err
	}
	if len(b) != fingerprintSize {
		return &AttrLengthErr{
			Expected: fingerprintSize,
			Got:      len(b),
			Attr:     AttrFingerprint,
		}
	}
	val := bin.Uint32(b)
	attrStart := len(m.Raw) - (fingerprintSize + attributeHeaderSize)
	expected := FingerprintValue(m.Raw[:attrStart])
	if expected != val {
		return &CRCMismatch{Expected: expected, Actual: val}
	}
	return nil
}
