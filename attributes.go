package stun

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"net"
	"strconv"
)

// AttrWriter wraps AddRaw method.
type AttrWriter interface {
	AddRaw(t AttrType, v []byte)
}

// AttrEncoder wraps Encode method.
type AttrEncoder interface {
	Encode(b []byte, m *Message) (AttrType, []byte, error)
}

// Attributes is list of message attributes.
type Attributes []RawAttribute

// Get returns first attribute from list by the type.
// If attribute is present the RawAttribute is returned and the
// boolean is true. Otherwise the returned RawAttribute will be
// empty and boolean will be false.
func (a Attributes) Get(t AttrType) (RawAttribute, bool) {
	for _, candidate := range a {
		if candidate.Type == t {
			return candidate, true
		}
	}
	return RawAttribute{}, false
}

// AttrType is attribute type.
type AttrType uint16

// Attributes from comprehension-required range (0x0000-0x7FFF).
const (
	AttrMappedAddress     AttrType = 0x0001 // MAPPED-ADDRESS
	AttrUsername          AttrType = 0x0006 // USERNAME
	AttrMessageIntegrity  AttrType = 0x0008 // MESSAGE-INTEGRITY
	AttrErrorCode         AttrType = 0x0009 // ERROR-CODE
	AttrUnknownAttributes AttrType = 0x000A // UNKNOWN-ATTRIBUTES
	AttrRealm             AttrType = 0x0014 // REALM
	AttrNonce             AttrType = 0x0015 // NONCE
	AttrXORMappedAddress  AttrType = 0x0020 // XOR-MAPPED-ADDRESS
)

// Attributes from comprehension-optional range (0x8000-0xFFFF).
const (
	AttrSoftware        AttrType = 0x8022 // SOFTWARE
	AttrAlternateServer AttrType = 0x8023 // ALTERNATE-SERVER
	AttrFingerprint     AttrType = 0x8028 // FINGERPRINT
)

// Attributes from RFC 5245 ICE.
const (
	AttrPriority       AttrType = 0x0024 // PRIORITY
	AttrUseCandidate   AttrType = 0x0025 // USE-CANDIDATE
	AttrICEControlled  AttrType = 0x8029 // ICE-CONTROLLED
	AttrICEControlling AttrType = 0x802A // ICE-CONTROLLING
)

// Attributes from RFC 5766 TURN.
const (
	AttrChannelNumber      AttrType = 0x000C // CHANNEL-NUMBER
	AttrLifetime           AttrType = 0x000D // LIFETIME
	AttrXORPeerAddress     AttrType = 0x0012 // XOR-PEER-ADDRESS
	AttrData               AttrType = 0x0013 // DATA
	AttrXORRelayedAddress  AttrType = 0x0016 // XOR-RELAYED-ADDRESS
	AttrEvenPort           AttrType = 0x0018 // EVEN-PORT
	AttrRequestedTransport AttrType = 0x0019 // REQUESTED-TRANSPORT
	AttrDontFragment       AttrType = 0x001A // DONT-FRAGMENT
	AttrReservationToken   AttrType = 0x0022 // RESERVATION-TOKEN
)

// Attributes from An Origin RawAttribute for the STUN Protocol.
const (
	AttrOrigin AttrType = 0x802F
)

// Value returns uint16 representation of attribute type.
func (t AttrType) Value() uint16 {
	return uint16(t)
}

var attrNames = map[AttrType]string{
	AttrMappedAddress:      "MAPPED-ADDRESS",
	AttrUsername:           "USERNAME",
	AttrErrorCode:          "ERROR-CODE",
	AttrMessageIntegrity:   "MESSAGE-INTEGRITY",
	AttrUnknownAttributes:  "UNKNOWN-ATTRIBUTES",
	AttrRealm:              "REALM",
	AttrNonce:              "NONCE",
	AttrXORMappedAddress:   "XOR-MAPPED-ADDRESS",
	AttrSoftware:           "SOFTWARE",
	AttrAlternateServer:    "ALTERNATE-SERVER",
	AttrFingerprint:        "FINGERPRINT",
	AttrPriority:           "PRIORITY",
	AttrUseCandidate:       "USE-CANDIDATE",
	AttrICEControlled:      "ICE-CONTROLLED",
	AttrICEControlling:     "ICE-CONTROLLING",
	AttrChannelNumber:      "CHANNEL-NUMBER",
	AttrLifetime:           "LIFETIME",
	AttrXORPeerAddress:     "XOR-PEER-ADDRESS",
	AttrData:               "DATA",
	AttrXORRelayedAddress:  "XOR-RELAYED-ADDRESS",
	AttrEvenPort:           "EVEN-PORT",
	AttrRequestedTransport: "REQUESTED-TRANSPORT",
	AttrDontFragment:       "DONT-FRAGMENT",
	AttrReservationToken:   "RESERVATION-TOKEN",
	AttrOrigin:             "ORIGIN",
}

