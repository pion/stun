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
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type attributeEncoder interface {
	AddTo(m *Message) error
}

func addAttr(t testing.TB, m *Message, a attributeEncoder) {
	if err := a.AddTo(m); err != nil {
		t.Error(err)
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
	if _, err := mDecoded.ReadFrom(bytes.NewReader(m.Raw)); err != nil {
		t.Error(err)
	}
	if !mDecoded.Equal(m) {
		t.Error(mDecoded, "!", m)
	}
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
		if b != tt.out {
			t.Errorf("Value(%s) -> %s, want %s", tt.in, bUint16(b), bUint16(tt.out))
		}
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
		if m != tt.out {
			t.Errorf("ReadValue(%s) -> %s, want %s", bUint16(tt.in), m, tt.out)
		}
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
		if m != tt {
			t.Errorf("ReadValue(%s -> %s) = %s, should be %s", tt, bUint16(v), m, tt)
			if m.Method != tt.Method {
				t.Errorf("%s != %s", bUint16(uint16(m.Method)), bUint16(uint16(tt.Method)))
			}
		}
	}
}

func TestMessage_WriteTo(t *testing.T) {
	m := New()
	m.Type = MessageType{Method: MethodBinding, Class: ClassRequest}
	m.TransactionID = NewTransactionID()
	m.Add(AttrErrorCode, []byte{0xff, 0xfe, 0xfa})
	m.WriteHeader()
	buf := new(bytes.Buffer)
	if _, err := m.WriteTo(buf); err != nil {
		t.Fatal(err)
	}
	mDecoded := New()
	if _, err := mDecoded.ReadFrom(buf); err != nil {
		t.Error(err)
	}
	if !mDecoded.Equal(m) {
		t.Error(mDecoded, "!", m)
	}
}

func TestMessage_Cookie(t *testing.T) {
	buf := make([]byte, 20)
	mDecoded := New()
	if _, err := mDecoded.ReadFrom(bytes.NewReader(buf)); err == nil {
		t.Error("should error")
	}
}

