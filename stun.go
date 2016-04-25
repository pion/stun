// Package stun implements Session Traversal Utilities for NAT (STUN) RFC 5389.
//
// See https://tools.ietf.org/html/rfc5389 for specification.
//
// Definitions
//
// STUN Agent: A STUN agent is an entity that implements the STUN
// protocol. The entity can be either a STUN client or a STUN
// server.
//
// STUN Client: A STUN client is an entity that sends STUN requests and
// receives STUN responses. A STUN client can also send indications.
// In this specification, the terms STUN client and client are
// synonymous.
//
// STUN Server: A STUN server is an entity that receives STUN requests
// and sends STUN responses. A STUN server can also send
// indications. In this specification, the terms STUN server and
// server are synonymous.
//
// Transport Address: The combination of an IP address and port number
// (such as a UDP or TCP port number).
package stun

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"

	log "github.com/Sirupsen/logrus"
)

const (
	// magicCookie is fixed value that aids in distinguishing STUN packets
	// from packets of other protocols when STUN is multiplexed with those
	// other protocols on the same port.
	//
	// The magic cookie field MUST contain the fixed value 0x2112A442 in
	// network byte order.
	//
	// Defined in "STUN Message Structure", section 6.
	magicCookie = 0x2112A442
)

const transactionIDSize = 12 // 96 bit

type attributes []attribute

func (a attributes) Get(t attrType) attribute {
	for _, candidate := range a {
		if candidate.Type == t {
			return candidate
		}
	}
	return attribute{}
}

type message struct {
	Type          messageType
	Length        uint16                  // TODO: is it needed?
	TransactionID [transactionIDSize]byte // used to uniquely identify STUN transactions.
	Attributes    attributes
}

func (m message) String() string {
	return fmt.Sprintf("%s (len=%d) (attr=%d) [%s]",
		m.Type,
		m.Length,
		len(m.Attributes),
		hex.EncodeToString(m.TransactionID[:]),
	)
}

func unexpected(err error) {
	if err != nil {
		panic(err)
	}
}

func newTransactionID() (b [transactionIDSize]byte) {
	_, err := rand.Read(b[:])
	unexpected(err)
	return b
}

// Put encodes message into buf. If len(buf) is not enough, it panics.
func (m message) Put(buf []byte) {
	// encoding header
	binary.BigEndian.PutUint16(buf[0:2], m.Type.Value())
	binary.BigEndian.PutUint32(buf[4:8], magicCookie)
	copy(buf[8:messageHeaderSize], m.TransactionID[:])
	offset := messageHeaderSize
	// encoding attributes
	for _, a := range m.Attributes {
		binary.BigEndian.PutUint16(buf[offset:offset+2], a.Type.Value())
		offset += 2
		binary.BigEndian.PutUint16(buf[offset:offset+2], a.Length)
		offset += 2
		copy(buf[offset:offset+len(a.Value)], a.Value[:])
		offset += len(a.Value)
	}
	// writing length as size, in bytes, not including the 20-byte STUN header.
	binary.BigEndian.PutUint16(buf[2:4], uint16(offset-20))
}

