package stun

import (
	"fmt"
	"math/rand"
	"net"

	"github.com/pkg/errors"
)

// MessageClass of 0b00 is a request, a class of 0b01 is an
//   indication, a class of 0b10 is a success response, and a class of
//   0b11 is an error response.
// https://tools.ietf.org/html/rfc5389#section-6
type MessageClass byte

const (
	// ClassRequest describes a request method type
	ClassRequest MessageClass = 0x00
	// ClassIndication describes an indication method type
	ClassIndication MessageClass = 0x01
	// ClassSuccessResponse describes an success response method type
	ClassSuccessResponse MessageClass = 0x02
	// ClassErrorResponse describes an error response method type
	ClassErrorResponse MessageClass = 0x03
)

// Method is selector which can be select MethodType
type Method uint16

//Method Type
const (
	MethodBinding          Method = 0x01 // STUN
	MethodSharedSecret     Method = 0x02 // STUN
	MethodAllocate         Method = 0x03 // TURN (Req/Rsp)
	MethodRefresh          Method = 0x04 // TURN (Req/Rsp)
	MethodSend             Method = 0x06 // TURN (Ind)
	MethodData             Method = 0x07 // TURN (Ind)
	MethodCreatePermission Method = 0x08 // TURN (Req/Rsp)
	MethodChannelBind      Method = 0x09 // TURN (Req/Rsp)
)

var messageClassName = map[MessageClass]string{
	ClassRequest:         "REQUEST",
	ClassIndication:      "INDICATION",
	ClassSuccessResponse: "SUCCESS-RESPONSE",
	ClassErrorResponse:   "ERROR-RESPONSE",
}

// String prints the known class names and a hex format for unknown class names
func (m MessageClass) String() string {
	s, err := messageClassName[m]
	if !err {
		// Falling back to hex representation.
		s = fmt.Sprintf("Unk 0x%x", uint16(m))
	}
	return s
}

var methodName = map[Method]string{
	MethodBinding:          "BINDING",
	MethodSharedSecret:     "SHARED-SECRET",
	MethodAllocate:         "ALLOCATE",
	MethodRefresh:          "REFRESH",
	MethodSend:             "SEND",
	MethodData:             "DATA",
	MethodCreatePermission: "CREATE-PERMISSION",
	MethodChannelBind:      "CHANNEL-BIND",
}

// String prints the known method names and a hex format for unknown method names
func (m Method) String() string {
	s, err := methodName[m]
	if !err {
		s = fmt.Sprintf("Unk 0x%x", uint16(m))
	}
	return s
}

//       0                   1                   2                   3
//       0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
//      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//      |0 0|     STUN Message Type     |         Message Length        |
//      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//      |                         Magic Cookie                          |
//      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//      |                                                               |
//      |                     Transaction ID (96 bits)                  |
//      |                                                               |
//      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//

const (
	messageHeaderStart  int = 0
	messageHeaderLength int = 20
	messageLengthStart  int = 2
	messageLengthLength int = 2
	magicCookieStart    int = 4
	magicCookieLength   int = 4
	transactionIDStart  int = 4
	transactionIDLength int = 16

	// TransactionIDSize is the size of the transaction according to RFC 5389: 96 bits
	TransactionIDSize = 96 / 8
)

// Message structs
type Message struct {
	Class         MessageClass
	Method        Method
	Length        uint16
	TransactionID []byte
	Attributes    []*RawAttribute
	Raw           []byte
}

// The most significant 2 bits of every STUN message MUST be zeroes.
// This can be used to differentiate STUN packets from other protocols
// when STUN is multiplexed with other protocols on the same port.
// https://tools.ietf.org/html/rfc5389#section-6
func verifyStunHeaderMostSignificant2Bits(header []byte) bool {
	return (header[0] >> 6) == 0
}

func verifyMagicCookie(header []byte) error {
	const magicCookie = 0x2112A442
	c := header[magicCookieStart : magicCookieStart+magicCookieLength]
	if enc.Uint32(c) != magicCookie {
		return errors.Errorf("stun header magic cookie invalid; %v != %v (expected)", enc.Uint32(c), magicCookie)
	}
	return nil
}