func (t AttrType) String() string {
	s, ok := attrNames[t]
	if !ok {
		// just return hex representation of unknown attribute type
		return "0x" + strconv.FormatUint(uint64(t), 16)
	}
	return s
}

// RawAttribute is a Type-Length-Value (TLV) object that
// can be added to a STUN message.  Attributes are divided into two
// types: comprehension-required and comprehension-optional.  STUN
// agents can safely ignore comprehension-optional attributes they
// don't understand, but cannot successfully process a message if it
// contains comprehension-required attributes that are not
// understood.
type RawAttribute struct {
	Type   AttrType
	Length uint16 // ignored while encoding
	Value  []byte
}

// Encode implements AttrEncoder.
func (a *RawAttribute) Encode(m *Message) ([]byte, error) {
	return m.Raw, nil
}

// Decode implements AttrDecoder.
func (a *RawAttribute) Decode(v []byte, m *Message) error {
	a.Value = v
	a.Length = uint16(len(v))
	return nil
}

// Equal returns true if a == b.
func (a RawAttribute) Equal(b RawAttribute) bool {
	if a.Type != b.Type {
		return false
	}
	if a.Length != b.Length {
		return false
	}
	if len(b.Value) != len(a.Value) {
		return false
	}
	for i, v := range a.Value {
		if b.Value[i] != v {
			return false
		}
	}
	return true
}

func (a RawAttribute) String() string {
	return fmt.Sprintf("%s: %x", a.Type, a.Value)
}

// getAttrValue returns byte slice that represents attribute value,
// if there is no value attribute with shuck type,
// ErrAttributeNotFound is returned.
func (m *Message) getAttrValue(t AttrType) ([]byte, error) {
	v, ok := m.Attributes.Get(t)
	if !ok {
		return nil, ErrAttributeNotFound
	}
	return v.Value, nil
}

// AddSoftware adds SOFTWARE attribute with value from string.
// Deprecated: use AddRaw.
func (m *Message) AddSoftware(software string) {
	m.AddRaw(AttrSoftware, []byte(software))
}

// Set sets the value of attribute if it presents.
func (m *Message) Set(a AttrEncoder) error {
	var (
		v   []byte
		err error
		t   AttrType
	)
	t, v, err = a.Encode(v, m)
	if err != nil {
		return err
	}
	buf, err := m.getAttrValue(t)
	if err != nil {
		return err
	}
	if len(v) != len(buf) {
		return ErrBadSetLength
	}
	copy(buf, v)
	return nil
}

// GetSoftwareBytes returns SOFTWARE attribute value in byte slice.
// If not found, returns nil.
func (m *Message) GetSoftwareBytes() []byte {
	v, ok := m.Attributes.Get(AttrSoftware)
	if !ok {
		return nil
	}
	return v.Value
}

// GetSoftware returns SOFTWARE attribute value in string.
// If not found, returns blank string.
// Deprecated.
func (m *Message) GetSoftware() string { return string(m.GetSoftwareBytes()) }

// Address family values.
const (
	FamilyIPv4 byte = 0x01
	FamilyIPv6 byte = 0x02
)

// XORMappedAddress implements XOR-MAPPED-ADDRESS attribute.
type XORMappedAddress struct {
	ip   net.IP
	port int
}

