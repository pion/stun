package stun

import (
	"fmt"
	"strconv"
	"testing"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"io"
	"encoding/binary"
	"strings"
)

func bUint16(v uint16) string {
	return fmt.Sprintf("0b%016s", strconv.FormatUint(uint64(v), 2))
}

func init() {
	log.SetLevel(log.DebugLevel)
}

func TestMessageCopy(t *testing.T) {
	m := AcquireMessage()
	defer ReleaseMessage(m)
	m.Type = MessageType{Method: MethodBinding, Class: ClassRequest}
	m.TransactionID = NewTransactionID()
	m.Add(AttrErrorCode, []byte{0xff, 0xfe, 0xfa})
	m.WriteHeader()
	mCopy := m.Clone()
	if !mCopy.Equal(*m) {
		t.Error(mCopy, "!=", m)
	}
}

func TestMessageBuffer(t *testing.T) {
	m := AcquireMessage()
	defer ReleaseMessage(m)
	m.Type = MessageType{Method: MethodBinding, Class: ClassRequest}
	m.TransactionID = NewTransactionID()
	m.Add(AttrErrorCode, []byte{0xff, 0xfe, 0xfa})
	m.WriteHeader()
	mDecoded := &Message{}
	if err := mDecoded.Get(m.buf.B); err != nil {
		t.Error(err)
	}
	if !mDecoded.Equal(*m) {
		t.Error(mDecoded, "!", m)
	}
}

func BenchmarkMessage_Write(b *testing.B) {
	b.ReportAllocs()
	attributeValue := []byte{0xff, 0x11, 0x12, 0x34}
	b.SetBytes(int64(len(attributeValue) + messageHeaderSize +
		attributeHeaderSize))
	transactionID := NewTransactionID()

	for i := 0; i < b.N; i++ {
		m := AcquireMessage()
		m.Add(AttrErrorCode, attributeValue)
		m.TransactionID = transactionID
		m.Type = MessageType{Method: MethodBinding, Class: ClassRequest}
		m.WriteHeader()
		ReleaseMessage(m)
	}
}

func TestMessageType_Value(t *testing.T) {
	var tests = []struct {
		in  MessageType
		out uint16
	}{
		{MessageType{Method: MethodBinding, Class: ClassRequest}, 0x0001},
		{MessageType{Method: MethodBinding, Class: ClassSuccessResponse}, 0x0101},
		{MessageType{Method: MethodBinding, Class: ClassErrorResponse}, 0x0111},
		{MessageType{Method: 0xb6d, Class: 0x3}, 0x2ddd},
	}
	for _, tt := range tests {
		b := tt.in.Value()
		if b != tt.out {
			t.Errorf("Value(%s) -> %s, want %s", tt.in, bUint16(b), bUint16(tt.out))
		}
	}
}

func TestMessageType_ReadValue(t *testing.T) {
	var tests = []struct {
		in  uint16
		out MessageType
	}{
		{0x0001, MessageType{Method: MethodBinding, Class: ClassRequest}},
		{0x0101, MessageType{Method: MethodBinding, Class: ClassSuccessResponse}},
		{0x0111, MessageType{Method: MethodBinding, Class: ClassErrorResponse}},
	}
	for _, tt := range tests {
		m := MessageType{}
		m.ReadValue(tt.in)
		if m != tt.out {
			t.Errorf("ReadValue(%s) -> %s, want %s", bUint16(tt.in), m, tt.out)
		}
	}
}

func TestMessageType_ReadWriteValue(t *testing.T) {
	var tests = []MessageType{
		{Method: MethodBinding, Class: ClassRequest},
		{Method: MethodBinding, Class: ClassSuccessResponse},
		{Method: MethodBinding, Class: ClassErrorResponse},
		{Method: 0x12, Class: ClassErrorResponse},
	}
	for _, tt := range tests {
		m := MessageType{}
		v := tt.Value()
		m.ReadValue(v)
		if m != tt {
			t.Errorf("ReadValue(%s -> %s) = %s, should be %s", tt, bUint16(v), m, tt)
			if m.Method != tt.Method {
				t.Errorf("%s != %s", bUint16(uint16(m.Method)), bUint16(uint16(tt.Method)))
			}
		}
	}
}