// Get decodes message from byte slice and return error if any.
//
// Can return ErrUnexpectedEOF, ErrInvalidMagicCookie, ErrInvalidMessageLength.
// Any error is unrecoverable, but message could be partially decoded.
//
// ErrUnexpectedEOF means that there were not enough bytes to read header or
// value and indicates possible data loss.
func (m *message) Get(buf []byte) error {
	if len(buf) < messageHeaderSize {
		return io.ErrUnexpectedEOF
	}

	// decoding message header
	m.Type.ReadValue(binary.BigEndian.Uint16(buf[0:2])) // first 2 bytes
	m.Length = binary.BigEndian.Uint16(buf[2:4])        // second 2 bytes
	cookie := binary.BigEndian.Uint32(buf[4:8])
	copy(m.TransactionID[:], buf[8:messageHeaderSize])

	if cookie != magicCookie {
		return ErrInvalidMagicCookie
	}

	offset := messageHeaderSize
	mLength := int(m.Length)
	if (mLength + offset) > len(buf) {
		log.WithFields(log.Fields{
			"len(b)": len(buf),
			"offset": offset,
		}).Debugln("message length", mLength, "is invalid?")
		return ErrInvalidMessageLength
	}

	for (mLength + messageHeaderSize - offset) > 0 {
		b := buf[offset:]
		// checking that we have enough bytes to read header
		if len(b) < attributeHeaderSize {
			return io.ErrUnexpectedEOF
		}
		a := attribute{}

		// decoding header
		t := binary.BigEndian.Uint16(b[0:2])       // first 2 bytes
		a.Length = binary.BigEndian.Uint16(b[2:4]) // second 2 bytes
		a.Type = attrType(t)
		l := int(a.Length)

		// reading value
		a.Value = make([]byte, l)   // we could possibly use pool here
		b = b[attributeHeaderSize:] // slicing again to simplify value read
		if len(b) < l {             // checking size
			return io.ErrUnexpectedEOF
		}
		copy(a.Value, b[:l])

		m.Attributes = append(m.Attributes, a)
		offset += l + attributeHeaderSize
	}
	return nil
}

const (
	attributeHeaderSize = 4
	messageHeaderSize   = 20
)

type attrType uint16

// attributes from comprehension-required range (0x0000-0x7FFF).
const (
	attrMappedAddress     attrType = 0x0001 // MAPPED-ADDRESS
	attrUsername          attrType = 0x0006 // USERNAME
	attrErrorCode         attrType = 0x0009 // ERROR-CODE
	attrMessageIntegrity  attrType = 0x0008 // MESSAGE-INTEGRITY
	attrUnknownAttributes attrType = 0x000A // UNKNOWN-ATTRIBUTES
	attrRealm             attrType = 0x0014 // REALM
	attrNonce             attrType = 0x0015 // NONCE
	attrXORMappedAddress  attrType = 0x0020 // XOR-MAPPED-ADDRESS
)

// attributes from comprehension-optional range (0x8000-0xFFFF).
const (
	attrSoftware        attrType = 0x8022 // SOFTWARE
	attrAlternateServer attrType = 0x8023 // ALTERNATE-SERVER
	attrFingerprint     attrType = 0x8028 // FINGERPRINT

)

// Value returns uint16 representation of attribute type.
func (t attrType) Value() uint16 {
	return uint16(t)
}

func (t attrType) String() string {
	switch t {
	case attrMappedAddress:
		return "MAPPED-ADDRESS"
	case attrUsername:
		return "USERNAME"
	case attrErrorCode:
		return "ERROR-CODE"
	case attrMessageIntegrity:
		return "MESSAGE-INTEGRITY"
	case attrUnknownAttributes:
		return "UNKNOWN-ATTRIBUTES"
	case attrRealm:
		return "REALM"
	case attrNonce:
		return "NONCE"
	case attrXORMappedAddress:
		return "XOR-MAPPED-ADDRESS"
	case attrSoftware:
		return "SOFTWARE"
	case attrAlternateServer:
		return "ALTERNATE-SERVER"
	case attrFingerprint:
		return "FINGERPRINT"
	default:
		// just return hex representation of unknown attribute type
		return "0x" + strconv.FormatUint(uint64(t), 16)
	}
}

type attribute struct {
	Type   attrType
	Length uint16
	Value  []byte
}

// Equal returns true if a == b.
func (a attribute) Equal(b attribute) bool {
	if a.Type != b.Type {
		return false
	}
	if a.Length != b.Length {
		return false
	}
	for i, v := range a.Value {
		if b.Value[i] != v {
			return false
		}
	}
	return true
}

func (a attribute) String() string {
	return fmt.Sprintf("%s: %x", a.Type, a.Value)
}

// messageClass is 8-bit representation of 2-bit class of STUN Message Type.
type messageClass byte

// possible values for message class in STUN Message Type.
const (
	classRequest         = 0x00 // 0b00
	classIndication      = 0x01 // 0b01
	classSuccessResponse = 0x02 // 0b10
	classErrorResponse   = 0x03 // 0b11
)