func TestMessage_LengthLessHeaderSize(t *testing.T) {
	buf := make([]byte, 8)
	mDecoded := New()
	if _, err := mDecoded.ReadFrom(bytes.NewReader(buf)); err == nil {
		t.Error("should error")
	}
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
	if _, err := mDecoded.Write(m.Raw); err == nil {
		t.Error("should error")
	}
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
	switch e := err.(type) {
	case *DecodeErr:
		if !e.IsPlace(DecodeErrPlace{"attribute", "header"}) {
			t.Error(e, "bad place")
		}
	default:
		t.Error(err, "should be bad format")
	}
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
	switch e := err.(type) {
	case *DecodeErr:
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
	mDecoded := New()
	_, err := mDecoded.ReadFrom(unexpectedEOFReader{})
	if !errors.Is(err, io.ErrUnexpectedEOF) {
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
		m.WriteTo(buf) // nolint:errcheck
		buf.Reset()
	}
}

func BenchmarkMessage_ReadFrom(b *testing.B) {
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
	reader := m.reader()
	mRec := New()
	for i := 0; i < b.N; i++ {
		if _, err := mRec.ReadFrom(reader); err != nil {
			b.Fatal(err)
		}
		reader.Reset(m.Raw)
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
		if k.String() == "" {
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
		if k.String() == "" {
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
	if MethodBinding.String() != "Binding" {
		t.Error("binding is not binding!")
	}
	if Method(0x616).String() != "0x616" {
		t.Error("Bad stringer", Method(0x616))
	}
}

func TestAttribute_Equal(t *testing.T) {
	a := RawAttribute{Length: 2, Value: []byte{0x1, 0x2}}
	b := RawAttribute{Length: 2, Value: []byte{0x1, 0x2}}
	if !a.Equal(b) {
		t.Error("should equal")
	}
	if a.Equal(RawAttribute{Type: 0x2}) {
		t.Error("should not equal")
	}
	if a.Equal(RawAttribute{Length: 0x2}) {
		t.Error("should not equal")
	}
	if a.Equal(RawAttribute{Length: 0x3}) {
		t.Error("should not equal")
	}
	if a.Equal(RawAttribute{Length: 2, Value: []byte{0x1, 0x3}}) {
		t.Error("should not equal")
	}
}

func TestMessage_Equal(t *testing.T) {
	attr := RawAttribute{Length: 2, Value: []byte{0x1, 0x2}, Type: 0x1}
	attrs := Attributes{attr}
	a := &Message{Attributes: attrs, Length: 4 + 2}
	b := &Message{Attributes: attrs, Length: 4 + 2}
	if !a.Equal(b) {
		t.Error("should equal")
	}
	if a.Equal(&Message{Type: MessageType{Class: 128}}) {
		t.Error("should not equal")
	}
	tID := [TransactionIDSize]byte{
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
	}
	if a.Equal(&Message{TransactionID: tID}) {
		t.Error("should not equal")
	}
	if a.Equal(&Message{Length: 3}) {
		t.Error("should not equal")
	}
	tAttrs := Attributes{
		{Length: 1, Value: []byte{0x1}, Type: 0x1},
	}
	if a.Equal(&Message{Attributes: tAttrs, Length: 4 + 2}) {
		t.Error("should not equal")
	}
	tAttrs = Attributes{
		{Length: 2, Value: []byte{0x1, 0x1}, Type: 0x2},
	}
	if a.Equal(&Message{Attributes: tAttrs, Length: 4 + 2}) {
		t.Error("should not equal")
	}
	if !(*Message)(nil).Equal(nil) {
		t.Error("nil should be equal to nil")
	}
	if a.Equal(nil) {
		t.Error("non-nil should not be equal to nil")
	}
	t.Run("Nil attributes", func(t *testing.T) {
		a := &Message{
			Attributes: nil,
			Length:     4 + 2,
		}
		b := &Message{
			Attributes: attrs,
			Length:     4 + 2,
		}
		if a.Equal(b) {
			t.Error("should not equal")
		}
		if b.Equal(a) {
			t.Error("should not equal")
		}
		b.Attributes = nil
		if !a.Equal(b) {
			t.Error("should equal")
		}
	})
	t.Run("Attributes length", func(t *testing.T) {
		attr := RawAttribute{Length: 2, Value: []byte{0x1, 0x2}, Type: 0x1}
		attr1 := RawAttribute{Length: 2, Value: []byte{0x1, 0x2}, Type: 0x1}
		a := &Message{Attributes: Attributes{attr}, Length: 4 + 2}
		b := &Message{Attributes: Attributes{attr, attr1}, Length: 4 + 2}
		if a.Equal(b) {
			t.Error("should not equal")
		}
	})
	t.Run("Attributes values", func(t *testing.T) {
		attr := RawAttribute{Length: 2, Value: []byte{0x1, 0x2}, Type: 0x1}
		attr1 := RawAttribute{Length: 2, Value: []byte{0x1, 0x1}, Type: 0x1}
		a := &Message{Attributes: Attributes{attr, attr}, Length: 4 + 2}
		b := &Message{Attributes: Attributes{attr, attr1}, Length: 4 + 2}
		if a.Equal(b) {
			t.Error("should not equal")
		}
	})
}

func TestMessageGrow(t *testing.T) {
	m := New()
	m.grow(512)
	if len(m.Raw) < 512 {
		t.Error("Bad length", len(m.Raw))
	}
}

func TestMessageGrowSmaller(t *testing.T) {
	m := New()
	m.grow(2)
	if cap(m.Raw) < 20 {
		t.Error("Bad capacity", cap(m.Raw))
	}
	if len(m.Raw) < 20 {
		t.Error("Bad length", len(m.Raw))
	}
}

func TestMessage_String(t *testing.T) {
	m := New()
	if m.String() == "" {
		t.Error("bad string")
	}
}

func TestIsMessage(t *testing.T) {
	m := New()
	NewSoftware("software").AddTo(m) // nolint:errcheck
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
		if got := IsMessage(v.in); got != v.out {
			t.Errorf("tt[%d]: IsMessage(%+v) %v != %v", i, v.in, got, v.out)
		}
	}
}

func BenchmarkIsMessage(b *testing.B) {
	m := New()
	m.Type = MessageType{Method: MethodBinding, Class: ClassRequest}
	m.TransactionID = NewTransactionID()
	NewSoftware("cydev/stun test").AddTo(m) // nolint:errcheck
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
	m := New()
	_, err := m.Write(buf)
	if err != nil {
		t.Errorf("Failed to parse ex1_chrome: %s", err)
	}
}

func TestMessageFromBrowsers(t *testing.T) {
	// file contains udp-packets captured from browsers (WebRTC)
	reader := csv.NewReader(bytes.NewReader(loadData(t, "frombrowsers.csv")))
	reader.Comma = ','
	_, err := reader.Read() // skipping header
	if err != nil {
		t.Fatal("failed to skip header of csv: ", err)
	}
	crcTable := crc64.MakeTable(crc64.ISO)
	m := New()
	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal("failed to read csv line: ", err)
		}
		data, err := base64.StdEncoding.DecodeString(line[1])
		if err != nil {
			t.Fatal("failed to decode ", line[1], " as base64: ", err)
		}
		b, err := strconv.ParseUint(line[2], 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		if b != crc64.Checksum(data, crcTable) {
			t.Error("crc64 check failed for ", line[1])
		}
		if _, err = m.Write(data); err != nil {
			t.Error("failed to decode ", line[1], " as message: ", err)
		}
		m.Reset()
	}
}

func BenchmarkMessage_NewTransactionID(b *testing.B) {
	b.ReportAllocs()
	m := new(Message)
	m.WriteHeader()
	for i := 0; i < b.N; i++ {
		if err := m.NewTransactionID(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMessageFull(b *testing.B) {
	b.ReportAllocs()
	m := new(Message)
	s := NewSoftware("software")
	addr := &XORMappedAddress{
		IP: net.IPv4(213, 1, 223, 5),
	}
	for i := 0; i < b.N; i++ {
		addAttr(b, m, addr)
		addAttr(b, m, &s)
		m.WriteAttributes()
		m.WriteHeader()
		Fingerprint.AddTo(m) // nolint:errcheck
		m.WriteHeader()
		m.Reset()
	}
}

func BenchmarkMessageFullHardcore(b *testing.B) {
	b.ReportAllocs()
	m := new(Message)
	s := NewSoftware("software")
	addr := &XORMappedAddress{
		IP: net.IPv4(213, 1, 223, 5),
	}
	for i := 0; i < b.N; i++ {
		if err := addr.AddTo(m); err != nil {
			b.Fatal(err)
		}
		if err := s.AddTo(m); err != nil {
			b.Fatal(err)
		}
		m.WriteHeader()
		m.Reset()
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
	if !m.Contains(AttrSoftware) {
		t.Error("message should contain software")
	}
	if m.Contains(AttrNonce) {
		t.Error("message should not contain nonce")
	}
}

func ExampleMessage() {
	buf := new(bytes.Buffer)
	m := new(Message)
	m.Build(BindingRequest, // nolint:errcheck
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
	fmt.Println(m, "buff length:", len(m.Raw))
	n, err := m.WriteTo(buf)
	fmt.Println("wrote", n, "err", err)

	// Decoding from buf new *Message.
	decoded := new(Message)
	decoded.Raw = make([]byte, 0, 1024) // for ReadFrom that reuses m.Raw
	// ReadFrom does not allocate internal buffer for reading from io.Reader,
	// instead it uses m.Raw, expanding it length to capacity.
	decoded.ReadFrom(buf) // nolint:errcheck
	fmt.Println("has software:", decoded.Contains(AttrSoftware))
	fmt.Println("has nonce:", decoded.Contains(AttrNonce))
	var software Software
	decoded.Parse(&software) // nolint:errcheck
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

	// Output:
	// Binding request l=48 attrs=3 id=AQIDBAUGBwgJAAEA buff length: 68
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
		TransactionID(),
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
			if err := s.AddTo(m); err != nil {
				t.Errorf("[%d] failed to add", i)
			}
		})
		if allocs > 0 {
			t.Errorf("[%d] allocated %.0f", i, allocs)
		}
	}
}

func TestAllocationsGetters(t *testing.T) {
	// Not testing AttrMessageIntegrity because it allocates.
	setters := []Setter{
		BindingRequest,
		TransactionID(),
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
	m := New()
	if err := m.Build(setters...); err != nil {
		t.Error("failed to build", err)
	}
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
			if err := g.GetFrom(m); err != nil {
				t.Errorf("[%d] failed to get", i)
			}
		})
		if allocs > 0 {
			t.Errorf("[%d] allocated %.0f", i, allocs)
		}
	}
}

func TestMessageFullSize(t *testing.T) {
	m := new(Message)
	if err := m.Build(BindingRequest,
		NewTransactionIDSetter([TransactionIDSize]byte{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1,
		}),
		NewSoftware("pion/stun"),
		NewLongTermIntegrity("username", "realm", "password"),
		Fingerprint,
	); err != nil {
		t.Fatal(err)
	}
	m.Raw = m.Raw[:len(m.Raw)-10]

	decoder := new(Message)
	decoder.Raw = m.Raw[:len(m.Raw)-10]
	if err := decoder.Decode(); err == nil {
		t.Error("decode on truncated buffer should error")
	}
}

func TestMessage_CloneTo(t *testing.T) {
	m := new(Message)
	if err := m.Build(BindingRequest,
		NewTransactionIDSetter([TransactionIDSize]byte{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1,
		}),
		NewSoftware("pion/stun"),
		NewLongTermIntegrity("username", "realm", "password"),
		Fingerprint,
	); err != nil {
		t.Fatal(err)
	}
	m.Encode()
	b := new(Message)
	if err := m.CloneTo(b); err != nil {
		t.Fatal(err)
	}
	if !b.Equal(m) {
		t.Fatal("not equal")
	}
	// Corrupting m and checking that b is not corrupted.
	s, ok := b.Attributes.Get(AttrSoftware)
	if !ok {
		t.Fatal("no software attribute")
	}
	s.Value[0] = 'k'
	if b.Equal(m) {
		t.Fatal("should not be equal")
	}
}

func BenchmarkMessage_CloneTo(b *testing.B) {
	b.ReportAllocs()
	m := new(Message)
	if err := m.Build(BindingRequest,
		NewTransactionIDSetter([TransactionIDSize]byte{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1,
		}),
		NewSoftware("pion/stun"),
		NewLongTermIntegrity("username", "realm", "password"),
		Fingerprint,
	); err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(m.Raw)))
	a := new(Message)
	m.CloneTo(a) // nolint:errcheck
	for i := 0; i < b.N; i++ {
		if err := m.CloneTo(a); err != nil {
			b.Fatal(err)
		}
	}
}

