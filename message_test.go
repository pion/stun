// SPDX-FileCopyrightText: 2026 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

//go:build !js
// +build !js

package stun

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/csv"
	"errors"
	"fmt"
	"hash/crc64"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type attributeEncoder interface {
	AddTo(m *Message) error
}

func addAttr(tb testing.TB, m *Message, a attributeEncoder) {
	tb.Helper()

	if err := a.AddTo(m); err != nil {
		tb.Error(err)
	}
}

func bUint16(v uint16) string {
	return fmt.Sprintf("0b%016s", strconv.FormatUint(uint64(v), 2))
}

func (m *Message) reader() *bytes.Reader {
	return bytes.NewReader(m.Raw)
}

func TestMessageBuffer(t *testing.T) {
	m := New()
	m.Type = MessageType{Method: MethodBinding, Class: ClassRequest}
	m.TransactionID = NewTransactionID()
	m.Add(AttrErrorCode, []byte{0xff, 0xfe, 0xfa})
	m.WriteHeader()
	mDecoded := New()
	_, err := mDecoded.ReadFrom(bytes.NewReader(m.Raw))
	assert.NoError(t, err)
	assert.True(t, mDecoded.Equal(m), "mDecoded != m")
}

func BenchmarkMessage_Write(b *testing.B) {
	b.ReportAllocs()
	attributeValue := []byte{0xff, 0x11, 0x12, 0x34}
	b.SetBytes(int64(len(attributeValue) + messageHeaderSize +
		attributeHeaderSize))
	transactionID := NewTransactionID()
	m := New()
	for i := 0; i < b.N; i++ {
		m.Add(AttrErrorCode, attributeValue)
		m.TransactionID = transactionID
		m.Type = MessageType{Method: MethodBinding, Class: ClassRequest}
		m.WriteHeader()
		m.Reset()
	}
}

func TestMessageType_Value(t *testing.T) {
	tests := []struct {
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
		assert.Equal(t, tt.out, b, "Value(%s) -> %s, want %s", tt.in, bUint16(b), bUint16(tt.out))
	}
}

func TestMessageType_ReadValue(t *testing.T) {
	tests := []struct {
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
		assert.Equal(t, tt.out, m, "ReadValue(%s) -> %s, want %s", bUint16(tt.in), m, tt.out)
	}
}

func TestMessageType_ReadWriteValue(t *testing.T) {
	tests := []MessageType{
		{Method: MethodBinding, Class: ClassRequest},
		{Method: MethodBinding, Class: ClassSuccessResponse},
		{Method: MethodBinding, Class: ClassErrorResponse},
		{Method: 0x12, Class: ClassErrorResponse},
	}
	for _, tt := range tests {
		m := MessageType{}
		v := tt.Value()
		m.ReadValue(v)
		assert.Equal(t, tt, m, "ReadValue(%s -> %s) = %s, should be %s", tt, bUint16(v), m, tt)
		assert.Equal(t, tt.Method, m.Method, "%s != %s", bUint16(uint16(m.Method)), bUint16(uint16(tt.Method)))
	}
}

func TestMessage_WriteTo(t *testing.T) {
	msg := New()
	msg.Type = MessageType{Method: MethodBinding, Class: ClassRequest}
	msg.TransactionID = NewTransactionID()
	msg.Add(AttrErrorCode, []byte{0xff, 0xfe, 0xfa})
	msg.WriteHeader()
	buf := new(bytes.Buffer)
	_, err := msg.WriteTo(buf)
	assert.NoError(t, err)
	mDecoded := New()
	_, err = mDecoded.ReadFrom(buf)
	assert.NoError(t, err)
	assert.True(t, mDecoded.Equal(msg), "mDecoded != msg")
}

func TestMessage_Cookie(t *testing.T) {
	buf := make([]byte, 20)
	mDecoded := New()
	_, err := mDecoded.ReadFrom(bytes.NewReader(buf))
	assert.Error(t, err, "should error")
}

func TestMessage_LengthLessHeaderSize(t *testing.T) {
	buf := make([]byte, 8)
	mDecoded := New()
	_, err := mDecoded.ReadFrom(bytes.NewReader(buf))
	assert.Error(t, err, "should error")
}

