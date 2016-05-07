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
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"sync"

	"github.com/cydev/buffer"
	"github.com/pkg/errors"
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

// Message represents a single STUN packet. It uses aggressive internal
// buffering to enable zero-allocation encoding and decoding,
// so there are some usage constraints:
//
// 		* Message and its fields is valid only until AcquireMessage call.
// 		* Decoded message is read-only and any changes will cause panic.
//
// To change read-only message one must allocate new Message and copy
// contents. The main reason of making Message read-only are
// decode methods for attributes. They grow internal buffer and sub-slice
// it instead of allocating one, but it is used for encoding, so
// one Message instance cannot be used to encode and decode.
type Message struct {
	Type     MessageType
	readOnly bool // RO flag. Moved here to minimize padding overhead.
	Length   uint32
	// TransactionID is used to uniquely identify STUN transactions.
	TransactionID [transactionIDSize]byte
	Attributes    Attributes

	// buf is underlying raw data buffer.
	buf *buffer.Buffer
}

// Clone returns new copy of m.
func (m Message) Clone() *Message {
	c := AcquireMessage()
	c.Type = m.Type
	c.Length = m.Length
	copy(c.TransactionID[:], m.TransactionID[:])
	buf := m.buf.B[:int(m.Length)+messageHeaderSize]
	c.buf.B = c.buf.B[:0]
	c.buf.Append(buf)
	buf = c.buf.B[messageHeaderSize:]
	for _, a := range m.Attributes {
		buf = buf[attributeHeaderSize:]
		c.Attributes = append(c.Attributes, Attribute{
			Length: a.Length,
			Type:   a.Type,
			Value:  buf[:int(a.Length)],
		})
		buf = buf[int(a.Length):]
	}
	return c
}

