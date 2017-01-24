package stun

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"

	"github.com/pkg/errors"
)

// blank is just blank string and exists just because it is ugly to keep it
// in code.
const blank = ""

// Attributes is list of message attributes.
type Attributes []Attribute

// BlankAttribute is attribute that is returned by
// Attributes.Get if nothing found.
var BlankAttribute = Attribute{}

// Get returns first attribute from list which match AttrType. If nothing
// found, it returns blank attribute.
func (a Attributes) Get(t AttrType) Attribute {
	for _, candidate := range a {
		if candidate.Type == t {
			return candidate
		}
	}
	return BlankAttribute
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

// Attributes from An Origin Attribute for the STUN Protocol.
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

// Attribute is a Type-Length-Value (TLV) object that
// can be added to a STUN message.  Attributes are divided into two
// types: comprehension-required and comprehension-optional.  STUN
// agents can safely ignore comprehension-optional attributes they
// don't understand, but cannot successfully process a message if it
// contains comprehension-required attributes that are not
// understood.
type Attribute struct {
	Type   AttrType
	Length uint16
	Value  []byte
}

// IsBlank returns true if attribute equals to BlankAttribute.
func (a Attribute) IsBlank() bool {
	return a.Equal(BlankAttribute)
}

// Equal returns true if a == b.
func (a Attribute) Equal(b Attribute) bool {
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

func (a Attribute) String() string {
	return fmt.Sprintf("%s: %x", a.Type, a.Value)
}

// getAttrValue returns byte slice that represents attribute value,
// and if there is no value found, error returned.
func (m *Message) getAttrValue(t AttrType) ([]byte, error) {
	v := m.Attributes.Get(t).Value
	if len(v) == 0 {
		return nil, errors.Wrap(ErrAttributeNotFound, "failed to find")
	}
	return v, nil
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
	v, err := m.getAttrValue(AttrXORMappedAddress)
	if len(v) == 0 {
		return nil, 0, errors.Wrap(err, "address not found")
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
func (m *Message) AddErrorCode(code int, reason string) {
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

// AddErrorCodeDefault is wrapper for AddErrorCode that uses recommended
// reason string from RFC. If error code is unknown, reason will be "Unknown
// Error".
func (m *Message) AddErrorCodeDefault(code int) {
	m.AddErrorCode(code, ErrorCode(code).Reason())
}

// GetErrorCode returns ERROR-CODE code, reason and decode error if any.
func (m *Message) GetErrorCode() (int, []byte, error) {
	v, err := m.getAttrValue(AttrErrorCode)
	if err != nil {
		return 0, nil, errors.Wrap(err, "error not found")
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
)
