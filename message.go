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
// Transport Address: The combination of an IP address and Port number
// (such as a UDP or TCP Port number).
package stun

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
)

const (
	// magicCookie is fixed value that aids in distinguishing STUN packets
	// from packets of other protocols when STUN is multiplexed with those
	// other protocols on the same Port.
	//
	// The magic cookie field MUST contain the fixed value 0x2112A442 in
	// network byte order.
	//
	// Defined in "STUN Message Structure", section 6.
	magicCookie         = 0x2112A442
	attributeHeaderSize = 4
	messageHeaderSize   = 20
	transactionIDSize   = 12 // 96 bit
)

// NewTransactionID returns new random transaction ID using crypto/rand
// as source.
func NewTransactionID() (b [transactionIDSize]byte) {
	_, err := rand.Read(b[:])
	if err != nil {
		panic(err)
	}
	return b
}

// IsMessage returns true if b looks like STUN message.
// Useful for multiplexing. IsMessage does not guarantee
// that decoding will be successful.
func IsMessage(b []byte) bool {
	return len(b) >= messageHeaderSize && bin.Uint32(b[4:8]) == magicCookie
}

// New returns *Message with pre-allocated Raw.
func New() *Message {
	const defaultRawCapacity = 120
	return &Message{
		Raw: make([]byte, messageHeaderSize, defaultRawCapacity),
	}
}

// Message represents a single STUN packet. It uses aggressive internal
// buffering to enable zero-allocation encoding and decoding,
// so there are some usage constraints:
//
// 		* Message and its fields is valid only until AcquireMessage call.
type Message struct {
	Type          MessageType
	Length        uint32 // len(Raw) not including header
	TransactionID [transactionIDSize]byte
	Attributes    Attributes
	Raw           []byte
}

// NewTransactionID sets m.TransactionID to random value from crypto/rand
// and returns error if any.
func (m *Message) NewTransactionID() error {
	_, err := rand.Read(m.TransactionID[:])
	return err
}

func (m Message) String() string {
	return fmt.Sprintf("%s l=%d attrs=%d id=%s",
		m.Type,
		m.Length,
		len(m.Attributes),
		base64.StdEncoding.EncodeToString(m.TransactionID[:]),
	)
}

// Reset resets Message, attributes and underlying buffer length.
func (m *Message) Reset() {
	m.Raw = m.Raw[:0]
	m.Length = 0
	m.Attributes = m.Attributes[:0]
}

// grow ensures that internal buffer will fit v more bytes and
// increases it capacity if necessary.
func (m *Message) grow(v int) {
	// Not performing any optimizations here
	// (e.g. preallocate len(buf) * 2 to reduce allocations)
	// because they are already done by []byte implementation.
	n := len(m.Raw) + v
	for cap(m.Raw) < n {
		m.Raw = append(m.Raw, 0)
	}
	m.Raw = m.Raw[:n]
}

// Add appends new attribute to message. Not goroutine-safe.
//
// Value of attribute is copied to internal buffer so
// it is safe to reuse v.
func (m *Message) Add(t AttrType, v []byte) {
	// Allocating buffer for TLV (type-length-value).
	// T = t, L = len(v), V = v.
	// m.Raw will look like:
	// [0:20]                               <- message header
	// [20:20+m.Length]                     <- existing message attributes
	// [20+m.Length:20+m.Length+len(v) + 4] <- allocated buffer for new TLV
	// [first:last]                         <- same as previous
	// [0 1|2 3|4    4 + len(v)]            <- mapping for allocated buffer
	//   T   L        V
	allocSize := attributeHeaderSize + len(v)  // len(TLV) = len(TL) + len(V)
	first := messageHeaderSize + int(m.Length) // first byte number
	last := first + allocSize                  // last byte number
	m.grow(last)                               // growing cap(Raw) to fit TLV
	m.Raw = m.Raw[:last]                       // now len(Raw) = last
	m.Length += uint32(allocSize)              // rendering length change

	// Sub-slicing internal buffer to simplify encoding.
	buf := m.Raw[first:last]           // slice for TLV
	value := buf[attributeHeaderSize:] // slice for V
	attr := RawAttribute{
		Type:   t,              // T
		Length: uint16(len(v)), // L
		Value:  value,          // V
	}

	// Encoding attribute TLV to allocated buffer.
	bin.PutUint16(buf[0:2], attr.Type.Value()) // T
	bin.PutUint16(buf[2:4], attr.Length)       // L
	copy(value, v)                             // V

	// Checking that attribute value needs padding.
	if attr.Length%padding != 0 {
		// Performing padding.
		bytesToAdd := nearestPaddedValueLength(len(v)) - len(v)
		last += bytesToAdd
		m.grow(last)
		// setting all padding bytes to zero
		// to prevent data leak from previous
		// data in next bytesToAdd bytes
		buf = m.Raw[last-bytesToAdd : last]
		for i := range buf {
			buf[i] = 0
		}
		m.Raw = m.Raw[:last]           // increasing buffer length
		m.Length += uint32(bytesToAdd) // rendering length change
	}
	m.Attributes = append(m.Attributes, attr)
}

