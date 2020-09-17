// +build !js

package stun

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"testing"
)

func BenchmarkXORMappedAddress_AddTo(b *testing.B) {
	m := New()
	b.ReportAllocs()
	ip := net.ParseIP("192.168.1.32")
	for i := 0; i < b.N; i++ {
		addr := &XORMappedAddress{IP: ip, Port: 3654}
		addr.AddTo(m) // nolint:errcheck
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

func TestXORMappedAddress_GetFrom(t *testing.T) {
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
	t.Run("UnexpectedEOF", func(t *testing.T) {
		m := New()
		// {0, 1} is correct addr family.
		m.Add(AttrXORMappedAddress, []byte{0, 1, 3, 4})
		addr := new(XORMappedAddress)
		if err = addr.GetFrom(m); !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Errorf("len(v) = 4 should render <%s> error, got <%s>",
				io.ErrUnexpectedEOF, err,
			)
		}
	})
	t.Run("AttrOverflowErr", func(t *testing.T) {
		m := New()
		// {0, 1} is correct addr family.
		m.Add(AttrXORMappedAddress, []byte{0, 1, 3, 4, 5, 6, 7, 8, 9, 1, 1, 1, 1, 1, 2, 3, 4})
		addr := new(XORMappedAddress)
		if err := addr.GetFrom(m); !IsAttrSizeOverflow(err) {
			t.Errorf("AddTo should return *AttrOverflowErr, got: %v", err)
		}
	})
}

func TestXORMappedAddress_GetFrom_Invalid(t *testing.T) {
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
	addr.AddTo(m) // nolint:errcheck
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

func TestXORMappedAddress_AddTo(t *testing.T) {
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

func TestXORMappedAddress_AddTo_IPv6(t *testing.T) {
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
	addr.AddTo(m) // nolint:errcheck
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

func TestXORMappedAddress_AddTo_Invalid(t *testing.T) {
	m := New()
	addr := &XORMappedAddress{
		IP:   []byte{1, 2, 3, 4, 5, 6, 7, 8},
		Port: 21254,
	}
	if err := addr.AddTo(m); !errors.Is(err, ErrBadIPLength) {
		t.Errorf("AddTo should return %q, got: %v", ErrBadIPLength, err)
	}
}

func TestXORMappedAddress_String(t *testing.T) {
	tt := []struct {
		in  XORMappedAddress
		out string
	}{
		{
			// 0
			XORMappedAddress{
				IP:   net.ParseIP("fe80::dc2b:44ff:fe20:6009"),
				Port: 124,
			}, "[fe80::dc2b:44ff:fe20:6009]:124",
		},
		{
			// 1
			XORMappedAddress{
				IP:   net.ParseIP("213.141.156.236"),
				Port: 8147,
			}, "213.141.156.236:8147",
		},
	}
	for i, c := range tt {
		if got := c.in.String(); got != c.out {
			t.Errorf("[%d]: XORMappesAddres.String() %s (got) != %s (expected)",
				i,
				got,
				c.out,
			)
		}
	}
}
