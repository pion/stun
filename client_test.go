package stun

import (
	"net"
	"testing"
)

type stubMultiplexer struct {
	buf      []byte
	response []byte
	addr     net.Addr
	clients  []multiplexClient
}

func (m *stubMultiplexer) Add(f MultiplexFunc, c func([]byte, net.Addr)) {
	m.clients = append(m.clients, multiplexClient{
		f: f,
		c: c,
	})
}

func (m *stubMultiplexer) WriteTo(b []byte, addr net.Addr) (int, error) {
	m.buf = append(m.buf, b...)
	m.addr = addr
	go func() {
		for _, client := range m.clients {
			if !client.f(m.response) {
				continue
			}
			client.c(m.response, m.addr)
		}
	}()
	return len(b), nil
}


func TestClient_Do(t *testing.T) {
	client := &Client{}
	m := &stubMultiplexer{
		addr: &net.UDPAddr{
			IP: net.IPv4(1, 2, 3, 4),
		},
	}
	client.Multiplex(m)
	if err := client.Dial(m.addr); err != nil {
		t.Error(err)
	}
	request, err := Build(TransactionID, BindingRequest)
	if err != nil {
		t.Fatal(err)
	}
	response, err := Build(NewTransactionIDSetter(request.TransactionID), BindingSuccess)
	if err != nil {
		t.Fatal(err)
	}
	m.response = response.Raw
	if err := client.Do(request, func(res *Message) error {
		if !res.Equal(response) {
			t.Error("not equal")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
