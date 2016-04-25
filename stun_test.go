package stun

import (
	"fmt"
	"strconv"
	"testing"

	log "github.com/Sirupsen/logrus"
)

func bUint16(v uint16) string {
	return fmt.Sprintf("0b%016s", strconv.FormatUint(uint64(v), 2))
}

func init() {
	log.SetLevel(log.DebugLevel)
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
	if err := mDecoded.Get(buf); err != ErrInvalidMagicCookie {
		t.Error("should error")
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
