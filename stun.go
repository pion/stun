// Package stun implements Session Traversal Utilities for NAT (STUN) RFC 5389.
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
	"sync/atomic"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/cydev/buffer"
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

// Attributes is list of message attributes.
type Attributes []Attribute

var (
	// BlankAttribute is attribute that is returned by
	// Attributes.Get if nothing found.
	BlankAttribute = Attribute{}
)

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

// Message represents a single STUN packet.
type Message struct {
	Type   MessageType
	Length uint32
	// TransactionID is used to uniquely identify STUN transactions.
	TransactionID [transactionIDSize]byte
	Attributes    Attributes

	// buf is underlying raw data buffer.
	buf *buffer.Buffer
}

func (m Message) String() string {
	return fmt.Sprintf("%s (len=%d) (attr=%d) [%s]",
		m.Type,
		m.Length,
		len(m.Attributes),
		hex.EncodeToString(m.TransactionID[:]),
	)
}

// unexpected panics if err is not nil.
func unexpected(err error) {
	if err != nil {
		panic(err)
	}
}

// NewTransactionID returns new random transaction ID using crypto/rand
// as source.
func NewTransactionID() (b [transactionIDSize]byte) {
	_, err := rand.Read(b[:])
	unexpected(err)
	return b
}

var messagePool = sync.Pool{
	New: func() interface{} {
		b := &buffer.Buffer{
			B: make([]byte, 0, defaultMessageBufferSize),
		}
		b.Grow(messageHeaderSize)
		m := &Message{
			Attributes: make(Attributes, 0, defaultAttributesSize),
			buf:        b,
		}
		return m
	},
}

const (
	defaultAttributesSize    = 12
	defaultMessageBufferSize = 416
)

// AcquireMessage returns new message from pool.
func AcquireMessage() *Message {
	return messagePool.Get().(*Message)
}

// ReleaseMessage returns message to pool rendering it to unusable state.
// After release, any usage of message and its attributes is invalid.
func ReleaseMessage(m *Message) {
	m.buf.Reset()
	m.Length = 0
	m.Attributes = m.Attributes[:0]
	messagePool.Put(m)
}

// Add appends new attribute to message.
func (m *Message) Add(t AttrType, v []byte) {
	// allocating space for buffer
	// m.buf.B[0:20] is reserved by header
	attrLength := uint32(len(v))
	m.buf.Grow(attributeHeaderSize + len(v))
	newLength := messageHeaderSize + atomic.AddUint32(&m.Length,
		attrLength+attributeHeaderSize) - attributeHeaderSize
	// writing attribute to allocated space
	buf := m.buf.B[newLength-attrLength : newLength]
	binary.BigEndian.PutUint16(buf[0:2], t.Value())
	binary.BigEndian.PutUint16(buf[2:4], uint16(attrLength))
	attrValue := buf[attributeHeaderSize : attributeHeaderSize+len(v)]
	copy(attrValue, v)

	// appending attribute
	m.Attributes = append(m.Attributes, Attribute{
		Type:   t,
		Value:  attrValue,
		Length: uint16(attrLength),
	})
}

// Equal returns true if Message b equals to m.
func (m Message) Equal(b Message) bool {
	if m.Type != b.Type {
		return false
	}
	if m.TransactionID != b.TransactionID {
		return false
	}
	if m.Length != b.Length {
		return false
	}
	for _, a := range m.Attributes {
		aB := b.Attributes.Get(a.Type)
		if !aB.Equal(a) {
			return false
		}
	}
	return true
}

// WriteHeader writes header to underlying buffer.
func (m *Message) WriteHeader() {
	buf := m.buf.B
	// encoding header
	binary.BigEndian.PutUint16(buf[0:2], m.Type.Value())
	binary.BigEndian.PutUint32(buf[4:8], magicCookie)
	copy(buf[8:messageHeaderSize], m.TransactionID[:])
	// attributes are already encoded
	// writing length as size, in bytes, not including the 20-byte STUN header.
	binary.BigEndian.PutUint16(buf[2:4], uint16(len(buf)-20))
}

// Read implements Reader.
func (m Message) Read(b []byte) (int, error) {
	copy(b, m.buf.B)
	return len(m.buf.B), nil
}

// Put encodes message into buf. If len(buf) is not enough, it panics.
func (m Message) Put(buf []byte) {
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
func (m *Message) Get(buf []byte) error {
	if len(buf) < messageHeaderSize {
		log.Debugln(len(buf), "<", messageHeaderSize, "message")
		return io.ErrUnexpectedEOF
	}

	// decoding message header
	m.Type.ReadValue(binary.BigEndian.Uint16(buf[0:2])) // first 2 bytes
	tLength := binary.BigEndian.Uint16(buf[2:4])        // second 2 bytes
	m.Length = uint32(tLength)
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
			log.Debugln(len(buf), "<", attributeHeaderSize, "header")
			return io.ErrUnexpectedEOF
		}
		a := Attribute{}

		// decoding header
		t := binary.BigEndian.Uint16(b[0:2])       // first 2 bytes
		a.Length = binary.BigEndian.Uint16(b[2:4]) // second 2 bytes
		a.Type = AttrType(t)
		l := int(a.Length)

		// reading value
		b = b[attributeHeaderSize:] // slicing again to simplify value read
		if len(b) < l {             // checking size
			return io.ErrUnexpectedEOF
		}
		a.Value = b[:l]

		m.Attributes = append(m.Attributes, a)
		offset += l + attributeHeaderSize
	}
	return nil
}