// Equal returns true if Message b equals to m.
// Ignores m.Raw.
func (m *Message) Equal(b *Message) bool {
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
		aB, ok := b.Attributes.Get(a.Type)
		if !ok {
			return false
		}
		if !aB.Equal(a) {
			return false
		}
	}
	return true
}

// WriteLength writes m.Length to m.Raw. Call is valid only if len(m.Raw) >= 4.
func (m *Message) WriteLength() {
	_ = m.Raw[4] // early bounds check to guarantee safety of writes below
	bin.PutUint16(m.Raw[2:4], uint16(m.Length))
}

// WriteHeader writes header to underlying buffer. Not goroutine-safe.
func (m *Message) WriteHeader() {
	if len(m.Raw) < messageHeaderSize {
		// Making WriteHeader call valid even when m.Raw
		// is nil or len(m.Raw) is less than needed for header.
		m.grow(messageHeaderSize)
	}
	_ = m.Raw[:messageHeaderSize] // early bounds check to guarantee safety of writes below

	bin.PutUint16(m.Raw[0:2], m.Type.Value())                       // message type
	bin.PutUint16(m.Raw[2:4], uint16(len(m.Raw)-messageHeaderSize)) // size of payload
	bin.PutUint32(m.Raw[4:8], magicCookie)                          // magic cookie
	copy(m.Raw[8:messageHeaderSize], m.TransactionID[:])            // transaction ID
}

// WriteAttributes encodes all m.Attributes to m.
func (m *Message) WriteAttributes() {
	for _, a := range m.Attributes {
		m.Add(a.Type, a.Value)
	}
}

// Encode resets m.Raw and calls WriteHeader and WriteAttributes.
func (m *Message) Encode() {
	m.Raw = m.Raw[:0]
	m.WriteHeader()
	m.WriteAttributes()
}

// WriteTo implements WriterTo via calling Write(m.Raw) on w and returning
// call result.
func (m *Message) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(m.Raw)
	return int64(n), err
}

// Append appends m.Raw to v. Useful to call after encoding message.
func (m *Message) Append(v []byte) []byte {
	return append(v, m.Raw...)
}

// ReadFrom implements ReaderFrom. Reads message from r into m.Raw,
// Decodes it and return error if any. If m.Raw is too small, will return
// ErrUnexpectedEOF, ErrUnexpectedHeaderEOF or *DecodeErr.
//
// Can return *DecodeErr while decoding too.
func (m *Message) ReadFrom(r io.Reader) (int64, error) {
	tBuf := m.Raw[:cap(m.Raw)]
	var (
		n   int
		err error
	)
	if n, err = r.Read(tBuf); err != nil {
		return int64(n), err
	}
	m.Raw = tBuf[:n]
	return int64(n), m.Decode()
}

const (
	// ErrUnexpectedHeaderEOF means that there were not enough bytes in
	// m.Raw to read header.
	ErrUnexpectedHeaderEOF Error = "unexpected EOF: not enough bytes to read header"
)