// The message length MUST contain the size, in bytes, of the message
// not including the 20-byte STUN header.  Since all STUN attributes are
// padded to a multiple of 4 bytes, the last 2 bits of this field are
// always zero.  This provides another way to distinguish STUN packets
// from packets of other protocols.
// https://tools.ietf.org/html/rfc5389#section-6
func getMessageLength(header []byte) (uint16, error) {
	messageLength := enc.Uint16(header[messageLengthStart : messageLengthStart+messageLengthLength])
	if messageLength%4 != 0 {
		return 0, errors.Errorf("stun header message length must be a factor of 4 (%d)", messageLength)
	}

	return messageLength, nil
}

//  0                 1
//  2  3  4 5 6 7 8 9 0 1 2 3 4 5
//
// +--+--+-+-+-+-+-+-+-+-+-+-+-+-+
// |M |M |M|M|M|C|M|M|M|C|M|M|M|M|
// |11|10|9|8|7|1|6|5|4|0|3|2|1|0|
// +--+--+-+-+-+-+-+-+-+-+-+-+-+-+
const (
	c0Mask   = 0x10 // 0b10000
	c1Mask   = 0x01 // 0b00001
	c0ShiftR = 4    // R 0b10000 -> 0b00001
	c1ShiftL = 1    // L 0b00001 -> 0b00010

	m0Mask   = 0x0F // 0b00001111
	m4Mask   = 0xE0 // 0b11100000
	m7Mask   = 0x3E // 0b00111110
	m4ShiftR = 1    // R 0b01110000 -> 0b00111000
	m7ShiftL = 5    // L 0b00111110 -> 0b0000011111000000
)

func setMessageType(header []byte, class MessageClass, method Method) {
	m := uint16(method)
	c := uint16(class)

	mt := m & m0Mask
	// Make room for c0
	mt |= (m & (m4Mask >> m4ShiftR)) << 1
	mt |= (m & (m7Mask << 6)) << 2
	mt |= (c & 0x1) << 4
	mt |= (c >> 1) << 8

	enc.PutUint16(header[messageHeaderStart:], mt)
}

func getMessageType(header []byte) (MessageClass, Method) {
	mByte0 := header[0]
	mByte1 := header[1]

	c0 := (mByte1 & c0Mask) >> c0ShiftR
	c1 := (mByte0 & c1Mask) << c1ShiftL

	class := MessageClass(c1 | c0)

	m := (uint16(mByte0) & m7Mask) << m7ShiftL
	m |= uint16(mByte1 & m0Mask)
	m |= uint16((mByte1 & m4Mask) >> m4ShiftR)

	method := Method(m)

	return class, method
}

//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |         Type                  |            Length             |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                         Value (variable)                ....
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
func getAttribute(attribute []byte, offset int) *RawAttribute {
	typ := AttrType(enc.Uint16(attribute))
	len := enc.Uint16(attribute[attrLengthStart : attrLengthStart+attrLengthLength])
	pad := (attrLengthMultiple - (len % attrLengthMultiple)) % attrLengthMultiple
	return &RawAttribute{typ, len, attribute[attrValueStart : attrValueStart+len], pad, offset}
}

// IsSTUN determines if a package is likely a STUN package
// Used for de-multiplexing STUN packages
func IsSTUN(packet []byte) bool {
	if len(packet) < 20 {
		return false
	}

	header := packet[messageHeaderStart : messageHeaderStart+messageHeaderLength]

	if !verifyStunHeaderMostSignificant2Bits(header) {
		return false
	}

	return verifyMagicCookie(header) == nil
}