func TestMessage_BadLength(t *testing.T) {
	mType := MessageType{Method: MethodBinding, Class: ClassRequest}
	m := &Message{
		Type:          mType,
		Length:        4,
		TransactionID: [TransactionIDSize]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
	}
	m.Add(0x1, []byte{1, 2})
	m.WriteHeader()
	m.Raw[20+3] = 10 // set attr length = 10
	mDecoded := New()
	_, err := mDecoded.Write(m.Raw)
	assert.Error(t, err, "should error")
}

func TestMessage_AttrLengthLessThanHeader(t *testing.T) {
	mType := MessageType{Method: MethodBinding, Class: ClassRequest}
	messageAttribute := RawAttribute{Length: 2, Value: []byte{1, 2}, Type: 0x1}
	messageAttributes := Attributes{
		messageAttribute,
	}
	m := &Message{
		Type:          mType,
		TransactionID: NewTransactionID(),
		Attributes:    messageAttributes,
	}
	m.Encode()
	mDecoded := New()
	binary.BigEndian.PutUint16(m.Raw[2:4], 2) // rewrite to bad length
	_, err := mDecoded.ReadFrom(bytes.NewReader(m.Raw[:20+2]))
	var e *DecodeErr
	assert.ErrorAs(t, err, &e)
	assert.True(t, e.IsPlace(DecodeErrPlace{"attribute", "header"}), "bad place")
}

func TestMessage_AttrSizeLessThanLength(t *testing.T) {
	mType := MessageType{Method: MethodBinding, Class: ClassRequest}
	messageAttribute := RawAttribute{
		Length: 4,
		Value:  []byte{1, 2, 3, 4}, Type: 0x1,
	}
	messageAttributes := Attributes{
		messageAttribute,
	}
	m := &Message{
		Type:          mType,
		TransactionID: NewTransactionID(),
		Attributes:    messageAttributes,
	}
	m.WriteAttributes()
	m.WriteHeader()
	bin.PutUint16(m.Raw[2:4], 5) // rewrite to bad length
	mDecoded := New()
	_, err := mDecoded.ReadFrom(bytes.NewReader(m.Raw[:20+5]))
	var e *DecodeErr
	assert.ErrorAs(t, err, &e)
	assert.True(t, e.IsPlace(DecodeErrPlace{"attribute", "value"}), "bad place")
}

type unexpectedEOFReader struct{}

func (r unexpectedEOFReader) Read([]byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func TestMessage_ReadFromError(t *testing.T) {
	mDecoded := New()
	_, err := mDecoded.ReadFrom(unexpectedEOFReader{})
	assert.ErrorIs(t, err, io.ErrUnexpectedEOF, "should be", io.ErrUnexpectedEOF)
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
	m := &Message{
		Type:   mType,
		Length: 0,
		TransactionID: [TransactionIDSize]byte{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
		},
	}
	m.WriteHeader()
	buf := new(bytes.Buffer)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m.WriteTo(buf) //nolint:errcheck,gosec
		buf.Reset()
	}
}

func BenchmarkMessage_ReadFrom(b *testing.B) {
	mType := MessageType{Method: MethodBinding, Class: ClassRequest}
	msg := &Message{
		Type:   mType,
		Length: 0,
		TransactionID: [TransactionIDSize]byte{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
		},
	}
	msg.WriteHeader()
	b.ReportAllocs()
	b.SetBytes(int64(len(msg.Raw)))
	reader := msg.reader()
	mRec := New()
	for i := 0; i < b.N; i++ {
		if _, err := mRec.ReadFrom(reader); err != nil {
			b.Fatal(err)
		}
		reader.Reset(msg.Raw)
		mRec.Reset()
	}
}

func BenchmarkMessage_ReadBytes(b *testing.B) {
	mType := MessageType{Method: MethodBinding, Class: ClassRequest}
	m := &Message{
		Type:   mType,
		Length: 0,
		TransactionID: [TransactionIDSize]byte{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
		},
	}
	m.WriteHeader()
	b.ReportAllocs()
	b.SetBytes(int64(len(m.Raw)))
	mRec := New()
	for i := 0; i < b.N; i++ {
		if _, err := mRec.Write(m.Raw); err != nil {
			b.Fatal(err)
		}
		mRec.Reset()
	}
}

