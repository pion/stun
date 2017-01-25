package stun

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
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
	mDecoded := AcquireMessage()
	if _, err := mDecoded.ReadFrom(bytes.NewReader(m.buf.B)); err != nil {
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

func TestMessage_WriteTo(t *testing.T) {
	m := AcquireMessage()
	defer ReleaseMessage(m)
	m.Type = MessageType{Method: MethodBinding, Class: ClassRequest}
	m.TransactionID = NewTransactionID()
	m.Add(AttrErrorCode, []byte{0xff, 0xfe, 0xfa})
	m.WriteHeader()
	buf := new(bytes.Buffer)
	if _, err := m.WriteTo(buf); err != nil {
		t.Fatal(err)
	}
	mDecoded := AcquireMessage()
	if _, err := mDecoded.ReadFrom(buf); err != nil {
		t.Error(err)
	}
	if !mDecoded.Equal(*m) {
		t.Error(mDecoded, "!", m)
	}
}

func TestMessage_Cookie(t *testing.T) {
	buf := make([]byte, 20)
	mDecoded := AcquireMessage()
	if _, err := mDecoded.ReadFrom(bytes.NewReader(buf)); err == nil {
		t.Error("should error")
	}
}

func TestMessage_LengthLessHeaderSize(t *testing.T) {
	buf := make([]byte, 8)
	mDecoded := AcquireMessage()
	if _, err := mDecoded.ReadFrom(bytes.NewReader(buf)); err == nil {
		t.Error("should error")
	}
}

func TestMessage_BadLength(t *testing.T) {
	mType := MessageType{Method: MethodBinding, Class: ClassRequest}
	messageAttribute := Attribute{Length: 2, Value: []byte{1, 2}, Type: 0x1}
	m := AcquireFields(Message{
		Type:          mType,
		Length:        4,
		TransactionID: [transactionIDSize]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		Attributes:    []Attribute{messageAttribute},
	})
	buf := make([]byte, 0, 100)
	m.WriteHeader()
	buf = m.Append(buf)
	buf[20+3] = 10 // set attr length = 10
	mDecoded := AcquireMessage()
	if _, err := mDecoded.ReadBytes(buf); err == nil {
		t.Error("should error")
	}
}

func TestMessage_AttrLengthLessThanHeader(t *testing.T) {
	mType := MessageType{Method: MethodBinding, Class: ClassRequest}
	messageAttribute := Attribute{Length: 2, Value: []byte{1, 2}, Type: 0x1}
	messageAttributes := Attributes{
		messageAttribute,
	}
	m := AcquireFields(Message{
		Type:          mType,
		TransactionID: NewTransactionID(),
		Attributes:    messageAttributes,
	})
	b := new(bytes.Buffer)
	m.WriteTo(b)
	buf := b.Bytes()
	mDecoded := AcquireMessage()
	binary.BigEndian.PutUint16(buf[2:4], 2) // rewrite to bad length
	_, err := mDecoded.ReadFrom(bytes.NewReader(buf[:20+2]))
	switch e := errors.Cause(err).(type) {
	case DecodeErr:
		if !e.IsPlace(DecodeErrPlace{"attribute", "header"}) {
			t.Error(e, "bad place")
		}
	default:
		t.Error(err, "should be bad format")
	}
}

func TestMessage_AttrSizeLessThanLength(t *testing.T) {
	mType := MessageType{Method: MethodBinding, Class: ClassRequest}
	messageAttribute := Attribute{Length: 4,
		Value: []byte{1, 2, 3, 4}, Type: 0x1,
	}
	messageAttributes := Attributes{
		messageAttribute,
	}
	m := AcquireFields(Message{
		Type:          mType,
		TransactionID: NewTransactionID(),
		Attributes:    messageAttributes,
	})
	b := new(bytes.Buffer)
	m.WriteTo(b)
	buf := b.Bytes()
	binary.BigEndian.PutUint16(buf[2:4], 5) // rewrite to bad length
	mDecoded := AcquireMessage()
	_, err := mDecoded.ReadFrom(bytes.NewReader(buf[:20+5]))
	switch e := errors.Cause(err).(type) {
	case DecodeErr:
		if !e.IsPlace(DecodeErrPlace{"attribute", "value"}) {
			t.Error(e, "bad place")
		}
	default:
		t.Error(err, "should be bad format")
	}
}

type unexpectedEOFReader struct{}

func (r unexpectedEOFReader) Read(b []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func TestMessage_ReadFromError(t *testing.T) {
	mDecoded := AcquireMessage()
	_, err := mDecoded.ReadFrom(unexpectedEOFReader{})
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

func BenchmarkMessage_WriteTo(b *testing.B) {
	mType := MessageType{Method: MethodBinding, Class: ClassRequest}
	m := AcquireFields(Message{
		Type:   mType,
		Length: 0,
		TransactionID: [transactionIDSize]byte{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
		},
	})
	buf := new(bytes.Buffer)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m.WriteTo(buf)
		buf.Reset()
	}
}

func BenchmarkMessage_ReadFrom(b *testing.B) {
	mType := MessageType{Method: MethodBinding, Class: ClassRequest}
	m := AcquireFields(Message{
		Type:   mType,
		Length: 0,
		TransactionID: [transactionIDSize]byte{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
		},
	})
	buf := new(bytes.Buffer)
	m.WriteTo(buf)
	b.ReportAllocs()
	tBuf := buf.Bytes()
	b.SetBytes(int64(len(tBuf)))
	reader := bytes.NewReader(tBuf)
	mRec := AcquireMessage()
	for i := 0; i < b.N; i++ {
		if _, err := mRec.ReadFrom(reader); err != nil {
			b.Fatal(err)
		}
		reader.Reset(tBuf)
		mRec.Reset()
	}
}

func BenchmarkMessage_ReadBytes(b *testing.B) {
	mType := MessageType{Method: MethodBinding, Class: ClassRequest}
	m := AcquireFields(Message{
		Type:   mType,
		Length: 0,
		TransactionID: [transactionIDSize]byte{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
		},
	})
	buf := new(bytes.Buffer)
	m.WriteTo(buf)
	b.ReportAllocs()
	tBuf := buf.Bytes()
	b.SetBytes(int64(len(tBuf)))
	mRec := AcquireMessage()
	for i := 0; i < b.N; i++ {
		if _, err := mRec.ReadBytes(tBuf); err != nil {
			b.Fatal(err)
		}
		mRec.Reset()
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
	p := MessageClass(0x05).String()
	t.Error("should panic!", p)
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
	a := Attribute{Length: 2, Value: []byte{0x1, 0x2}}
	b := Attribute{Length: 2, Value: []byte{0x1, 0x2}}
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
	if a.Equal(Attribute{Length: 2, Value: []byte{0x1, 0x3}}) {
		t.Error("should not equal")
	}
}

func TestMessage_Equal(t *testing.T) {
	attr := Attribute{Length: 2, Value: []byte{0x1, 0x2}}
	attrs := Attributes{attr}
	a := Message{Attributes: attrs, Length: 4 + 2}
	b := Message{Attributes: attrs, Length: 4 + 2}
	if !a.Equal(b) {
		t.Error("should equal")
	}
	if a.Equal(Message{Type: MessageType{Class: 128}}) {
		t.Error("should not equal")
	}
	tID := [transactionIDSize]byte{
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
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
	if a.Equal(Message{Attributes: tAttrs, Length: 4 + 2}) {
		t.Error("should not equal")
	}
}

func TestMessageGrow(t *testing.T) {
	m := AcquireMessage()
	defer ReleaseMessage(m)
	m.grow(512)
	if len(m.buf.B) < 532 {
		t.Error("Bad length", len(m.buf.B))
	}
}

func TestMessageGrowSmaller(t *testing.T) {
	m := AcquireMessage()
	defer ReleaseMessage(m)
	m.grow(2)
	if cap(m.buf.B) < 22 {
		t.Error("Bad capacity", cap(m.buf.B))
	}
	if len(m.buf.B) < 22 {
		t.Error("Bad length", len(m.buf.B))
	}
}

func TestMessage_String(t *testing.T) {
	m := AcquireMessage()
	defer ReleaseMessage(m)
	if len(m.String()) == 0 {
		t.Error("bad string")
	}
}

func TestIsMessage(t *testing.T) {
	m := AcquireMessage()
	defer ReleaseMessage(m)
	m.AddSoftware("test")
	m.WriteHeader()

	mBlank := AcquireMessage()
	defer ReleaseMessage(mBlank)
	mBlank.WriteHeader()
	var tt = [...]struct {
		in  []byte
		out bool
	}{
		{nil, false},                                // 0
		{[]byte{1, 2, 3}, false},                    // 1
		{[]byte{1, 2, 4}, false},                    // 2
		{[]byte{1, 2, 4, 5, 6, 7, 8, 9, 20}, false}, // 3
		{mBlank.buf.B, true},                        // 4
		{m.buf.B, true},                             // 5
		{[]byte{0, 0, 0, 0, 33, 18,
			164, 66, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0}, true}, // 6
	}
	for i, v := range tt {
		if got := IsMessage(v.in); got != v.out {
			t.Errorf("tt[%d]: IsMessage(%+v) %v != %v", i, v.in, got, v.out)
		}
	}
}

func BenchmarkIsMessage(b *testing.B) {
	m := AcquireMessage()
	defer ReleaseMessage(m)
	m.Type = MessageType{Method: MethodBinding, Class: ClassRequest}
	m.TransactionID = NewTransactionID()
	m.AddSoftware("cydev/stun test")
	m.WriteHeader()

	b.SetBytes(int64(messageHeaderSize))
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if !IsMessage(m.buf.B) {
			b.Fatal("Should be message")
		}
	}
}

func loadData(tb testing.TB, name string) []byte {
	name = filepath.Join("testdata", name)
	f, err := os.Open(name)
	if err != nil {
		tb.Fatal(err)
	}
	defer func() {
		if errClose := f.Close(); errClose != nil {
			tb.Fatal(errClose)
		}
	}()
	v, err := ioutil.ReadAll(f)
	if err != nil {
		tb.Fatal(err)
	}
	return v
}

func TestExampleChrome(t *testing.T) {
	buf := loadData(t, "ex1_chrome.stun")
	m := AcquireMessage()
	defer ReleaseMessage(m)
	_, err := m.ReadBytes(buf)
	if err != nil {
		t.Errorf("Failed to parse ex1_chrome: %s", err)
	}
}

func TestNearestLen(t *testing.T) {
	tt := []struct {
		in, out int
	}{
		{4, 4},   // 0
		{2, 4},   // 1
		{5, 8},   // 2
		{8, 8},   // 3
		{11, 12}, // 4
		{1, 4},   // 5
		{3, 4},   // 6
		{6, 8},   // 7
		{7, 8},   // 8
		{0, 0},   // 9
		{40, 40}, // 10
	}
	for i, c := range tt {
		if got := nearestLength(c.in); got != c.out {
			t.Errorf("[%d]: padd(%d) %d (got) != %d (expected)",
				i, c.in, got, c.out,
			)
		}
	}
}
