// SPDX-FileCopyrightText: 2026 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package stun

import (
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMappedAddress(t *testing.T) {
	msg := new(Message)
	addr := &MappedAddress{
		IP:   net.ParseIP("122.12.34.5"),
		Port: 5412,
	}
	assert.Equal(t, "122.12.34.5:5412", addr.String(), "bad string")
	t.Run("Bad length", func(t *testing.T) {
		badAddr := &MappedAddress{
			IP: net.IP{1, 2, 3},
		}
		assert.Error(t, badAddr.AddTo(msg), "should error")
	})
	t.Run("AddTo", func(t *testing.T) {
		assert.NoError(t, addr.AddTo(msg))
		t.Run("GetFrom", func(t *testing.T) {
			got := new(MappedAddress)
			assert.NoError(t, got.GetFrom(msg))
			assert.True(t, got.IP.Equal(addr.IP), "got bad IP: %v", got.IP)
			t.Run("Not found", func(t *testing.T) {
				message := new(Message)
				assert.ErrorIs(t, got.GetFrom(message), ErrAttributeNotFound, "should be not found")
			})
			t.Run("Bad family", func(t *testing.T) {
				v, _ := msg.Attributes.Get(AttrMappedAddress)
				v.Value[0] = 32
				assert.Error(t, got.GetFrom(msg), "should error")
			})
			t.Run("Bad length", func(t *testing.T) {
				message := new(Message)
				message.Add(AttrMappedAddress, []byte{1, 2, 3})
				assert.ErrorIs(t, got.GetFrom(message), io.ErrUnexpectedEOF)
			})
		})
	})
}

func TestMappedAddressV6(t *testing.T) { //nolint:dupl
	m := new(Message)
	addr := &MappedAddress{
		IP:   net.ParseIP("::"),
		Port: 5412,
	}
	t.Run("AddTo", func(t *testing.T) {
		assert.NoError(t, addr.AddTo(m))
		t.Run("GetFrom", func(t *testing.T) {
			got := new(MappedAddress)
			assert.NoError(t, got.GetFrom(m))
			assert.True(t, got.IP.Equal(addr.IP), "got bad IP: %v", got.IP)
			t.Run("Not found", func(t *testing.T) {
				message := new(Message)
				assert.ErrorIs(t, got.GetFrom(message), ErrAttributeNotFound, "should be not found")
			})
		})
	})
}

func TestAlternateServer(t *testing.T) { //nolint:dupl
	m := new(Message)
	addr := &AlternateServer{
		IP:   net.ParseIP("122.12.34.5"),
		Port: 5412,
	}
	t.Run("AddTo", func(t *testing.T) {
		assert.NoError(t, addr.AddTo(m))
		t.Run("GetFrom", func(t *testing.T) {
			got := new(AlternateServer)
			assert.NoError(t, got.GetFrom(m))
			assert.True(t, got.IP.Equal(addr.IP), "got bad IP: %v", got.IP)
			t.Run("Not found", func(t *testing.T) {
				message := new(Message)
				assert.ErrorIs(t, got.GetFrom(message), ErrAttributeNotFound, "should be not found")
			})
		})
	})
}

func TestOtherAddress(t *testing.T) { //nolint:dupl
	m := new(Message)
	addr := &OtherAddress{
		IP:   net.ParseIP("122.12.34.5"),
		Port: 5412,
	}
	t.Run("AddTo", func(t *testing.T) {
		assert.NoError(t, addr.AddTo(m))
		t.Run("GetFrom", func(t *testing.T) {
			got := new(OtherAddress)
			assert.NoError(t, got.GetFrom(m))
			assert.True(t, got.IP.Equal(addr.IP), "got bad IP: %v", got.IP)
			t.Run("Not found", func(t *testing.T) {
				message := new(Message)
				assert.ErrorIs(t, got.GetFrom(message), ErrAttributeNotFound, "should be not found")
			})
		})
	})
}

func BenchmarkMappedAddress_AddTo(b *testing.B) {
	m := new(Message)
	b.ReportAllocs()
	addr := &MappedAddress{
		IP:   net.ParseIP("122.12.34.5"),
		Port: 5412,
	}
	for i := 0; i < b.N; i++ {
		if err := addr.AddTo(m); err != nil {
			b.Fatal(err)
		}
		m.Reset()
	}
}

func BenchmarkAlternateServer_AddTo(b *testing.B) {
	m := new(Message)
	b.ReportAllocs()
	addr := &AlternateServer{
		IP:   net.ParseIP("122.12.34.5"),
		Port: 5412,
	}
	for i := 0; i < b.N; i++ {
		if err := addr.AddTo(m); err != nil {
			b.Fatal(err)
		}
		m.Reset()
	}
}

func TestMappedAddress_GetFrom_WrongValueLength(t *testing.T) {
	ipv4 := []byte{0, 1, 0x15, 0x24, 122, 12, 34, 5}
	ipv6 := append([]byte{0, 2, 0x15, 0x24}, make([]byte, net.IPv6len)...)
	cases := map[string]struct {
		value []byte
		err   error
	}{
		"IPv4 truncated": {value: ipv4[:6], err: io.ErrUnexpectedEOF},
		"IPv6 truncated": {value: ipv6[:12], err: io.ErrUnexpectedEOF},
		"IPv4 too long":  {value: append(append([]byte{}, ipv4...), 1), err: ErrAttributeSizeOverflow},
		"IPv4 exact":     {value: ipv4},
		"IPv6 exact":     {value: ipv6},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			msg := new(Message)
			msg.Add(AttrMappedAddress, testCase.value)
			addr := new(MappedAddress)
			if testCase.err != nil {
				assert.ErrorIs(t, addr.GetFrom(msg), testCase.err)

				return
			}
			assert.NoError(t, addr.GetFrom(msg))
			assert.Equal(t, 5412, addr.Port)
		})
	}
}