func TestMessageClass_String(t *testing.T) {
	defer func() {
		assert.NotNil(t, recover())
	}()

	v := [...]MessageClass{
		ClassRequest,
		ClassErrorResponse,
		ClassSuccessResponse,
		ClassIndication,
	}
	for _, k := range v {
		assert.NotEmpty(t, k.String(), "%v bad stringer", k)
	}

	// should panic
	p := MessageClass(0x05).String()
	assert.Fail(t, "should panic", p)
}

func TestAttrType_String(t *testing.T) {
	attrType := [...]AttrType{
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
	for _, k := range attrType {
		assert.NotEmpty(t, k.String(), "%v bad stringer", k)
		assert.False(t, strings.HasPrefix(k.String(), "0x"), "%v bad stringer", k)
	}
	vNonStandard := AttrType(0x512)
	assert.True(t, strings.HasPrefix(vNonStandard.String(), "0x512"), "%v bad prefix", vNonStandard)
}

func TestMethod_String(t *testing.T) {
	assert.Equal(t, "Binding", MethodBinding.String(), "binding is not binding!")
	assert.Equal(t, "0x616", Method(0x616).String(), "Bad stringer")
}

func TestAttribute_Equal(t *testing.T) {
	attr1 := RawAttribute{Length: 2, Value: []byte{0x1, 0x2}, Type: 0x1}
	attr2 := RawAttribute{Length: 2, Value: []byte{0x1, 0x2}, Type: 0x1}
	assert.True(t, attr1.Equal(attr2))
	assert.False(t, attr1.Equal(RawAttribute{Type: 0x2}))
	assert.False(t, attr1.Equal(RawAttribute{Length: 0x2}))
	assert.False(t, attr1.Equal(RawAttribute{Length: 0x3}))
	assert.False(t, attr1.Equal(RawAttribute{Length: 2, Value: []byte{0x1, 0x3}}))
}

func TestMessage_Equal(t *testing.T) { //nolint:cyclop
	attr := RawAttribute{Length: 2, Value: []byte{0x1, 0x2}, Type: 0x1}
	attrs := Attributes{attr}
	msg1 := &Message{Attributes: attrs, Length: 4 + 2}
	msg2 := &Message{Attributes: attrs, Length: 4 + 2}
	assert.True(t, msg1.Equal(msg2))
	assert.False(t, msg1.Equal(&Message{Type: MessageType{Class: 128}}))
	tID := [TransactionIDSize]byte{
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
	}
	assert.False(t, msg1.Equal(&Message{TransactionID: tID}))
	assert.False(t, msg1.Equal(&Message{Length: 3}))
	tAttrs := Attributes{
		{Length: 1, Value: []byte{0x1}, Type: 0x1},
	}
	assert.False(t, msg1.Equal(&Message{Attributes: tAttrs, Length: 4 + 2}))
	tAttrs = Attributes{
		{Length: 2, Value: []byte{0x1, 0x1}, Type: 0x2},
	}
	assert.False(t, msg1.Equal(&Message{Attributes: tAttrs, Length: 4 + 2}))
	assert.True(t, (*Message)(nil).Equal(nil), "nil should be equal to nil")
	assert.False(t, msg1.Equal(nil), "non-nil should not be equal to nil")
	t.Run("Nil attributes", func(t *testing.T) {
		msg1 := &Message{
			Attributes: nil,
			Length:     4 + 2,
		}
		msg2 := &Message{
			Attributes: attrs,
			Length:     4 + 2,
		}
		assert.False(t, msg1.Equal(msg2))
		assert.False(t, msg2.Equal(msg1))
		msg2.Attributes = nil
		assert.True(t, msg1.Equal(msg2))
	})
	t.Run("Attributes length", func(t *testing.T) {
		attr := RawAttribute{Length: 2, Value: []byte{0x1, 0x2}, Type: 0x1}
		attr1 := RawAttribute{Length: 2, Value: []byte{0x1, 0x2}, Type: 0x1}
		a := &Message{Attributes: Attributes{attr}, Length: 4 + 2}
		b := &Message{Attributes: Attributes{attr, attr1}, Length: 4 + 2}
		assert.False(t, a.Equal(b))
	})
	t.Run("Attributes values", func(t *testing.T) {
		attr := RawAttribute{Length: 2, Value: []byte{0x1, 0x2}, Type: 0x1}
		attr1 := RawAttribute{Length: 2, Value: []byte{0x1, 0x1}, Type: 0x1}
		a := &Message{Attributes: Attributes{attr, attr}, Length: 4 + 2}
		b := &Message{Attributes: Attributes{attr, attr1}, Length: 4 + 2}
		assert.False(t, a.Equal(b))
	})
}

func TestMessageGrow(t *testing.T) {
	m := New()
	m.grow(512)
	assert.GreaterOrEqual(t, len(m.Raw), 512)
}

func TestMessageGrowSmaller(t *testing.T) {
	m := New()
	m.grow(2)
	assert.GreaterOrEqual(t, cap(m.Raw), 20)
	assert.GreaterOrEqual(t, len(m.Raw), 20)
}

func TestMessage_String(t *testing.T) {
	m := New()
	assert.NotEmpty(t, m.String())
}

func TestIsMessage(t *testing.T) {
	m := New()
	NewSoftware("software").AddTo(m) //nolint:errcheck,gosec
	m.WriteHeader()

	tt := [...]struct {
		in  []byte
		out bool
	}{
		{nil, false},                                // 0
		{[]byte{1, 2, 3}, false},                    // 1
		{[]byte{1, 2, 4}, false},                    // 2
		{[]byte{1, 2, 4, 5, 6, 7, 8, 9, 20}, false}, // 3
		{m.Raw, true},                               // 5
		{[]byte{
			0, 0, 0, 0, 33, 18,
			164, 66, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0,
		}, true}, // 6
	}
	for i, v := range tt {
		assert.Equal(t, v.out, IsMessage(v.in), "tt[%d]: IsMessage(%+v)", i, v.in)
	}
}

func BenchmarkIsMessage(b *testing.B) {
	m := New()
	m.Type = MessageType{Method: MethodBinding, Class: ClassRequest}
	m.TransactionID = NewTransactionID()
	NewSoftware("cydev/stun test").AddTo(m) //nolint:errcheck,gosec
	m.WriteHeader()

	b.SetBytes(int64(messageHeaderSize))
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if !IsMessage(m.Raw) {
			b.Fatal("Should be message")
		}
	}
}

