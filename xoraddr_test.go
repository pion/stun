// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

//go:build !js
// +build !js

package stun

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkXORMappedAddress_AddTo(b *testing.B) {
	m := New()
	b.ReportAllocs()
	ip := net.ParseIP("192.168.1.32")
	for i := 0; i < b.N; i++ {
		addr := &XORMappedAddress{IP: ip, Port: 3654}
		addr.AddTo(m) //nolint:errcheck,gosec
		m.Reset()
	}
}

func BenchmarkXORMappedAddress_GetFrom(b *testing.B) {
	msg := New()
	transactionID, err := base64.StdEncoding.DecodeString("jxhBARZwX+rsC6er")
	assert.NoError(b, err)
	copy(msg.TransactionID[:], transactionID)
	addrValue, err := hex.DecodeString("00019cd5f49f38ae")
	assert.NoError(b, err)
	msg.Add(AttrXORMappedAddress, addrValue)
	addr := new(XORMappedAddress)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		assert.NoError(b, addr.GetFrom(msg))
	}
}

func TestXORMappedAddress_GetFrom(t *testing.T) {
	m := New()
	transactionID, err := base64.StdEncoding.DecodeString("jxhBARZwX+rsC6er")
	assert.NoError(t, err)
	copy(m.TransactionID[:], transactionID)
	addrValue, err := hex.DecodeString("00019cd5f49f38ae")
	assert.NoError(t, err)
	m.Add(AttrXORMappedAddress, addrValue)
	addr := new(XORMappedAddress)
	assert.NoError(t, addr.GetFrom(m))
	assert.True(t, addr.IP.Equal(net.ParseIP("213.141.156.236")))
	assert.Equal(t, 48583, addr.Port)
	t.Run("UnexpectedEOF", func(t *testing.T) {
		m := New()
		// {0, 1} is correct addr family.
		m.Add(AttrXORMappedAddress, []byte{0, 1, 3, 4})
		addr := new(XORMappedAddress)
		assert.ErrorIs(t, addr.GetFrom(m), io.ErrUnexpectedEOF, "len(v) = 4 should return io.ErrUnexpectedEOF")
	})
	t.Run("AttrOverflowErr", func(t *testing.T) {
		m := New()
		// {0, 1} is correct addr family.
		m.Add(AttrXORMappedAddress, []byte{0, 1, 3, 4, 5, 6, 7, 8, 9, 1, 1, 1, 1, 1, 2, 3, 4})
		addr := new(XORMappedAddress)
		assert.True(t, IsAttrSizeOverflow(addr.GetFrom(m)), "GetFrom should return *AttrOverflowErr")
	})
}

func TestXORMappedAddress_GetFrom_Invalid(t *testing.T) {
	msg := New()
	transactionID, err := base64.StdEncoding.DecodeString("jxhBARZwX+rsC6er")
	assert.NoError(t, err)
	copy(msg.TransactionID[:], transactionID)
	expectedIP := net.ParseIP("213.141.156.236")
	expectedPort := 21254
	addr := new(XORMappedAddress)

	assert.Error(t, addr.GetFrom(msg))

	addr.IP = expectedIP
	addr.Port = expectedPort
	addr.AddTo(msg) //nolint:errcheck,gosec
	msg.WriteHeader()

	mRes := New()
	binary.BigEndian.PutUint16(msg.Raw[20+4:20+4+2], 0x21)
	_, err = mRes.ReadFrom(bytes.NewReader(msg.Raw))
	assert.NoError(t, err)
	assert.Error(t, addr.GetFrom(msg))
}

func TestXORMappedAddress_AddTo(t *testing.T) {
	msg := New()
	transactionID, err := base64.StdEncoding.DecodeString("jxhBARZwX+rsC6er")
	assert.NoError(t, err)
	copy(msg.TransactionID[:], transactionID)
	expectedIP := net.ParseIP("213.141.156.236")
	expectedPort := 21254
	addr := &XORMappedAddress{
		IP:   net.ParseIP("213.141.156.236"),
		Port: expectedPort,
	}
	assert.NoError(t, addr.AddTo(msg))
	msg.WriteHeader()
	mRes := New()
	_, err = mRes.Write(msg.Raw)
	assert.NoError(t, err)
	assert.NoError(t, addr.GetFrom(mRes))
	assert.True(t, addr.IP.Equal(expectedIP), "Expected %s, got %s", expectedIP, addr.IP)
	assert.Equal(t, expectedPort, addr.Port)
}

func TestXORMappedAddress_AddTo_IPv6(t *testing.T) {
	msg := New()
	transactionID, err := base64.StdEncoding.DecodeString("jxhBARZwX+rsC6er")
	assert.NoError(t, err)
	copy(msg.TransactionID[:], transactionID)
	expectedIP := net.ParseIP("fe80::dc2b:44ff:fe20:6009")
	expectedPort := 21254
	addr := &XORMappedAddress{
		IP:   net.ParseIP("fe80::dc2b:44ff:fe20:6009"),
		Port: 21254,
	}
	addr.AddTo(msg) //nolint:errcheck,gosec
	msg.WriteHeader()

	mRes := New()
	_, err = mRes.ReadFrom(msg.reader())
	assert.NoError(t, err)
	gotAddr := new(XORMappedAddress)
	assert.NoError(t, gotAddr.GetFrom(mRes))
	assert.True(t, gotAddr.IP.Equal(expectedIP), "Expected %s, got %s", expectedIP, gotAddr.IP)
	assert.Equal(t, expectedPort, gotAddr.Port)
}

func TestXORMappedAddress_AddTo_Invalid(t *testing.T) {
	m := New()
	addr := &XORMappedAddress{
		IP:   []byte{1, 2, 3, 4, 5, 6, 7, 8},
		Port: 21254,
	}
	assert.ErrorIs(t, addr.AddTo(m), ErrBadIPLength)
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
		assert.Equalf(t, c.out, c.in.String(), "[%d]: XORMappesAddres.String()", i)
	}
}