const (
	attributeHeaderSize = 4
	messageHeaderSize   = 20
)

// AttrType is attribute type.
type AttrType uint16

// Attributes from comprehension-required range (0x0000-0x7FFF).
const (
	AttrMappedAddress     AttrType = 0x0001 // MAPPED-ADDRESS
	AttrUsername          AttrType = 0x0006 // USERNAME
	AttrErrorCode         AttrType = 0x0009 // ERROR-CODE
	AttrMessageIntegrity  AttrType = 0x0008 // MESSAGE-INTEGRITY
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

// Value returns uint16 representation of attribute type.
func (t AttrType) Value() uint16 {
	return uint16(t)
}

func (t AttrType) String() string {
	switch t {
	case AttrMappedAddress:
		return "MAPPED-ADDRESS"
	case AttrUsername:
		return "USERNAME"
	case AttrErrorCode:
		return "ERROR-CODE"
	case AttrMessageIntegrity:
		return "MESSAGE-INTEGRITY"
	case AttrUnknownAttributes:
		return "UNKNOWN-ATTRIBUTES"
	case AttrRealm:
		return "REALM"
	case AttrNonce:
		return "NONCE"
	case AttrXORMappedAddress:
		return "XOR-MAPPED-ADDRESS"
	case AttrSoftware:
		return "SOFTWARE"
	case AttrAlternateServer:
		return "ALTERNATE-SERVER"
	case AttrFingerprint:
		return "FINGERPRINT"
	default:
		// just return hex representation of unknown attribute type
		return "0x" + strconv.FormatUint(uint64(t), 16)
	}
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

// Equal returns true if a == b.
func (a Attribute) Equal(b Attribute) bool {
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

func (a Attribute) String() string {
	return fmt.Sprintf("%s: %x", a.Type, a.Value)
}

// MessageClass is 8-bit representation of 2-bit class of STUN Message Class.
type MessageClass byte

// Possible values for message class in STUN Message Type.
const (
	ClassRequest         = 0x00 // 0b00
	ClassIndication      = 0x01 // 0b01
	ClassSuccessResponse = 0x02 // 0b10
	ClassErrorResponse   = 0x03 // 0b11
)

func (c MessageClass) String() string {
	switch c {
	case ClassRequest:
		return "request"
	case ClassIndication:
		return "indication"
	case ClassSuccessResponse:
		return "success response"
	case ClassErrorResponse:
		return "error response"
	default:
		panic("unknown message class")
	}
}

// Method is uint16 representation of 12-bit STUN method.
type Method uint16

// Possible methods for STUN Message.
const (
	MethodBinding Method = 0x01 // 0b000000000001
)

func (m Method) String() string {
	switch m {
	case MethodBinding:
		return "binding"
	default:
		return fmt.Sprintf("0x%s", strconv.FormatUint(uint64(m), 16))
	}
}

// MessageType is STUN Message Type Field.
type MessageType struct {
	Class  MessageClass
	Method Method
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
func (t MessageType) Value() uint16 {
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
	a := m & methodABits // A = M * 0b0000000000001111 (right 4 bits)
	b := m & methodBBits // B = M * 0b0000000001110000 (3 bits after A)
	d := m & methodDBits // D = M * 0b0000111110000000 (5 bits after B)

	// shifting to add "holes" for C0 (at 4 bit) and C1 (8 bit)
	m = a + (b << methodBShift) + (d << methodDShift)

	// C0 is zero bit of C, C1 is fist bit.
	// C0 = C * 0b01, C1 = (C * 0b10) >> 1
	// Ct = C0 << 4 + C1 << 8.
	// Optimizations: "((C * 0b10) >> 1) << 8" as "(C * 0b10) << 7"
	// We need C0 shifted by 4, and C1 by 8 to fit "11" and "7" positions
	// (see figure 3).
	c := uint16(t.Class)
	c0 := (c & c0Bit) << classC0Shift
	c1 := (c & c1Bit) << classC1Shift
	class := c0 + c1

	return m + class
}

// ReadValue decodes uint16 into MessageType.
func (t *MessageType) ReadValue(v uint16) {
	// decoding class
	// we are taking first bit from v >> 4 and second from v >> 7.
	c0 := (v >> classC0Shift) & c0Bit
	c1 := (v >> classC1Shift) & c1Bit
	class := c0 + c1
	t.Class = MessageClass(class)

	// decoding method
	a := v & methodABits                   // A(M0-M3)
	b := (v >> methodBShift) & methodBBits // B(M4-M6)
	d := (v >> methodDShift) & methodDBits // D(M7-M11)
	m := a + b + d
	t.Method = Method(m)
}

func (t MessageType) String() string {
	return fmt.Sprintf("%s %s", t.Method, t.Class)
}

var (
	// ErrInvalidMagicCookie means that magic cookie field has invalid value.
	ErrInvalidMagicCookie = errors.New("Magic cookie value is invalid")
	// ErrInvalidMessageLength means that actual message size is smaller that
	// length from header field.
	ErrInvalidMessageLength = errors.New("Message size is smaller than length")
)
