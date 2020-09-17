package stun

import (
	"errors"
	"io"
	"net"
	"testing"
)

func TestMappedAddress(t *testing.T) {
	m := new(Message)
	addr := &MappedAddress{
		IP:   net.ParseIP("122.12.34.5"),
		Port: 5412,
	}
	if addr.String() != "122.12.34.5:5412" {
		t.Error("bad string", addr)
	}
	t.Run("Bad length", func(t *testing.T) {
		badAddr := &MappedAddress{
			IP: net.IP{1, 2, 3},
		}
		if err := badAddr.AddTo(m); err == nil {
			t.Error("should error")
		}
	})
	t.Run("AddTo", func(t *testing.T) {
		if err := addr.AddTo(m); err != nil {
			t.Error(err)
		}
		t.Run("GetFrom", func(t *testing.T) {
			got := new(MappedAddress)
			if err := got.GetFrom(m); err != nil {
				t.Error(err)
			}
			if !got.IP.Equal(addr.IP) {
				t.Error("got bad IP: ", got.IP)
			}
			t.Run("Not found", func(t *testing.T) {
				message := new(Message)
				if err := got.GetFrom(message); !errors.Is(err, ErrAttributeNotFound) {
					t.Error("should be not found: ", err)
				}
			})
			t.Run("Bad family", func(t *testing.T) {
				v, _ := m.Attributes.Get(AttrMappedAddress)
				v.Value[0] = 32
				if err := got.GetFrom(m); err == nil {
					t.Error("should error")
				}
			})
			t.Run("Bad length", func(t *testing.T) {
				message := new(Message)
				message.Add(AttrMappedAddress, []byte{1, 2, 3})
				if err := got.GetFrom(message); !errors.Is(err, io.ErrUnexpectedEOF) {
					t.Errorf("<%s> should be <%s>", err, io.ErrUnexpectedEOF)
				}
			})
		})
	})
}

func TestMappedAddressV6(t *testing.T) { // nolint:dupl
	m := new(Message)
	addr := &MappedAddress{
		IP:   net.ParseIP("::"),
		Port: 5412,
	}
	t.Run("AddTo", func(t *testing.T) {
		if err := addr.AddTo(m); err != nil {
			t.Error(err)
		}
		t.Run("GetFrom", func(t *testing.T) {
			got := new(MappedAddress)
			if err := got.GetFrom(m); err != nil {
				t.Error(err)
			}
			if !got.IP.Equal(addr.IP) {
				t.Error("got bad IP: ", got.IP)
			}
			t.Run("Not found", func(t *testing.T) {
				message := new(Message)
				if err := got.GetFrom(message); !errors.Is(err, ErrAttributeNotFound) {
					t.Error("should be not found: ", err)
				}
			})
		})
	})
}

func TestAlternateServer(t *testing.T) { // nolint:dupl
	m := new(Message)
	addr := &AlternateServer{
		IP:   net.ParseIP("122.12.34.5"),
		Port: 5412,
	}
	t.Run("AddTo", func(t *testing.T) {
		if err := addr.AddTo(m); err != nil {
			t.Error(err)
		}
		t.Run("GetFrom", func(t *testing.T) {
			got := new(AlternateServer)
			if err := got.GetFrom(m); err != nil {
				t.Error(err)
			}
			if !got.IP.Equal(addr.IP) {
				t.Error("got bad IP: ", got.IP)
			}
			t.Run("Not found", func(t *testing.T) {
				message := new(Message)
				if err := got.GetFrom(message); !errors.Is(err, ErrAttributeNotFound) {
					t.Error("should be not found: ", err)
				}
			})
		})
	})
}

func TestOtherAddress(t *testing.T) { // nolint:dupl
	m := new(Message)
	addr := &OtherAddress{
		IP:   net.ParseIP("122.12.34.5"),
		Port: 5412,
	}
	t.Run("AddTo", func(t *testing.T) {
		if err := addr.AddTo(m); err != nil {
			t.Error(err)
		}
		t.Run("GetFrom", func(t *testing.T) {
			got := new(OtherAddress)
			if err := got.GetFrom(m); err != nil {
				t.Error(err)
			}
			if !got.IP.Equal(addr.IP) {
				t.Error("got bad IP: ", got.IP)
			}
			t.Run("Not found", func(t *testing.T) {
				message := new(Message)
				if err := got.GetFrom(message); !errors.Is(err, ErrAttributeNotFound) {
					t.Error("should be not found: ", err)
				}
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