// NewMessage parses a binary STUN message into a Message struct
// TODO Break this apart, too big
func NewMessage(packet []byte) (*Message, error) {
	buf := make([]byte, len(packet))
	copy(buf, packet)
	if len(buf) < 20 {
		return nil, errors.Errorf("stun header must be at least 20 bytes, was %d", len(buf))
	}

	header := buf[messageHeaderStart : messageHeaderStart+messageHeaderLength]

	if !verifyStunHeaderMostSignificant2Bits(header) {
		return nil, errors.New("stun header most significant 2 bits must equal 0b00")
	}

	err := verifyMagicCookie(header)
	if err != nil {
		return nil, errors.Wrap(err, "stun header invalid")
	}

	ml, err := getMessageLength(header)
	if err != nil {
		return nil, errors.Wrap(err, "stun header invalid")
	}

	if len(buf) != messageHeaderLength+int(ml) {
		return nil, errors.Errorf("stun header length invalid; %d != %d (expected)", messageHeaderLength+int(ml), len(buf))
	}

	t := header[transactionIDStart : transactionIDStart+transactionIDLength]

	class, method := getMessageType(header)

	ra := []*RawAttribute{}
	// TODO Check attr length <= attr slice remaining
	attr := buf[messageHeaderLength:]
	for len(attr) > 0 {
		a := getAttribute(attr, cap(buf)-cap(attr))
		attr = attr[attrValueStart+a.Length+a.Pad:]
		ra = append(ra, a)
	}

	m := Message{}
	m.Class = class
	m.Method = method
	m.Length = ml
	m.TransactionID = t[0:transactionIDLength]
	m.Attributes = ra
	m.Raw = buf

	return &m, nil
}

// GetOneAttribute can get a RawAttribute which adopts attrbute type
func (m *Message) GetOneAttribute(attrType AttrType) (*RawAttribute, bool) {
	for _, v := range m.Attributes {
		if v.Type == attrType {
			return v, true
		}
	}

	return nil, false
}

// GetAllAttributes can get all RawAttributes which adopt attrbute type
func (m *Message) GetAllAttributes(attrType AttrType) ([]*RawAttribute, bool) {
	var attrs []*RawAttribute
	for _, v := range m.Attributes {
		if v.Type == attrType {
			attrs = append(attrs, v)
		}
	}

	return attrs, len(attrs) > 0
}

// CommitLength returns message length
func (m *Message) CommitLength() {
	enc.PutUint16(m.Raw[messageLengthStart:], m.Length)
}

// AddAttribute append bytes formatted RawAttribute to message
func (m *Message) AddAttribute(attrType AttrType, v []byte) {

	ra := RawAttribute{
		Type:   attrType,
		Value:  v,
		Pad:    uint16(getAttrPadding(len(v))),
		Length: uint16(len(v)),
		Offset: int(m.Length),
	}

	a := make([]byte, attrHeaderLength+ra.Length+ra.Pad)

	enc.PutUint16(a, uint16(ra.Type))
	enc.PutUint16(a[attrLengthStart:attrLengthStart+attrLengthLength], ra.Length)

	copy(a[attrValueStart:], ra.Value)

	m.Attributes = append(m.Attributes, &ra)
	m.Raw = append(m.Raw, a...)
	m.Length += uint16(len(a))
	m.CommitLength()
}

// Pack returns Message.Raw
func (m *Message) Pack() []byte {

	setMessageType(m.Raw[messageHeaderStart:2], m.Class, m.Method)
	m.CommitLength()
	copy(m.Raw[transactionIDStart:], m.TransactionID)

	return m.Raw
}

// BuildAndSend is building message, pack using attribute and send
func BuildAndSend(conn net.PacketConn, addr *TransportAddr, class MessageClass, method Method, transactionID []byte, attrs ...Attribute) error {
	rsp, err := Build(class, method, transactionID, attrs...)
	if err != nil {
		return err
	}

	b := rsp.Pack()
	l, err := conn.WriteTo(b, addr.Addr())
	if err != nil {
		return errors.Wrap(err, "failed writing to socket")
	}

	if l != len(b) {
		return errors.Errorf("packet write smaller than packet %d != %d (expected)", l, len(b))
	}

	return nil
}

// GenerateTransactionID returns 16bytes ids
func GenerateTransactionID() []byte {
	randSeq := func(n int) string {
		letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
		b := make([]rune, n)
		for i := range b {
			b[i] = letters[rand.Intn(len(letters))]
		}
		return string(b)
	}

	transactionID := []byte(randSeq(16))
	transactionID[0] = 33
	transactionID[1] = 18
	transactionID[2] = 164
	transactionID[3] = 66
	return transactionID
}