func loadData(tb testing.TB, name string) []byte {
	tb.Helper()

	name = filepath.Join("testdata", name)
	f, err := os.Open(name) //nolint:gosec
	assert.NoError(tb, err)
	defer func() {
		assert.NoError(tb, f.Close())
	}()
	v, err := io.ReadAll(f)
	assert.NoError(tb, err)

	return v
}

func TestExampleChrome(t *testing.T) {
	buf := loadData(t, "ex1_chrome.stun")
	m := New()
	_, err := m.Write(buf)
	assert.NoError(t, err, "Failed to parse ex1_chrome")
}

func TestMessageFromBrowsers(t *testing.T) {
	// file contains udp-packets captured from browsers (WebRTC)
	reader := csv.NewReader(bytes.NewReader(loadData(t, "frombrowsers.csv")))
	reader.Comment = '#'
	_, err := reader.Read() // skipping header
	assert.NoError(t, err, "failed to skip header of csv")
	crcTable := crc64.MakeTable(crc64.ISO)
	msg := New()
	for {
		line, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		assert.NoError(t, err, "failed to read csv line")
		data, err := base64.StdEncoding.DecodeString(line[1])
		assert.NoError(t, err)
		b, err := strconv.ParseUint(line[2], 10, 64)
		assert.NoError(t, err)
		assert.Equal(t, b, crc64.Checksum(data, crcTable), "crc64 check failed for %s", line[1])
		_, err = msg.Write(data)
		assert.NoError(t, err, "failed to decode %s as message: %s", line[1], err)
		msg.Reset()
	}
}