func TestMessage_PutGet(t *testing.T) {
	mType := MessageType{Method: MethodBinding, Class: ClassRequest}
	messageAttribute := Attribute{Length: 2, Value: []byte{1, 2}, Type: 0x1}
	messageAttributes := Attributes{
		messageAttribute,
	}
	m := Message{
		Type:          mType,
		Length:        6,
		TransactionID: [transactionIDSize]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		Attributes:    messageAttributes,
	}
	buf := make([]byte, 128)
	m.Put(buf)
	mDecoded := Message{}
	if err := mDecoded.Get(buf); err != nil {
		t.Error(err)
	}
	if mDecoded.Type != m.Type {
		t.Error("incorrect type")
	}
	if mDecoded.Length != m.Length {
		t.Error("incorrect length")
	}
	if mDecoded.TransactionID != m.TransactionID {
		t.Error("incorrect transaction ID")
	}
	aDecoded := mDecoded.Attributes.Get(messageAttribute.Type)
	if !aDecoded.Equal(messageAttribute) {
		t.Error(aDecoded, "!=", messageAttribute)
	}
}

func TestMessage_Cookie(t *testing.T) {
	buf := make([]byte, 20)
	mDecoded := Message{}
	if err := mDecoded.Get(buf); err == nil {
		t.Error("should error")
	}
}

func TestMessage_LengthLessHeaderSize(t *testing.T) {
	buf := make([]byte, 8)
	mDecoded := Message{}
	if err := mDecoded.Get(buf); err == nil {
		t.Error("should error")
	}
}

func TestMessage_BadLength(t *testing.T) {
	mType := MessageType{Method: MethodBinding, Class: ClassRequest}
	messageAttribute := Attribute{Length: 2, Value: []byte{1, 2}, Type: 0x1}
	messageAttributes := Attributes{
		messageAttribute,
	}
	m := Message{
		Type:          mType,
		Length:        4,
		TransactionID: [transactionIDSize]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		Attributes:    messageAttributes,
	}
	buf := make([]byte, 128)
	m.Put(buf)
	mDecoded := Message{}
	if err := mDecoded.Get(buf[:20+3]); err == nil {
		t.Error("should error")
	}
}

func TestMessage_AttrLengthLessThanHeader(t *testing.T) {
	mType := MessageType{Method: MethodBinding, Class: ClassRequest}
	messageAttribute := Attribute{Length: 2, Value: []byte{1, 2}, Type: 0x1}
	messageAttributes := Attributes{
		messageAttribute,
	}
	m := Message{
		Type:          mType,
		TransactionID: NewTransactionID(),
		Attributes:    messageAttributes,
	}
	buf := make([]byte, 128)
	m.Put(buf)
	binary.BigEndian.PutUint16(buf[2:4], 2) // rewrite to bad length
	mDecoded := Message{}
	err := mDecoded.Get(buf[:20+2])
	if errors.Cause(err) != io.ErrUnexpectedEOF {
		t.Error(err, "should be", io.ErrUnexpectedEOF)
	}
}

func TestMessage_AttrSizeLessThanLength(t *testing.T) {
	mType := MessageType{Method: MethodBinding, Class: ClassRequest}
	messageAttribute := Attribute{Length: 2, Value: []byte{1, 2}, Type: 0x1}
	messageAttributes := Attributes{
		messageAttribute,
	}
	m := Message{
		Type:          mType,
		TransactionID: NewTransactionID(),
		Attributes:    messageAttributes,
	}
	buf := make([]byte, 128)
	m.Put(buf)
	binary.BigEndian.PutUint16(buf[2:4], 2) // rewrite to bad length
	mDecoded := Message{}
	err := mDecoded.Get(buf[:20+5])
	if errors.Cause(err) != io.ErrUnexpectedEOF {
		t.Error(err, "should be", io.ErrUnexpectedEOF)
	}
}