func TestMessage_AddTo(t *testing.T) {
	m := new(Message)
	if err := m.Build(BindingRequest,
		NewTransactionIDSetter([TransactionIDSize]byte{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1,
		}),
		Fingerprint,
	); err != nil {
		t.Fatal(err)
	}
	m.Encode()
	b := new(Message)
	if err := m.CloneTo(b); err != nil {
		t.Fatal(err)
	}
	m.TransactionID = [TransactionIDSize]byte{
		1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 2,
	}
	if b.Equal(m) {
		t.Fatal("should not be equal")
	}
	m.AddTo(b) // nolint:errcheck
	if !b.Equal(m) {
		t.Fatal("should be equal")
	}
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
	m.CloneTo(a) // nolint:errcheck
	for i := 0; i < b.N; i++ {
		if err := m.AddTo(a); err != nil {
			b.Fatal(err)
		}
	}
}

func TestDecode(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		if err := Decode(nil, nil); !errors.Is(err, ErrDecodeToNil) {
			t.Errorf("unexpected error: %v", err)
		}
	})
	m := New()
	m.Type = MessageType{Method: MethodBinding, Class: ClassRequest}
	m.TransactionID = NewTransactionID()
	m.Add(AttrErrorCode, []byte{0xff, 0xfe, 0xfa})
	m.WriteHeader()
	mDecoded := New()
	if err := Decode(m.Raw, mDecoded); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !mDecoded.Equal(m) {
		t.Error("decoded result is not equal to encoded message")
	}
	t.Run("ZeroAlloc", func(t *testing.T) {
		allocs := testing.AllocsPerRun(10, func() {
			mDecoded.Reset()
			if err := Decode(m.Raw, mDecoded); err != nil {
				t.Error(err)
			}
		})
		if allocs > 0 {
			t.Error("unexpected allocations")
		}
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
