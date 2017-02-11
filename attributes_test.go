package stun

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"net"
	"strings"
	"testing"
)

func TestSoftware_GetFrom(t *testing.T) {
	m := New()
	v := "Client v0.0.1"
	m.Add(AttrSoftware, []byte(v))
	m.WriteHeader()

	m2 := &Message{
		Raw: make([]byte, 0, 256),
	}
	software := new(Software)
	if _, err := m2.ReadFrom(m.reader()); err != nil {
		t.Error(err)
	}
	if err := software.GetFrom(m); err != nil {
		t.Fatal(err)
	}
	if software.String() != v {
		t.Errorf("Expected %q, got %q.", v, software)
	}

	sAttr, ok := m.Attributes.Get(AttrSoftware)
	if !ok {
		t.Error("sowfware attribute should be found")
	}
	s := sAttr.String()
	if !strings.HasPrefix(s, "SOFTWARE:") {
		t.Error("bad string representation", s)
	}
}

func BenchmarkXORMappedAddress_AddTo(b *testing.B) {
	m := New()
	b.ReportAllocs()
	ip := net.ParseIP("192.168.1.32")
	for i := 0; i < b.N; i++ {
		addr := &XORMappedAddress{IP: ip, Port: 3654}
		addr.AddTo(m)
		m.Reset()
	}
}

func BenchmarkXORMappedAddress_GetFrom(b *testing.B) {
	m := New()
	transactionID, err := base64.StdEncoding.DecodeString("jxhBARZwX+rsC6er")
	if err != nil {
		b.Error(err)
	}
	copy(m.TransactionID[:], transactionID)
	addrValue, err := hex.DecodeString("00019cd5f49f38ae")
	if err != nil {
		b.Error(err)
	}
	m.Add(AttrXORMappedAddress, addrValue)
	addr := new(XORMappedAddress)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := addr.GetFrom(m); err != nil {
			b.Fatal(err)
		}
	}
}

func TestMessage_GetXORMappedAddress(t *testing.T) {
	m := New()
	transactionID, err := base64.StdEncoding.DecodeString("jxhBARZwX+rsC6er")
	if err != nil {
		t.Error(err)
	}
	copy(m.TransactionID[:], transactionID)
	addrValue, err := hex.DecodeString("00019cd5f49f38ae")
	if err != nil {
		t.Error(err)
	}
	m.Add(AttrXORMappedAddress, addrValue)
	addr := new(XORMappedAddress)
	if err = addr.GetFrom(m); err != nil {
		t.Error(err)
	}
	if !addr.IP.Equal(net.ParseIP("213.141.156.236")) {
		t.Error("bad IP", addr.IP, "!=", "213.141.156.236")
	}
	if addr.Port != 48583 {
		t.Error("bad Port", addr.Port, "!=", 48583)
	}
}

func TestMessage_GetXORMappedAddressBad(t *testing.T) {
	m := New()
	transactionID, err := base64.StdEncoding.DecodeString("jxhBARZwX+rsC6er")
	if err != nil {
		t.Error(err)
	}
	copy(m.TransactionID[:], transactionID)
	expectedIP := net.ParseIP("213.141.156.236")
	expectedPort := 21254
	addr := new(XORMappedAddress)

	if err = addr.GetFrom(m); err == nil {
		t.Fatal(err, "should be nil")
	}

	addr.IP = expectedIP
	addr.Port = expectedPort
	addr.AddTo(m)
	m.WriteHeader()

	mRes := New()
	binary.BigEndian.PutUint16(m.Raw[20+4:20+4+2], 0x21)
	if _, err = mRes.ReadFrom(bytes.NewReader(m.Raw)); err != nil {
		t.Fatal(err)
	}
	if err = addr.GetFrom(m); err == nil {
		t.Fatal(err, "should not be nil")
	}
}

