package stun

import (
	"net"
	"testing"
	"time"
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
	m.buf = append(m.buf[:0], b...)
	m.addr = addr
	for _, client := range m.clients {
		if !client.f(m.response) {
			continue
		}
		client.c(m.response, m.addr)
	}
	return len(b), nil
}

func BenchmarkClient_AddTransaction(b *testing.B) {
	b.ReportAllocs()
	c := &Client{}
	id := transactionID{}
	for i := 0; i < b.N; i++ {
		c.addTransaction(id)
	}
}

func BenchmarkClient_Do(b *testing.B) {
	b.ReportAllocs()
	client := &Client{}
	m := &stubMultiplexer{
		addr: &net.UDPAddr{
			IP: net.IPv4(1, 2, 3, 4),
		},
	}
	client.Multiplex(m)
	if err := client.Dial(m.addr); err != nil {
		b.Error(err)
	}

	var (
		response = MustBuild(TransactionID, BindingSuccess)
		request  = MustBuild(NewTransactionIDSetter(response.TransactionID), BindingRequest)
	)
	m.response = response.Raw

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := client.Do(request, func(res *Message) error {
			return nil
		}); err != nil {
			b.Fatal(err)
		}
	}
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

func TestAgent_Process(t *testing.T) {
	a := NewAgent(AgentOptions{})
	if err := a.Close(); err != nil {
		t.Error(err)
	}
}

var noopHandler = func(e AgentEvent) {}

func BenchmarkAgent_Process(b *testing.B) {
	a := NewAgent(AgentOptions{
		Handler: noopHandler,
	})
	deadline := time.Now().AddDate(0, 0, 1)
	for i := 0; i < 1000; i++ {
		if err := a.Start(NewTransactionID(), deadline, noopHandler); err != nil {
			b.Fatal(err)
		}
	}
	defer func() {
		if err := a.Close(); err != nil {
			b.Error(err)
		}
	}()
	b.ReportAllocs()
	ev := AgentProcessArgs{
		Message: MustBuild(
			TransactionID,
		),
	}
	for i := 0; i < b.N; i++ {
		if err := a.Process(ev); err != nil {
			b.Fatal(err)
		}
	}
}