// Decode decodes m.Raw into m.
func (m *Message) Decode() error {
	// decoding message header
	buf := m.Raw
	if len(buf) < messageHeaderSize {
		return ErrUnexpectedHeaderEOF
	}
	var (
		t        = binary.BigEndian.Uint16(buf[0:2])      // first 2 bytes
		size     = int(binary.BigEndian.Uint16(buf[2:4])) // second 2 bytes
		cookie   = binary.BigEndian.Uint32(buf[4:8])
		fullSize = messageHeaderSize + size
	)
	if cookie != magicCookie {
		msg := fmt.Sprintf(
			"%x is invalid magic cookie (should be %x)",
			cookie, magicCookie,
		)
		return newDecodeErr("message", "cookie", msg)
	}
	if len(buf) < fullSize {
		msg := fmt.Sprintf(
			"buffer length %d is less than %d (expected message size)",
			len(buf), fullSize,
		)
		return newAttrDecodeErr("message", msg)
	}
	// saving header data
	m.Type.ReadValue(t)
	m.Length = uint32(size)
	copy(m.TransactionID[:], buf[8:messageHeaderSize])

	var (
		offset = 0
		b      = buf[messageHeaderSize:fullSize]
	)
	for offset < size {
		// checking that we have enough bytes to read header
		if len(b) < attributeHeaderSize {
			msg := fmt.Sprintf(
				"buffer length %d is less than %d (expected header size)",
				len(b), attributeHeaderSize,
			)
			return newAttrDecodeErr("header", msg)
		}
		var (
			a = RawAttribute{
				Type:   AttrType(bin.Uint16(b[0:2])), // first 2 bytes
				Length: bin.Uint16(b[2:4]),           // second 2 bytes
			}
			aL     = int(a.Length)                // attribute length
			aBuffL = nearestPaddedValueLength(aL) // expected buffer length (with padding)
		)
		b = b[attributeHeaderSize:] // slicing again to simplify value read
		offset += attributeHeaderSize
		if len(b) < aBuffL { // checking size
			msg := fmt.Sprintf(
				"buffer length %d is less than %d (expected value size)",
				len(b), aBuffL,
			)
			return newAttrDecodeErr("value", msg)
		}
		a.Value = b[:aL]
		offset += aBuffL
		b = b[aBuffL:]

		m.Attributes = append(m.Attributes, a)
	}
	return nil
}

// Write decodes message and return error if any.
//
// Any error is unrecoverable, but message could be partially decoded.
func (m *Message) Write(tBuf []byte) (int, error) {
	m.Raw = append(m.Raw[:0], tBuf...)
	return len(tBuf), m.Decode()
}

// MaxPacketSize is maximum size of UDP packet that is processable in
// this package for STUN message.
const MaxPacketSize = 2048

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
	MethodBinding          Method = 0x001
	MethodAllocate         Method = 0x003
	MethodRefresh          Method = 0x004
	MethodSend             Method = 0x006
	MethodData             Method = 0x007
	MethodCreatePermission Method = 0x008
	MethodChannelBind      Method = 0x009
)

func (m Method) String() string {
	switch m {
	case MethodBinding:
		return "binding"
	case MethodAllocate:
		return "allocate"
	case MethodRefresh:
		return "refresh"
	case MethodSend:
		return "send"
	case MethodData:
		return "data"
	case MethodCreatePermission:
		return "create permission"
	case MethodChannelBind:
		return "channel bind"
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

	// Warning: Abandon all hope ye who enter here.
	// Splitting M into A(M0-M3), B(M4-M6), D(M7-M11).
	m := uint16(t.Method)
	a := m & methodABits // A = M * 0b0000000000001111 (right 4 bits)
	b := m & methodBBits // B = M * 0b0000000001110000 (3 bits after A)
	d := m & methodDBits // D = M * 0b0000111110000000 (5 bits after B)

	// Shifting to add "holes" for C0 (at 4 bit) and C1 (8 bit).
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
	// Decoding class.
	// We are taking first bit from v >> 4 and second from v >> 7.
	c0 := (v >> classC0Shift) & c0Bit
	c1 := (v >> classC1Shift) & c1Bit
	class := c0 + c1
	t.Class = MessageClass(class)

	// Decoding method.
	a := v & methodABits                   // A(M0-M3)
	b := (v >> methodBShift) & methodBBits // B(M4-M6)
	d := (v >> methodDShift) & methodDBits // D(M7-M11)
	m := a + b + d
	t.Method = Method(m)
}

func (t MessageType) String() string {
	return fmt.Sprintf("%s %s", t.Method, t.Class)
}
