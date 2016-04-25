package stun

import (
	"testing"

	log "github.com/Sirupsen/logrus"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

func TestMessageType_Value(t *testing.T) {
	var tests = []struct {
		in  messageType
		out uint16
	}{
		{messageType{Method: methodBinding, Class: classRequest}, 0x0001},
		{messageType{Method: methodBinding, Class: classSuccessResponse}, 0x0101},
		{messageType{Method: methodBinding, Class: classErrorResponse}, 0x0111},
		{messageType{Method: 0xb6d, Class: 0x3}, 0x2ddd},
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
		out messageType
	}{
		{0x0001, messageType{Method: methodBinding, Class: classRequest}},
		{0x0101, messageType{Method: methodBinding, Class: classSuccessResponse}},
		{0x0111, messageType{Method: methodBinding, Class: classErrorResponse}},
	}
	for _, tt := range tests {
		m := messageType{}
		m.ReadValue(tt.in)
		if m != tt.out {
			t.Errorf("ReadValue(%s) -> %s, want %s", bUint16(tt.in), m, tt.out)
		}
	}
}

func TestMessageType_ReadWriteValue(t *testing.T) {
	var tests = []messageType{
		{Method: methodBinding, Class: classRequest},
		{Method: methodBinding, Class: classSuccessResponse},
		{Method: methodBinding, Class: classErrorResponse},
		{Method: 0x12, Class: classErrorResponse},
	}
	for _, tt := range tests {
		m := messageType{}
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
	mType := messageType{Method: methodBinding, Class: classRequest}
	messageAttribute := attribute{Length: 2, Value: []byte{1, 2}, Type: 0x1}
	messageAttributes := attributes{
		messageAttribute,
	}
	m := message{
		Type:          mType,
		Length:        6,
		TransactionID: [transactionIDSize]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		Attributes:    messageAttributes,
	}
	buf := make([]byte, 128)
	m.Put(buf)
	mDecoded := message{}
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
	mDecoded := message{}
	if err := mDecoded.Get(buf); err != ErrInvalidMagicCookie {
		t.Error("should error")
	}
}

func BenchmarkMessageType_Value(b *testing.B) {
	m := messageType{Method: methodBinding, Class: classRequest}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m.Value()
	}
}

func BenchmarkMessage_Put(b *testing.B) {
	mType := messageType{Method: methodBinding, Class: classRequest}
	m := message{
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