func BenchmarkMessageType_Value(b *testing.B) {
	m := MessageType{Method: MethodBinding, Class: ClassRequest}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m.Value()
	}
}

func BenchmarkMessage_Put(b *testing.B) {
	mType := MessageType{Method: MethodBinding, Class: ClassRequest}
	m := Message{
		Type:          mType,
		Length:        0,
		TransactionID: [transactionIDSize]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
	}
	buf := make([]byte, 20)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m.Put(buf)
	}
}

func TestMessageClass_String(t *testing.T) {
	defer func() {
		if err := recover(); err == nil {
			t.Error(err, "should be not nil")
		}
	}()

	v := [...]MessageClass{
		ClassRequest,
		ClassErrorResponse,
		ClassSuccessResponse,
		ClassIndication,
	}
	for _, k := range v {
		if len(k.String()) == 0 {
			t.Error(k, "bad stringer")
		}
	}

	// should panic
	MessageClass(0x05).String()
}

func TestAttrType_String(t *testing.T) {
	v := [...]AttrType{
		AttrMappedAddress,
		AttrUsername,
		AttrErrorCode,
		AttrMessageIntegrity,
		AttrUnknownAttributes,
		AttrRealm,
		AttrNonce,
		AttrXORMappedAddress,
		AttrSoftware,
		AttrAlternateServer,
		AttrFingerprint,
	}
	for _, k := range v {
		if len(k.String()) == 0 {
			t.Error(k, "bad stringer")
		}
		if strings.HasPrefix(k.String(), "0x") {
			t.Error(k, "bad stringer")
		}
	}
	vNonStandard := AttrType(0x512)
	if !strings.HasPrefix(vNonStandard.String(), "0x512") {
		t.Error(vNonStandard, "bad prefix")
	}
}

func TestMethod_String(t *testing.T) {
	if MethodBinding.String() != "binding" {
		t.Error("binding is not binding!")
	}
	if Method(0x616).String() != "0x616" {
		t.Error("Bad stringer", Method(0x616))
	}
}

func TestMessageReadOnly(t *testing.T) {
	defer func() {
		if err := recover(); err == nil {
			t.Error(err, "should be not nil")
		}
	}()
	m := Message{readOnly: true}
	m.mustWrite()
}

func TestAttribute_Equal(t *testing.T) {
	a := Attribute{Length: 2, Value:[]byte{0x1, 0x2}}
	b := Attribute{Length: 2, Value:[]byte{0x1, 0x2}}
	if !a.Equal(b) {
		t.Error("should equal")
	}
	if a.Equal(Attribute{Type: 0x2}) {
		t.Error("should not equal")
	}
	if a.Equal(Attribute{Length: 0x2}) {
		t.Error("should not equal")
	}
	if a.Equal(Attribute{Length: 0x3}) {
		t.Error("should not equal")
	}
	if a.Equal(Attribute{Length: 2, Value:[]byte{0x1, 0x3}}) {
		t.Error("should not equal")
	}
}

func TestMessage_Equal(t *testing.T) {
	attr := Attribute{Length: 2, Value:[]byte{0x1, 0x2}}
	attrs := Attributes{attr}
	a := Message{Attributes: attrs, Length: 4+2}
	b := Message{Attributes: attrs, Length: 4+2}
	if !a.Equal(b) {
		t.Error("should equal")
	}
	if a.Equal(Message{Type: MessageType{Class: 128}}) {
		t.Error("should not equal")
	}
	tID := [transactionIDSize]byte{
		1,2,3,4,5,6,7,8,9,10,11,12,
	}
	if a.Equal(Message{TransactionID: tID}) {
		t.Error("should not equal")
	}
	if a.Equal(Message{Length: 3}) {
		t.Error("should not equal")
	}
	tAttrs := Attributes{
		{Length: 1, Value: []byte{0x1}},
	}
	if a.Equal(Message{Attributes: tAttrs, Length: 4+2}) {
		t.Error("should not equal")
	}
}

func TestMessageGrow(t *testing.T) {
	m := AcquireMessage()
	defer ReleaseMessage(m)
	m.grow(512)
}