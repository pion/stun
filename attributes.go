package stun

import (
	"encoding/binary"
	"net"

	"github.com/pkg/errors"
)

// blank is just blank string and exists just because it is ugly to keep it
// in code.
const blank = ""

func (m *Message) getAttrValue(t AttrType) []byte {
	return m.Attributes.Get(t).Value
}

// AddSoftwareBytes adds SOFTWARE attribute with value from byte slice.
func (m *Message) AddSoftwareBytes(software []byte) {
	m.Add(AttrSoftware, software)
}

// AddSoftware adds SOFTWARE attribute with value from string.
func (m *Message) AddSoftware(software string) {
	m.Add(AttrSoftware, []byte(software))
}

// GetSoftwareBytes returns SOFTWARE attribute value in byte slice.
// If not found, returns nil.
func (m *Message) GetSoftwareBytes() []byte {
	return m.Attributes.Get(AttrSoftware).Value
}

// GetSoftware returns SOFTWARE attribute value in string.
// If not found, returns blank string.
func (m *Message) GetSoftware() string {
	v := m.GetSoftwareBytes()
	if len(v) == 0 {
		return blank
	}
	return string(v)
}

// Address family values.
const (
	FamilyIPv4 byte = 0x01
	FamilyIPv6 byte = 0x02
)

// AddXORMappedAddress adds XOR MAPPED ADDRESS attribute to message.
func (m *Message) AddXORMappedAddress(ip net.IP, port int) {
	// X-Port is computed by taking the mapped port in host byte order,
	// XOR’ing it with the most significant 16 bits of the magic cookie, and
	// then the converting the result to network byte order.
	value := make([]byte, 32+128)
	value[0] = 0 // first 8 bits are zeroes
	family := FamilyIPv4
	if len(ip) == net.IPv6len {
		family = FamilyIPv6
	}
	binary.BigEndian.PutUint16(value[0:2], uint16(family))
	port ^= magicCookie >> 16
	binary.BigEndian.PutUint16(value[2:4], uint16(port))
	xorValue := make([]byte, 128)
	binary.BigEndian.PutUint32(xorValue[0:4], magicCookie)
	copy(xorValue[4:], m.TransactionID[:])
	xorBytes(value[4:4+len(ip)], ip, xorValue)

	m.Add(AttrXORMappedAddress, value[:4+len(ip)])
}

func (m *Message) allocBuffer(size int) []byte {
	capacity := len(m.buf.B) + size
	m.grow(capacity)
	m.buf.B = m.buf.B[:capacity]
	return m.buf.B[len(m.buf.B)-size:]
}

// GetXORMappedAddress returns ip, port from attribute and error if any.
// Value for ip is valid until Message is released or underlying buffer is
// corrupted.
func (m *Message) GetXORMappedAddress() (net.IP, int, error) {
	// X-Port is computed by taking the mapped port in host byte order,
	// XOR’ing it with the most significant 16 bits of the magic cookie, and
	// then the converting the result to network byte order.
	v := m.getAttrValue(AttrXORMappedAddress)
	if len(v) == 0 {
		return nil, 0, errors.Wrap(ErrAttributeNotFound, "not found")
	}
	family := byte(binary.BigEndian.Uint16(v[0:2]))
	if family != FamilyIPv6 && family != FamilyIPv4 {
		err := errors.Wrapf(ErrAttributeDecodeError, "bad family %d", family)
		return nil, 0, err
	}
	ipLen := net.IPv4len
	if family == FamilyIPv6 {
		ipLen = net.IPv6len
	}
	ip := net.IP(m.allocBuffer(ipLen))
	port := int(binary.BigEndian.Uint16(v[2:4])) ^ (magicCookie >> 16)
	xorValue := make([]byte, 128)
	binary.BigEndian.PutUint32(xorValue[0:4], magicCookie)
	copy(xorValue[4:], m.TransactionID[:])
	xorBytes(ip, v[4:], xorValue)
	return ip, port, nil
}

// shouldBeLess panics with err if s is not less than n characters.
func shouldBeLess(s string, n int, err error) {
	if len(s) >= n {
		panic(err)
	}
}

// constants for ERROR-CODE encoding.
const (
	errorCodeReasonStart = 4
	errorCodeClassByte   = 2
	errorCodeNumberByte  = 3
	errorCodeReasonMax   = 128
	errorCodeReasonMaxB  = 763
	errorCodeModulo      = 100
)

// AddErrorCode adds ERROR-CODE attribute to message.
//
// The reason phrase MUST be a UTF-8 [RFC 3629] encoded
// sequence of less than 128 characters (which can be as long as 763
// bytes).
func (m *Message) AddErrorCode(code int, reason string) {
	shouldBeLess(reason, errorCodeReasonMax, ErrErrorTooLong)
	value := make([]byte,
		errorCodeReasonStart, errorCodeReasonMaxB+errorCodeReasonStart,
	)
	number := byte(code % errorCodeModulo) // error code modulo 100
	class := byte(code / errorCodeModulo)  // hundred digit
	value[errorCodeClassByte] = class
	value[errorCodeNumberByte] = number
	value = append(value, reason...)
	m.Add(AttrErrorCode, value)
}

// GetErrorCode returns ERROR-CODE code, reason and decode error if any.
func (m *Message) GetErrorCode() (int, []byte, error) {
	v := m.getAttrValue(AttrErrorCode)
	if len(v) == 0 {
		return 0, nil, errors.Wrap(ErrAttributeNotFound, "not found")
	}
	var (
		class  = uint16(v[errorCodeClassByte])
		number = uint16(v[errorCodeNumberByte])
		code   = int(class*errorCodeModulo + number)
		reason = v[errorCodeReasonStart:]
	)
	return code, reason, nil
}

var (
	// ErrAttributeNotFound means that there is no such attribute.
	ErrAttributeNotFound Error = "Attribute not found"

	// ErrAttributeDecodeError means that agent is unable to decode value.
	ErrAttributeDecodeError Error = "Attribute decode error"

	// ErrErrorTooLong means that error reason is too long
	ErrErrorTooLong Error = "Error reason is too long"
)
