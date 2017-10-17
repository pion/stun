package stun

import (
	"io"
	"sync"
	"testing"
	"time"
)

type TestAgent struct {
	f chan AgentFn
}

func (n *TestAgent) Close() error {
	close(n.f)
	return nil
}

func (TestAgent) Collect(time.Time) error { return nil }

func (TestAgent) Process(m *Message) error { return nil }

func (n *TestAgent) Start(id [TransactionIDSize]byte, deadline time.Time, f AgentFn) error {
	n.f <- f
	return nil
}

func (n *TestAgent) Stop([TransactionIDSize]byte) error {
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
	agent := &TestAgent{
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

type testConnection struct {
	write   func([]byte) (int, error)
	b       []byte
	l       sync.Mutex
	stopped bool
}

func (t *testConnection) Write(b []byte) (int, error) {
	t.l.Unlock()
	return t.write(b)
}

func (t *testConnection) Close() error {
	t.stopped = true
	t.l.Unlock()
	return nil
}

func (t *testConnection) Read(b []byte) (int, error) {
	t.l.Lock()
	if t.stopped {
		return 0, io.EOF
	}
	return copy(b, t.b), nil
}

func TestClosedOrPanic(t *testing.T) {
	closedOrPanic(nil)
	closedOrPanic(ErrAgentClosed)
	func() {
		defer func() {
			r := recover()
			if r != io.EOF {
				t.Error(r)
			}
		}()
		closedOrPanic(io.EOF)
	}()
}

func TestClient_Do(t *testing.T) {
	response := MustBuild(TransactionID, BindingSuccess)
	response.Encode()
	conn := &testConnection{
		b: response.Raw,
		write: func(bytes []byte) (int, error) {
			return len(bytes), nil
		},
	}
	conn.l.Lock()
	c := NewClient(ClientOptions{
		Connection: conn,
	})
	defer func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
		if err := c.Close(); err == nil {
			t.Error("second close should fail")
		}
	}()
	m := new(Message)
	m.TransactionID = response.TransactionID
	m.Encode()
	d := time.Now().Add(time.Second)
	if err := c.Do(m, d, func(event AgentEvent) {
		if event.Error != nil {
			t.Error(event.Error)
		}
	}); err != nil {
		t.Error(err)
	}
}


func TestCloseErr_Error(t *testing.T) {
	for id, c := range []struct {
		Err CloseErr
		Out string
	}{
		{CloseErr{}, "failed to close: <nil> (connection), <nil> (agent)"},
		{CloseErr{
			AgentErr: io.ErrUnexpectedEOF,
		}, "failed to close: <nil> (connection), unexpected EOF (agent)"},
		{CloseErr{
			ConnectionErr: io.ErrUnexpectedEOF,
		}, "failed to close: unexpected EOF (connection), <nil> (agent)"},
	}{
		if out := c.Err.Error(); out != c.Out {
			t.Errorf("[%d]: Error(%#v) %q (got) != %q (expected)",
				id, c.Err, out, c.Out,
			)
		}
	}
}

func TestStopErr_Error(t *testing.T) {
	for id, c := range []struct {
		Err StopErr
		Out string
	}{
		{StopErr{}, "error while stopping due to <nil>: <nil>"},
		{StopErr{
			Err: io.ErrUnexpectedEOF,
		}, "error while stopping due to <nil>: unexpected EOF"},
		{StopErr{
			Cause: io.ErrUnexpectedEOF,
		}, "error while stopping due to unexpected EOF: <nil>"},
	}{
		if out := c.Err.Error(); out != c.Out {
			t.Errorf("[%d]: Error(%#v) %q (got) != %q (expected)",
				id, c.Err, out, c.Out,
			)
		}
	}
}