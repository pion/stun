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
	"encoding/binary"
	"fmt"
	"strconv"

	"github.com/syndtr/goleveldb/leveldb/errors"
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

type message struct {
	Type          messageType
	Length        uint16                  // size, in bytes, not including the 20-byte STUN header
	TransactionID [transactionIDSize]byte // used to uniquely identify STUN transactions.
	Attributes    []attribute
}

func (m message) Put(buf []byte) {
	binary.BigEndian.PutUint16(buf[0:2], m.Type.Value())
	binary.BigEndian.PutUint16(buf[2:4], m.Length)
	binary.BigEndian.PutUint32(buf[4:8], magicCookie)
	copy(buf[8:20], m.TransactionID[:])
	offset := 20
	for _, a := range m.Attributes {
		binary.BigEndian.PutUint16(buf[offset:offset+2], a.Type.Value())
		offset += 2
		binary.BigEndian.PutUint16(buf[offset:offset+2], a.Length)
		offset += 2
		copy(buf[offset:offset+len(a.Value)], a.Value[:])
	}
}

func (m *message) Get(buf []byte) error {
	m.Type.ReadValue(binary.BigEndian.Uint16(buf[0:2]))
	m.Length = binary.BigEndian.Uint16(buf[2:4])
	cookie := binary.BigEndian.Uint32(buf[4:8])
	copy(m.TransactionID[:], buf[8:20])
	if cookie != magicCookie {
		return ErrInvalidMagicCookie
	}
	return nil
}

type attributeType uint16

func (t attributeType) Value() uint16 {
	return uint16(t)
}

type attribute struct {
	Type   attributeType
	Length uint16
	Value  []byte
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
	methodBinding = 0x01 // 0b000000000001
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

	firstBit  = 0x01
	secondBit = 0x02

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

	// splitting M into A(M0-M3), B(M4-M6), D(M7-M11)
	m := uint16(t.Method)
	a := m & methodABits                              // A = M * 0b0000000000001111
	b := m & methodBBits                              // B = M * 0b0000000001110000
	d := m & methodDBits                              // D = M * 0b0000111110000000
	m = a + (b << methodBShift) + (d << methodDShift) // shifting to add "holes" for C0 (at 4 bit) and C1 (8 bit)

	// C0 is zero bit of C, C1 is fist bit.
	// C0 = C * 0b01, C1 = (C * 0b10) >> 1
	// Ct = C0 << 4 + C1 << 8.
	// Optimizations: "((C * 0b10) >> 1) << 8" as "(C * 0b10) << 7"
	// We need C0 shifted by 4, and C1 by 8 to fit "11" and "7" positions (see figure 3).
	class := (uint16(t.Class&firstBit) << classC0Shift) + (uint16(t.Class&secondBit) << classC1Shift)

	return m + class
}

func (t *messageType) ReadValue(v uint16) {
	// decoding class
	// we are taking first bit from v >> 4 and second from v >> 7.
	class := (v>>classC0Shift)&firstBit + (v>>classC1Shift)&secondBit
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
	ErrInvalidMagicCookie = errors.New("Magic cookie value is invalid")
)