func TestMessage_AddXORMappedAddress(t *testing.T) {
	m := New()
	transactionID, err := base64.StdEncoding.DecodeString("jxhBARZwX+rsC6er")
	if err != nil {
		t.Error(err)
	}
	copy(m.TransactionID[:], transactionID)
	expectedIP := net.ParseIP("213.141.156.236")
	expectedPort := 21254
	addr := &XORMappedAddress{
		IP:   net.ParseIP("213.141.156.236"),
		Port: expectedPort,
	}
	if err = addr.AddTo(m); err != nil {
		t.Fatal(err)
	}
	m.WriteHeader()
	mRes := New()
	if _, err = mRes.Write(m.Raw); err != nil {
		t.Fatal(err)
	}
	if err = addr.GetFrom(mRes); err != nil {
		t.Fatal(err)
	}
	if !addr.IP.Equal(expectedIP) {
		t.Errorf("%s (got) != %s (expected)", addr.IP, expectedIP)
	}
	if addr.Port != expectedPort {
		t.Error("bad Port", addr.Port, "!=", expectedPort)
	}
}

func TestMessage_AddXORMappedAddressV6(t *testing.T) {
	m := New()
	transactionID, err := base64.StdEncoding.DecodeString("jxhBARZwX+rsC6er")
	if err != nil {
		t.Error(err)
	}
	copy(m.TransactionID[:], transactionID)
	expectedIP := net.ParseIP("fe80::dc2b:44ff:fe20:6009")
	expectedPort := 21254
	addr := &XORMappedAddress{
		IP:   net.ParseIP("fe80::dc2b:44ff:fe20:6009"),
		Port: 21254,
	}
	addr.AddTo(m)
	m.WriteHeader()

	mRes := New()
	if _, err = mRes.ReadFrom(m.reader()); err != nil {
		t.Fatal(err)
	}
	gotAddr := new(XORMappedAddress)
	if err = gotAddr.GetFrom(m); err != nil {
		t.Fatal(err)
	}
	if !gotAddr.IP.Equal(expectedIP) {
		t.Error("bad IP", gotAddr.IP, "!=", expectedIP)
	}
	if gotAddr.Port != expectedPort {
		t.Error("bad Port", gotAddr.Port, "!=", expectedPort)
	}
}

func BenchmarkErrorCode_AddTo(b *testing.B) {
	m := New()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		CodeStaleNonce.AddTo(m)
		m.Reset()
	}
}

func BenchmarkErrorCodeAttribute_AddTo(b *testing.B) {
	m := New()
	b.ReportAllocs()
	a := &ErrorCodeAttribute{
		Code:   404,
		Reason: []byte("not found!"),
	}
	for i := 0; i < b.N; i++ {
		a.AddTo(m)
		m.Reset()
	}
}

func BenchmarkErrorCodeAttribute_GetFrom(b *testing.B) {
	m := New()
	b.ReportAllocs()
	a := &ErrorCodeAttribute{
		Code:   404,
		Reason: []byte("not found!"),
	}
	a.AddTo(m)
	for i := 0; i < b.N; i++ {
		a.GetFrom(m)
	}
}

func TestMessage_AddErrorCode(t *testing.T) {
	m := New()
	transactionID, err := base64.StdEncoding.DecodeString("jxhBARZwX+rsC6er")
	if err != nil {
		t.Error(err)
	}
	copy(m.TransactionID[:], transactionID)
	expectedCode := ErrorCode(428)
	expectedReason := "Stale Nonce"
	CodeStaleNonce.AddTo(m)
	m.WriteHeader()

	mRes := New()
	if _, err = mRes.ReadFrom(m.reader()); err != nil {
		t.Fatal(err)
	}
	errCodeAttr := new(ErrorCodeAttribute)
	if err = errCodeAttr.GetFrom(mRes); err != nil {
		t.Error(err)
	}
	code := errCodeAttr.Code
	if err != nil {
		t.Error(err)
	}
	if code != expectedCode {
		t.Error("bad code", code)
	}
	if string(errCodeAttr.Reason) != expectedReason {
		t.Error("bad reason", string(errCodeAttr.Reason))
	}
}

func BenchmarkMessage_GetNotFound(b *testing.B) {
	m := New()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m.Get(AttrRealm)
	}
}