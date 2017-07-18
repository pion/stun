package stun

import (
	"io"
	"testing"
	"time"
)

type NoopAgent struct {
	f chan AgentFn
}

func (n *NoopAgent) Close() error {
	close(n.f)
	return nil
}

func (NoopAgent) Collect(time.Time) error { return nil }

func (NoopAgent) Process(m *Message) error { return nil }

func (n *NoopAgent) Start(id [TransactionIDSize]byte, deadline time.Time, f AgentFn) error {
	n.f <- f
	return nil
}

func (n *NoopAgent) Stop([TransactionIDSize]byte) error {
	return nil
}

type noopConnection struct{}

func (noopConnection) Write(b []byte) (int, error) {
	return len(b), nil
}

func (noopConnection) Read(b []byte) (int, error) {
	time.Sleep(time.Millisecond)
	return 0, io.EOF
}

func (noopConnection) Close() error {
	return nil
}

func BenchmarkClient_Do(b *testing.B) {
	b.ReportAllocs()
	agent := &NoopAgent{
		f: make(chan AgentFn),
	}
	client := NewClient(ClientOptions{
		Agent:      agent,
		Connection: noopConnection{},
	})
	defer client.Close()
	go func() {
		e := AgentEvent{
			Error:   nil,
			Message: nil,
		}
		for f := range agent.f {
			f(e)
		}
	}()
	m := new(Message)
	m.Encode()
	noopF := func(event AgentEvent) {
		// pass
	}
	for i := 0; i < b.N; i++ {
		if err := client.Do(m, time.Time{}, noopF); err != nil {
			b.Fatal(err)
		}
	}
}