// Encode implements AttrEncoder.
func (a *XORMappedAddress) Encode(buf []byte, m *Message) (AttrType, []byte, error) {
	// X-Port is computed by taking the mapped port in host byte order,
	// XOR’ing it with the most significant 16 bits of the magic cookie, and
	// then the converting the result to network byte order.
	family := FamilyIPv6
	ip := a.ip
	port := a.port
	if ipV4 := ip.To4(); ipV4 != nil {
		ip = ipV4
		family = FamilyIPv4
	}
	value := make([]byte, 32+128)
	value[0] = 0 // first 8 bits are zeroes
	xorValue := make([]byte, net.IPv6len)
	copy(xorValue[4:], m.TransactionID[:])
	binary.BigEndian.PutUint32(xorValue[0:4], magicCookie)
	port ^= magicCookie >> 16
	binary.BigEndian.PutUint16(value[0:2], uint16(family))
	binary.BigEndian.PutUint16(value[2:4], uint16(port))
	xorBytes(value[4:4+len(ip)], ip, xorValue)
	buf = append(buf, value...)
	return AttrXORMappedAddress, buf, nil
}

// Decode implements AttrDecoder.
// TODO(ar): fix signature.
func (a *XORMappedAddress) Decode(v []byte, m *Message) error {
	// X-Port is computed by taking the mapped port in host byte order,
	// XOR’ing it with the most significant 16 bits of the magic cookie, and
	// then the converting the result to network byte order.
	v, err := m.getAttrValue(AttrXORMappedAddress)
	if err != nil {
		return err
	}
	family := byte(binary.BigEndian.Uint16(v[0:2]))
	if family != FamilyIPv6 && family != FamilyIPv4 {
		return newDecodeErr("xor-mapped address", "family",
			fmt.Sprintf("bad value %d", family),
		)
	}
	ipLen := net.IPv4len
	if family == FamilyIPv6 {
		ipLen = net.IPv6len
	}
	ip := net.IP(m.allocBuffer(ipLen))
	a.port = int(binary.BigEndian.Uint16(v[2:4])) ^ (magicCookie >> 16)
	xorValue := make([]byte, 128)
	binary.BigEndian.PutUint32(xorValue[0:4], magicCookie)
	copy(xorValue[4:], m.TransactionID[:])
	xorBytes(ip, v[4:], xorValue)
	a.ip = ip
	return nil
}

// AddXORMappedAddress adds XOR MAPPED ADDRESS attribute to message.
// Deprecated: use AddRaw.
func (m *Message) AddXORMappedAddress(ip net.IP, port int) {
	// X-Port is computed by taking the mapped port in host byte order,
	// XOR’ing it with the most significant 16 bits of the magic cookie, and
	// then the converting the result to network byte order.
	family := FamilyIPv6
	if ipV4 := ip.To4(); ipV4 != nil {
		ip = ipV4
		family = FamilyIPv4
	}
	value := make([]byte, 32+128)
	value[0] = 0 // first 8 bits are zeroes
	xorValue := make([]byte, net.IPv6len)
	copy(xorValue[4:], m.TransactionID[:])
	binary.BigEndian.PutUint32(xorValue[0:4], magicCookie)
	port ^= magicCookie >> 16
	binary.BigEndian.PutUint16(value[0:2], uint16(family))
	binary.BigEndian.PutUint16(value[2:4], uint16(port))
	xorBytes(value[4:4+len(ip)], ip, xorValue)
	m.AddRaw(AttrXORMappedAddress, value[:4+len(ip)])
}

func (m *Message) allocBuffer(size int) []byte {
	capacity := len(m.Raw) + size
	m.grow(capacity)
	m.Raw = m.Raw[:capacity]
	return m.Raw[len(m.Raw)-size:]
}