func BenchmarkMessage_NewTransactionID(b *testing.B) {
	b.ReportAllocs()
	m := new(Message)
	m.WriteHeader()
	for i := 0; i < b.N; i++ {
		assert.NoError(b, m.NewTransactionID())
	}
}

func BenchmarkMessageFull(b *testing.B) {
	b.ReportAllocs()
	msg := new(Message)
	s := NewSoftware("software")
	addr := &XORMappedAddress{
		IP: net.IPv4(213, 1, 223, 5),
	}
	for i := 0; i < b.N; i++ {
		assert.NoError(b, addr.AddTo(msg))
		assert.NoError(b, s.AddTo(msg))
		msg.WriteAttributes()
		msg.WriteHeader()
		Fingerprint.AddTo(msg) //nolint:errcheck,gosec
		msg.WriteHeader()
		msg.Reset()
	}
}

func BenchmarkMessageFullHardcore(b *testing.B) {
	b.ReportAllocs()
	msg := new(Message)
	s := NewSoftware("software")
	addr := &XORMappedAddress{
		IP: net.IPv4(213, 1, 223, 5),
	}
	for i := 0; i < b.N; i++ {
		assert.NoError(b, addr.AddTo(msg))
		assert.NoError(b, s.AddTo(msg))
		msg.WriteHeader()
		msg.Reset()
	}
}

func BenchmarkMessage_WriteHeader(b *testing.B) {
	m := &Message{
		TransactionID: NewTransactionID(),
		Raw:           make([]byte, 120),
		Type: MessageType{
			Class:  ClassRequest,
			Method: MethodBinding,
		},
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m.WriteHeader()
	}
}

func TestMessage_Contains(t *testing.T) {
	m := new(Message)
	m.Add(AttrSoftware, []byte("value"))
	assert.True(t, m.Contains(AttrSoftware), "message should contain software")
	assert.False(t, m.Contains(AttrNonce), "message should not contain nonce")
}

func ExampleMessage() {
	buf := new(bytes.Buffer)
	msg := new(Message)
	msg.Build(BindingRequest, //nolint:errcheck,gosec
		NewTransactionIDSetter([TransactionIDSize]byte{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1,
		}),
		NewSoftware("ernado/stun"),
		NewLongTermIntegrity("username", "realm", "password"),
		Fingerprint,
	)
	// Instead of calling Build, use AddTo(m) directly for all setters
	// to reduce allocations.
	// For example:
	//	software := NewSoftware("ernado/stun")
	//	software.AddTo(m)  // no allocations
	// Or pass software as follows:
	//	m.Build(&software) // no allocations
	// If you pass software as value, there will be 1 allocation.
	// This rule is correct for all setters.
	fmt.Println(msg, "buff length:", len(msg.Raw))
	n, err := msg.WriteTo(buf)
	fmt.Println("wrote", n, "err", err)

	// Decoding from buf new *Message.
	decoded := new(Message)
	decoded.Raw = make([]byte, 0, 1024) // for ReadFrom that reuses m.Raw
	// ReadFrom does not allocate internal buffer for reading from io.Reader,
	// instead it uses m.Raw, expanding it length to capacity.
	decoded.ReadFrom(buf) //nolint:errcheck,gosec
	fmt.Println("has software:", decoded.Contains(AttrSoftware))
	fmt.Println("has nonce:", decoded.Contains(AttrNonce))
	var software Software
	decoded.Parse(&software) //nolint:errcheck,gosec
	// Rule for Parse method is same as for Build.
	fmt.Println("software:", software)
	if err := Fingerprint.Check(decoded); err == nil {
		fmt.Println("fingerprint is correct")
	} else {
		fmt.Println("fingerprint is incorrect:", err)
	}
	// Checking integrity
	i := NewLongTermIntegrity("username", "realm", "password")
	if err := i.Check(decoded); err == nil {
		fmt.Println("integrity ok")
	} else {
		fmt.Println("integrity bad:", err)
	}
	fmt.Println("for corrupted message:")
	decoded.Raw[22] = 33
	if Fingerprint.Check(decoded) == nil {
		fmt.Println("fingerprint: ok")
	} else {
		fmt.Println("fingerprint: failed")
	}

	//nolint:lll
	// Output:
	// Binding request l=48 attrs=3 id=AQIDBAUGBwgJAAEA, attr0=SOFTWARE attr1=MESSAGE-INTEGRITY attr2=FINGERPRINT  buff length: 68
	// wrote 68 err <nil>
	// has software: true
	// has nonce: false
	// software: ernado/stun
	// fingerprint is correct
	// integrity ok
	// for corrupted message:
	// fingerprint: failed
}