func (m Message) String() string {
	return fmt.Sprintf("%s (l=%d,%d/%d) attr[%d] id[%s]",
		m.Type,
		m.Length,
		len(m.buf.B),
		cap(m.buf.B),
		len(m.Attributes),
		base64.StdEncoding.EncodeToString(m.TransactionID[:]),
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

// messagePool minimizes memory allocation by pooling Message,
// attribute slices and underlying buffers.
var messagePool = sync.Pool{
	New: func() interface{} {
		b := &buffer.Buffer{
			B: make([]byte, 0, defaultMessageBufferCapacity),
		}
		return &Message{
			Attributes: make(Attributes, 0, defaultAttributesCapacity),
			buf:        b,
		}
	},
}

// defaults for pool.
const (
	defaultAttributesCapacity    = 12
	defaultMessageBufferCapacity = 416
)

// AcquireMessage returns new message from pool.
func AcquireMessage() *Message {
	m := messagePool.Get().(*Message)
	m.grow(messageHeaderSize)
	return m
}

// ReleaseMessage returns message to pool rendering it to unusable state.
// After release, any usage of message and its attributes, also any
// value obtained via attribute decoding methods is invalid.
func ReleaseMessage(m *Message) {
	m.Reset()
	messagePool.Put(m)
}

// Reset resets Message length, attributes and underlying buffer.
func (m *Message) Reset() {
	m.buf.Reset()
	m.Length = 0
	m.readOnly = false
	m.Attributes = m.Attributes[:0]
}

// mustWrite panics if message is read-only.
func (m *Message) mustWrite() {
	if m.readOnly {
		unexpected(ErrMessageIsReadOnly)
	}
}

// grow ensures that internal buffer will fit v more bytes and
// increases it length.
func (m *Message) grow(v int) {
	// growing buffer if attribute value+header won't fit
	// not performing any optimizations here
	// because initial capacity and maximum theoretical size of buffer
	// are not far from each other.
	m.buf.Grow(v)
}

// Add appends new attribute to message. Not goroutine-safe.
//
// Value of attribute is copied to internal buffer so there are no
// constraints on validity.
func (m *Message) Add(t AttrType, v []byte) {
	m.mustWrite()
	// allocating space for buffer
	// m.buf.B[0:20] is reserved by header
	attrLength := uint32(len(v))

	// [0:20]                               <- header
	// [20:20+m.Length]                     <- attributes
	// [20+m.Length:20+m.Length+len(v) + 4] <- allocated

	allocSize := attributeHeaderSize + len(v)  // total attr size
	first := messageHeaderSize + int(m.Length) // first byte
	last := first + allocSize                  // last byte
	m.grow(last)
	m.buf.B = m.buf.B[:last]      // now len(b) = last
	m.Length += uint32(allocSize) // rendering changes

	// encoding attribute TLV to internal buffer
	buf := m.buf.B[first:last]
	binary.BigEndian.PutUint16(buf[0:2], t.Value())
	binary.BigEndian.PutUint16(buf[2:4], uint16(attrLength))
	copy(buf[attributeHeaderSize:], v)

	// appending attribute
	// note that we are reusing buf (actually a slice of m.buf.B) there
	m.Attributes = append(m.Attributes, Attribute{
		Type:   t,
		Value:  buf[attributeHeaderSize:],
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

// WriteHeader writes header to underlying buffer. Not goroutine-safe.
func (m *Message) WriteHeader() {
	m.mustWrite()

	buf := m.buf.B
	// encoding header
	binary.BigEndian.PutUint16(buf[0:2], m.Type.Value())
	binary.BigEndian.PutUint32(buf[4:8], magicCookie)
	copy(buf[8:messageHeaderSize], m.TransactionID[:])
	// attributes are already encoded
	// writing length as size, in bytes, not including the 20-byte STUN header.
	binary.BigEndian.PutUint16(buf[2:4], uint16(len(buf)-20))
}

// WriteTo implements WriterTo.
func (m Message) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(m.buf.B)
	return int64(n), err
}

// AcquireFields is shorthand for AcquireMessage that sets fields
// before returning *Message.
func AcquireFields(message Message) *Message {
	m := AcquireMessage()
	copy(m.TransactionID[:], message.TransactionID[:])
	m.Type = message.Type
	for _, a := range message.Attributes {
		m.Add(a.Type, a.Value)
	}
	m.WriteHeader()
	return m
}

func (m *Message) reader() *bytes.Reader {
	return bytes.NewReader(m.buf.B)
}

// ReadFrom implements ReaderFrom. Decodes message and return error if any.
//
// Can return ErrUnexpectedEOF, ErrInvalidMagicCookie, ErrInvalidMessageLength.
// Any error is unrecoverable, but message could be partially decoded.
//
// ErrUnexpectedEOF means that there were not enough bytes to read header or
func (m *Message) ReadFrom(r io.Reader) (int64, error) {
	m.mustWrite()
	tBuf := buffer.Acquire()
	if cap(tBuf.B) < MaxPacketSize {
		tBuf.Grow(MaxPacketSize)
	}
	var (
		n   int
		err error
	)
	if n, err = r.Read(tBuf.B[:MaxPacketSize]); err != nil {
		buffer.Release(tBuf)
		return int64(n), err
	}
	n, err = m.ReadBytes(tBuf.B[:n])
	buffer.Release(tBuf)
	return int64(n), err
}

func newAttrDecodeErr(children, message string) DecodeErr {
	return newDecodeErr("attribute", children, message)
}

// ReadBytes decodes message and return error if any.
//
// Any error is unrecoverable, but message could be partially decoded.
func (m *Message) ReadBytes(tBuf []byte) (int, error) {
	var (
		read int
		err  error
		n    = len(tBuf)
	)
	m.mustWrite()
	m.buf.Reset()
	_, err = m.buf.Write(tBuf)
	unexpected(err) // should never return error

	read += n
	buf := m.buf.B[:n]

	// decoding message header
	m.Type.ReadValue(binary.BigEndian.Uint16(buf[0:2])) // first 2 bytes
	tLength := binary.BigEndian.Uint16(buf[2:4])        // second 2 bytes
	m.Length = uint32(tLength)
	l := int(tLength)
	cookie := binary.BigEndian.Uint32(buf[4:8])
	copy(m.TransactionID[:], buf[8:messageHeaderSize])

	if cookie != magicCookie {
		msg := fmt.Sprintf(
			"%v is invalid magic cookie (should be %v)",
			cookie, magicCookie,
		)
		err := newDecodeErr("message", "cookie", msg)
		return read, errors.Wrap(err, "check failed")
	}

	buf = buf[messageHeaderSize : messageHeaderSize+l]
	offset := 0
	for offset < l {
		b := buf[offset:]
		// checking that we have enough bytes to read header
		if len(b) < attributeHeaderSize {
			msg := fmt.Sprintf(
				"buffer length %d is less than %d (expected header size)",
				len(b), attributeHeaderSize,
			)
			err := newAttrDecodeErr("header", msg)
			return offset + read, errors.Wrap(err, "failed to decode")
		}
		a := Attribute{}

		// decoding header
		t := binary.BigEndian.Uint16(b[0:2])       // first 2 bytes
		a.Length = binary.BigEndian.Uint16(b[2:4]) // second 2 bytes
		a.Type = AttrType(t)
		aL := int(a.Length)

		// reading value
		b = b[attributeHeaderSize:] // slicing again to simplify value read
		if len(b) < aL {            // checking size
			msg := fmt.Sprintf(
				"buffer length %d is less than %d (expected value size)",
				len(b), aL,
			)
			err := newAttrDecodeErr("value", msg)
			return offset + read, errors.Wrap(err, "failed to decode")
		}
		a.Value = b[:aL]

		m.Attributes = append(m.Attributes, a)
		offset += aL + attributeHeaderSize
	}
	return l + messageHeaderSize, nil
}

const (
	attributeHeaderSize = 4
	messageHeaderSize   = 20
)

const (
	// MaxPacketSize is maximum size of UDP packet that is processable in
	// this package for STUN message.
	MaxPacketSize = 2048
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

// MessageClass is 8-bit representation of 2-bit class of STUN Message Class.
type MessageClass byte

// Possible values for message class in STUN Message Type.
const (
	ClassRequest         MessageClass = 0x00 // 0b00
	ClassIndication      MessageClass = 0x01 // 0b01
	ClassSuccessResponse MessageClass = 0x02 // 0b10
	ClassErrorResponse   MessageClass = 0x03 // 0b11
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

const (
	// ErrInvalidMagicCookie means that magic cookie field has invalid value.
	ErrInvalidMagicCookie Error = "Magic cookie value is invalid"
	// ErrMessageIsReadOnly means that you are trying to modify readonly
	// Message.
	ErrMessageIsReadOnly Error = "Message is readonly"
)