// GetXORMappedAddress returns ip, port from attribute and error if any.
// Value for ip is valid until Message is released or underlying buffer is
// corrupted. Returns *DecodeError or ErrAttributeNotFound.
// Deprecated: use GetRaw.
func (m *Message) GetXORMappedAddress() (net.IP, int, error) {
	// X-Port is computed by taking the mapped port in host byte order,
	// XOR’ing it with the most significant 16 bits of the magic cookie, and
	// then the converting the result to network byte order.
	v, err := m.getAttrValue(AttrXORMappedAddress)
	if err != nil {
		return nil, 0, err
	}
	family := byte(binary.BigEndian.Uint16(v[0:2]))
	if family != FamilyIPv6 && family != FamilyIPv4 {
		return nil, 0, newDecodeErr("xor-mapped address", "family",
			fmt.Sprintf("bad value %d", family),
		)
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

// constants for ERROR-CODE encoding.
const (
	errorCodeReasonStart = 4
	errorCodeClassByte   = 2
	errorCodeNumberByte  = 3
	errorCodeReasonMaxB  = 763
	errorCodeModulo      = 100
)

// AddErrorCode adds ERROR-CODE attribute to message.
//
// The reason phrase MUST be a UTF-8 [RFC 3629] encoded
// sequence of less than 128 characters (which can be as long as 763
// bytes).
// Deprecated: use AddRaw.
func (m *Message) AddErrorCode(code int, reason string) {
	value := make([]byte,
		errorCodeReasonStart, errorCodeReasonMaxB+errorCodeReasonStart,
	)
	number := byte(code % errorCodeModulo) // error code modulo 100
	class := byte(code / errorCodeModulo)  // hundred digit
	value[errorCodeClassByte] = class
	value[errorCodeNumberByte] = number
	value = append(value, reason...)
	m.AddRaw(AttrErrorCode, value)
}

// AddErrorCodeDefault is wrapper for AddErrorCode that uses recommended
// reason string from RFC. If error code is unknown, reason will be "Unknown
// Error".
// Deprecated: use AddRaw.
func (m *Message) AddErrorCodeDefault(code int) {
	m.AddErrorCode(code, ErrorCode(code).Reason())
}

// GetErrorCode returns ERROR-CODE code, reason and decode error if any.
// Deprecated: use GetRaw.
func (m *Message) GetErrorCode() (int, []byte, error) {
	v, err := m.getAttrValue(AttrErrorCode)
	if err != nil {
		return 0, nil, err
	}
	var (
		class  = uint16(v[errorCodeClassByte])
		number = uint16(v[errorCodeNumberByte])
		code   = int(class*errorCodeModulo + number)
		reason = v[errorCodeReasonStart:]
	)
	return code, reason, nil
}

const (
	// ErrAttributeNotFound means that there is no such attribute.
	ErrAttributeNotFound Error = "Attribute not found"

	// ErrBadSetLength means that previous attribute value length differs from
	// new value.
	ErrBadSetLength Error = "Previous attribute length is different"
)

// Software is SOFTWARE attribute.
type Software struct {
	Raw []byte
}

func (s Software) String() string {
	return string(s.Raw)
}

// NewSoftware returns *Software from string.
func NewSoftware(software string) *Software {
	return &Software{Raw: []byte(software)}
}

// Encode implements AttrEncoder.
func (s *Software) Encode(b []byte, m *Message) (AttrType, []byte, error) {
	return AttrSoftware, append(b, s.Raw...), nil
}

// Decode implements AttrDecoder.
func (s *Software) Decode(v []byte, m *Message) error {
	s.Raw = v
	return nil
}

const (
	fingerprintXORValue uint32 = 0x5354554e
)

// Fingerprint represents FINGERPRINT attribute.
type Fingerprint struct {
	Value uint32 // CRC-32 of message XOR-ed with 0x5354554e
}

const (
	fingerprintSize = 4 // 32 bit
)

// AddTo adds fingerprint to message.
func (f *Fingerprint) AddTo(m *Message) error {
	l := m.Length
	// length in header should include size of fingerprint attribute
	m.Length += fingerprintSize + attributeHeaderSize // increasing length
	m.WriteLength()                                   // writing Length to Raw
	b := make([]byte, fingerprintSize)
	f.Value = crc32.ChecksumIEEE(m.Raw) ^ fingerprintXORValue // XOR
	bin.PutUint32(b, f.Value)
	m.Length = l
	m.AddRaw(AttrFingerprint, b)
	return nil
}

// Check reads fingerprint value from m and checks it, returning error if any.
// Can return *DecodeErr, ErrAttributeNotFound, ErrCRCMissmatch.
func (f *Fingerprint) Check(m *Message) error {
	v, err := m.getAttrValue(AttrFingerprint)
	if err != nil {
		return err
	}
	if len(v) != fingerprintSize {
		return newDecodeErr("message", "fingerprint", "bad length")
	}
	f.Value = bin.Uint32(v)
	attrStart := len(m.Raw) - (fingerprintSize + attributeHeaderSize)
	expected := crc32.ChecksumIEEE(m.Raw[:attrStart]) ^ fingerprintXORValue
	if expected != f.Value {
		return ErrCRCMissmatch
	}
	return nil
}

// ErrCRCMissmatch means that calculated fingerprint attribute differs from
// expected one.
const ErrCRCMissmatch Error = "CRC32 missmatch: bad fingerprint value"