func TestAllocations(t *testing.T) {
	// Not testing AttrMessageIntegrity because it allocates.
	setters := []Setter{
		BindingRequest,
		TransactionID,
		Fingerprint,
		NewNonce("nonce"),
		NewUsername("username"),
		XORMappedAddress{
			IP:   net.IPv4(11, 22, 33, 44),
			Port: 334,
		},
		UnknownAttributes{AttrLifetime, AttrChannelNumber},
		CodeInsufficientCapacity,
		ErrorCodeAttribute{
			Code:   200,
			Reason: []byte("hello"),
		},
	}
	m := New()
	for i, s := range setters {
		s := s
		i := i
		allocs := testing.AllocsPerRun(10, func() {
			m.Reset()
			m.WriteHeader()
			assert.NoError(t, s.AddTo(m), "[%d] failed to add", i)
		})
		assert.Zero(t, allocs, "[%d] allocated", i)
	}
}

func TestAllocationsGetters(t *testing.T) {
	// Not testing AttrMessageIntegrity because it allocates.
	setters := []Setter{
		BindingRequest,
		TransactionID,
		NewNonce("nonce"),
		NewUsername("username"),
		XORMappedAddress{
			IP:   net.IPv4(11, 22, 33, 44),
			Port: 334,
		},
		UnknownAttributes{AttrLifetime, AttrChannelNumber},
		CodeInsufficientCapacity,
		ErrorCodeAttribute{
			Code:   200,
			Reason: []byte("hello"),
		},
		NewShortTermIntegrity("pwd"),
		Fingerprint,
	}
	msg := New()
	assert.NoError(t, msg.Build(setters...))
	getters := []Getter{
		new(Nonce),
		new(Username),
		new(XORMappedAddress),
		new(UnknownAttributes),
		new(ErrorCodeAttribute),
	}
	for i, g := range getters {
		g := g
		i := i
		allocs := testing.AllocsPerRun(10, func() {
			assert.NoError(t, g.GetFrom(msg), "[%d] failed to get", i)
		})
		assert.Zero(t, allocs, "[%d] allocated", i)
	}
}

func TestMessageFullSize(t *testing.T) {
	msg := new(Message)
	assert.NoError(t, msg.Build(BindingRequest,
		NewTransactionIDSetter([TransactionIDSize]byte{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1,
		}),
		NewSoftware("pion/stun"),
		NewLongTermIntegrity("username", "realm", "password"),
		Fingerprint,
	))
	msg.Raw = msg.Raw[:len(msg.Raw)-10]

	decoder := new(Message)
	decoder.Raw = msg.Raw[:len(msg.Raw)-10]
	assert.Error(t, decoder.Decode(), "decode on truncated buffer should error")
}

func TestMessage_CloneTo(t *testing.T) {
	msg := new(Message)
	assert.NoError(t, msg.Build(BindingRequest,
		NewTransactionIDSetter([TransactionIDSize]byte{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1,
		}),
		NewSoftware("pion/stun"),
		NewLongTermIntegrity("username", "realm", "password"),
		Fingerprint,
	))
	msg.Encode()
	msg2 := new(Message)
	assert.NoError(t, msg.CloneTo(msg2))
	assert.True(t, msg2.Equal(msg), "cloned message should equal original")
	// Corrupting m and checking that b is not corrupted.
	s, ok := msg2.Attributes.Get(AttrSoftware)
	assert.True(t, ok)
	s.Value[0] = 'k'
	assert.False(t, msg2.Equal(msg), "should not be equal")
}