func (c messageClass) String() string {
	switch c {
	case classRequest:
		return "request"
	case classIndication:
		return "indication"
	case classSuccessResponse:
		return "success response"
	case classErrorResponse:
		return "error response"
	default:
		panic("unknown message class")
	}
}

// method is uint16 representation of 12-bit STUN method.
type method uint16

// possible methods for STUN Message.
const (
	methodBinding method = 0x01 // 0b000000000001
)

func (m method) String() string {
	switch m {
	case methodBinding:
		return "binding"
	default:
		return fmt.Sprintf("0x%s", strconv.FormatUint(uint64(m), 16))
	}
}

// messageType is STUN Message Type Field.
type messageType struct {
	Class  messageClass
	Method method
}

const (
	methodABits = 0xf   // 0b0000000000001111
	methodBBits = 0x70  // 0b0000000001110000
	methodDBits = 0xf80 // 0b0000111110000000

	methodBShift = 1
	methodDShift = 2

	firstBit  = 0x1
	secondBit = 0x2

	c0Bit = firstBit
	c1Bit = secondBit

	classC0Shift = 4
	classC1Shift = 7
)

// Value returns bit representation of messageType.
func (t messageType) Value() uint16 {
	//	 0                 1
	//	 2  3  4 5 6 7 8 9 0 1 2 3 4 5
	//	+--+--+-+-+-+-+-+-+-+-+-+-+-+-+
	//	|M |M |M|M|M|C|M|M|M|C|M|M|M|M|
	//	|11|10|9|8|7|1|6|5|4|0|3|2|1|0|
	//	+--+--+-+-+-+-+-+-+-+-+-+-+-+-+
	// Figure 3: Format of STUN Message Type Field

	// warning: Abandon all hope ye who enter here.
	// splitting M into A(M0-M3), B(M4-M6), D(M7-M11)
	m := uint16(t.Method)
	a := m & methodABits                              // A = M * 0b0000000000001111 (right 4 bits)
	b := m & methodBBits                              // B = M * 0b0000000001110000 (3 bits after A)
	d := m & methodDBits                              // D = M * 0b0000111110000000 (5 bits after B)
	m = a + (b << methodBShift) + (d << methodDShift) // shifting to add "holes" for C0 (at 4 bit) and C1 (8 bit)

	// C0 is zero bit of C, C1 is fist bit.
	// C0 = C * 0b01, C1 = (C * 0b10) >> 1
	// Ct = C0 << 4 + C1 << 8.
	// Optimizations: "((C * 0b10) >> 1) << 8" as "(C * 0b10) << 7"
	// We need C0 shifted by 4, and C1 by 8 to fit "11" and "7" positions (see figure 3).
	c := uint16(t.Class)
	c0 := (c & c0Bit) << classC0Shift
	c1 := (c & c1Bit) << classC1Shift
	class := c0 + c1

	return m + class
}

func (t *messageType) ReadValue(v uint16) {
	// decoding class
	// we are taking first bit from v >> 4 and second from v >> 7.
	c0 := (v >> classC0Shift) & c0Bit
	c1 := (v >> classC1Shift) & c1Bit
	class := c0 + c1
	t.Class = messageClass(class)

	// decoding method
	a := v & methodABits                   // A(M0-M3)
	b := (v >> methodBShift) & methodBBits // B(M4-M6)
	d := (v >> methodDShift) & methodDBits // D(M7-M11)
	m := a + b + d
	t.Method = method(m)
}

func (t messageType) String() string {
	return fmt.Sprintf("%s %s", t.Method, t.Class)
}

var (
	// ErrInvalidMagicCookie means that magic cookie field has invalid value.
	ErrInvalidMagicCookie = errors.New("Magic cookie value is invalid")
	// ErrInvalidMessageLength means that actual message size is smaller that length
	// from header field.
	ErrInvalidMessageLength = errors.New("Message size is smaller than specified length")
)