func BenchmarkMessage_CloneTo(b *testing.B) {
	b.ReportAllocs()
	msg := new(Message)
	if err := msg.Build(BindingRequest,
		NewTransactionIDSetter([TransactionIDSize]byte{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1,
		}),
		NewSoftware("pion/stun"),
		NewLongTermIntegrity("username", "realm", "password"),
		Fingerprint,
	); err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(msg.Raw)))
	a := new(Message)
	msg.CloneTo(a) //nolint:errcheck,gosec
	for i := 0; i < b.N; i++ {
		if err := msg.CloneTo(a); err != nil {
			b.Fatal(err)
		}
	}
}

func TestMessage_AddTo(t *testing.T) {
	msg := new(Message)
	assert.NoError(t, msg.Build(BindingRequest,
		NewTransactionIDSetter([TransactionIDSize]byte{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1,
		}),
		Fingerprint,
	))
	msg.Encode()
	b := new(Message)
	assert.NoError(t, msg.CloneTo(b))
	msg.TransactionID = [TransactionIDSize]byte{
		1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 2,
	}
	assert.False(t, b.Equal(msg), "should not be equal")
	msg.AddTo(b) //nolint:errcheck,gosec
	assert.True(t, b.Equal(msg), "should be equal")
}

func BenchmarkMessage_AddTo(b *testing.B) {
	b.ReportAllocs()
	m := new(Message)
	if err := m.Build(BindingRequest,
		NewTransactionIDSetter([TransactionIDSize]byte{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1,
		}),
		Fingerprint,
	); err != nil {
		b.Fatal(err)
	}
	a := new(Message)
	m.CloneTo(a) //nolint:errcheck,gosec
	for i := 0; i < b.N; i++ {
		if err := m.AddTo(a); err != nil {
			b.Fatal(err)
		}
	}
}

func TestDecode(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		assert.ErrorIs(t, Decode(nil, nil), ErrDecodeToNil)
	})
	msg := New()
	msg.Type = MessageType{Method: MethodBinding, Class: ClassRequest}
	msg.TransactionID = NewTransactionID()
	msg.Add(AttrErrorCode, []byte{0xff, 0xfe, 0xfa})
	msg.WriteHeader()
	mDecoded := New()
	assert.NoError(t, Decode(msg.Raw, mDecoded))
	assert.True(t, mDecoded.Equal(msg), "decoded result is not equal to encoded message")
	t.Run("ZeroAlloc", func(t *testing.T) {
		allocs := testing.AllocsPerRun(10, func() {
			mDecoded.Reset()
			assert.NoError(t, Decode(msg.Raw, mDecoded))
		})
		assert.Zero(t, allocs, "unexpected allocations")
	})
}

func BenchmarkDecode(b *testing.B) {
	m := New()
	m.Type = MessageType{Method: MethodBinding, Class: ClassRequest}
	m.TransactionID = NewTransactionID()
	m.Add(AttrErrorCode, []byte{0xff, 0xfe, 0xfa})
	m.WriteHeader()
	mDecoded := New()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mDecoded.Reset()
		if err := Decode(m.Raw, mDecoded); err != nil {
			b.Fatal(err)
		}
	}
}

func TestMessage_MarshalBinary(t *testing.T) {
	msg := MustBuild(
		NewSoftware("software"),
		&XORMappedAddress{
			IP: net.IPv4(213, 1, 223, 5),
		},
	)
	data, err := msg.MarshalBinary()
	assert.NoError(t, err)

	// Reset m.Raw to check retention.
	for i := range msg.Raw {
		msg.Raw[i] = 0
	}
	assert.NoError(t, msg.UnmarshalBinary(data))

	// Reset data to check retention.
	for i := range data {
		data[i] = 0
	}
	assert.NoError(t, msg.Decode())
}

func TestMessage_GobDecode(t *testing.T) {
	msg := MustBuild(
		NewSoftware("software"),
		&XORMappedAddress{
			IP: net.IPv4(213, 1, 223, 5),
		},
	)
	data, err := msg.GobEncode()
	assert.NoError(t, err)

	// Reset m.Raw to check retention.
	for i := range msg.Raw {
		msg.Raw[i] = 0
	}
	assert.NoError(t, msg.GobDecode(data))

	// Reset data to check retention.
	for i := range data {
		data[i] = 0
	}
	assert.NoError(t, msg.Decode())
}
