package stun

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"log"
	"os"
	"testing"
	"time"
)

type TestAgent struct {
	f chan Handler
}

func (n *TestAgent) Close() error {
	close(n.f)
	return nil
}

func (TestAgent) Collect(time.Time) error { return nil }

func (TestAgent) Process(m *Message) error { return nil }

func (n *TestAgent) Start(id [TransactionIDSize]byte, deadline time.Time, f Handler) error {
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
		f: make(chan Handler, 1000),
	}
	client, err := NewClient(ClientOptions{
		Agent:      agent,
		Connection: noopConnection{},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	go func() {
		e := Event{
			Error:   nil,
			Message: nil,
		}
		for f := range agent.f {
			f.HandleEvent(e)
		}
	}()
	m := new(Message)
	m.Encode()
	noopF := func(event Event) {
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
	stopped bool
}

func (t *testConnection) Write(b []byte) (int, error) {
	return t.write(b)
}

func (t *testConnection) Close() error {
	if t.stopped {
		return errors.New("already stopped")
	}
	t.stopped = true
	return nil
}

func (t *testConnection) Read(b []byte) (int, error) {
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
	c, err := NewClient(ClientOptions{
		Connection: conn,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
		if err := c.Close(); err == nil {
			t.Error("second close should fail")
		}
		if err := c.Do(MustBuild(TransactionID), time.Time{}, nil); err == nil {
			t.Error("Do after Close should fail")
		}
	}()
	m := MustBuild(
		NewTransactionIDSetter(response.TransactionID),
	)
	d := time.Now().Add(time.Second)
	if err := c.Do(m, d, func(event Event) {
		if event.Error != nil {
			t.Error(event.Error)
		}
	}); err != nil {
		t.Error(err)
	}
	m = MustBuild(TransactionID)
	if err := c.Do(m, d, nil); err != nil {
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
	} {
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
	} {
		if out := c.Err.Error(); out != c.Out {
			t.Errorf("[%d]: Error(%#v) %q (got) != %q (expected)",
				id, c.Err, out, c.Out,
			)
		}
	}
}

type errorAgent struct {
	startErr error
	stopErr  error
	closeErr error
}

func (a errorAgent) Close() error { return a.closeErr }

func (errorAgent) Collect(time.Time) error { return nil }

func (errorAgent) Process(m *Message) error { return nil }

func (a errorAgent) Start(id [TransactionIDSize]byte, deadline time.Time, f Handler) error {
	return a.startErr
}

func (a errorAgent) Stop([TransactionIDSize]byte) error {
	return a.stopErr
}

func TestClientAgentError(t *testing.T) {
	response := MustBuild(TransactionID, BindingSuccess)
	response.Encode()
	conn := &testConnection{
		b: response.Raw,
		write: func(bytes []byte) (int, error) {
			return len(bytes), nil
		},
	}
	c, err := NewClient(ClientOptions{
		Agent: errorAgent{
			startErr: io.ErrUnexpectedEOF,
		},
		Connection: conn,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
	}()
	m := MustBuild(NewTransactionIDSetter(response.TransactionID))
	if err := c.Do(m, time.Time{}, nil); err != nil {
		t.Error(err)
	}
	if err := c.Do(m, time.Time{}, func(event Event) {
		if event.Error == nil {
			t.Error("error expected")
		}
	}); err != io.ErrUnexpectedEOF {
		t.Error("error expected")
	}
}

func TestClientConnErr(t *testing.T) {
	conn := &testConnection{
		write: func(bytes []byte) (int, error) {
			return 0, io.ErrClosedPipe
		},
	}
	c, err := NewClient(ClientOptions{
		Connection: conn,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
	}()
	m := MustBuild(TransactionID)
	if err := c.Do(m, time.Time{}, nil); err == nil {
		t.Error("error expected")
	}
	if err := c.Do(m, time.Time{}, noopHandler); err == nil {
		t.Error("error expected")
	}
}

func TestClientConnErrStopErr(t *testing.T) {
	conn := &testConnection{
		write: func(bytes []byte) (int, error) {
			return 0, io.ErrClosedPipe
		},
	}
	c, err := NewClient(ClientOptions{
		Connection: conn,
		Agent: errorAgent{
			stopErr: io.ErrUnexpectedEOF,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
	}()
	m := MustBuild(TransactionID)
	if err := c.Do(m, time.Time{}, noopHandler); err == nil {
		t.Error("error expected")
	}
}

func TestCallbackWaitHandler_setCallback(t *testing.T) {
	c := callbackWaitHandler{}
	defer func() {
		if err := recover(); err == nil {
			t.Error("should panic")
		}
	}()
	c.setCallback(nil)
}

func TestCallbackWaitHandler_HandleEvent(t *testing.T) {
	c := callbackWaitHandler{}
	defer func() {
		if err := recover(); err == nil {
			t.Error("should panic")
		}
	}()
	c.HandleEvent(Event{})
}

func TestNewClientNoConnection(t *testing.T) {
	c, err := NewClient(ClientOptions{
		Connection: nil,
	})
	if c != nil {
		t.Error("c should be nil")
	}
	if err != ErrNoConnection {
		t.Error("bad error")
	}
}

func TestDial(t *testing.T) {
	c, err := Dial("udp4", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = c.Close(); err != nil {
			t.Error(err)
		}
	}()
}

func TestDialError(t *testing.T) {
	_, err := Dial("bad?network", "?????")
	if err == nil {
		t.Fatal("error expected")
	}
}
func TestClientCloseErr(t *testing.T) {
	response := MustBuild(TransactionID, BindingSuccess)
	response.Encode()
	conn := &testConnection{
		b: response.Raw,
		write: func(bytes []byte) (int, error) {
			return len(bytes), nil
		},
	}
	c, err := NewClient(ClientOptions{
		Agent: errorAgent{
			closeErr: io.ErrUnexpectedEOF,
		},
		Connection: conn,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err, ok := c.Close().(CloseErr); !ok || err.AgentErr != io.ErrUnexpectedEOF {
			t.Error("unexpected close err")
		}
	}()
}

type gcWaitAgent struct {
	gc chan struct{}
}

func (a *gcWaitAgent) Stop(id [TransactionIDSize]byte) error {
	return nil
}

func (a *gcWaitAgent) Close() error {
	close(a.gc)
	return nil
}

func (a *gcWaitAgent) Collect(time.Time) error {
	a.gc <- struct{}{}
	return nil
}

func (a *gcWaitAgent) Process(m *Message) error {
	return nil
}

func (a *gcWaitAgent) Start(id [TransactionIDSize]byte, deadline time.Time, f Handler) error {
	return nil
}

func TestClientGC(t *testing.T) {
	response := MustBuild(TransactionID, BindingSuccess)
	response.Encode()
	conn := &testConnection{
		b: response.Raw,
		write: func(bytes []byte) (int, error) {
			return len(bytes), nil
		},
	}
	agent := &gcWaitAgent{
		gc: make(chan struct{}),
	}
	c, err := NewClient(ClientOptions{
		Agent:       agent,
		Connection:  conn,
		TimeoutRate: time.Millisecond * 1,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err = c.Close(); err != nil {
			t.Error(err)
		}
	}()
	select {
	case <-agent.gc:
	case <-time.After(time.Millisecond * 200):
		t.Error("timed out")
	}
}

func TestClientCheckInit(t *testing.T) {
	if err := (&Client{}).Indicate(nil); err != ErrClientNotInitialized {
		t.Error("unexpected error")
	}
	if err := (&Client{}).Do(nil, time.Time{}, nil); err != ErrClientNotInitialized {
		t.Error("unexpected error")
	}
}

func captureLog() (*bytes.Buffer, func()) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	f := log.Flags()
	log.SetFlags(0)
	return &buf, func() {
		log.SetFlags(f)
		log.SetOutput(os.Stderr)
	}
}

func TestClientFinalizer(t *testing.T) {
	buf, stopCapture := captureLog()
	defer stopCapture()
	clientFinalizer(nil) // should not panic
	clientFinalizer(&Client{})
	conn := &testConnection{
		write: func(bytes []byte) (int, error) {
			return 0, io.ErrClosedPipe
		},
	}
	c, err := NewClient(ClientOptions{
		Connection: conn,
	})
	if err != nil {
		log.Fatal(err)
	}
	clientFinalizer(c)
	clientFinalizer(c)
	response := MustBuild(TransactionID, BindingSuccess)
	response.Encode()
	conn = &testConnection{
		b: response.Raw,
		write: func(bytes []byte) (int, error) {
			return len(bytes), nil
		},
	}
	c, err = NewClient(ClientOptions{
		Agent: errorAgent{
			closeErr: io.ErrUnexpectedEOF,
		},
		Connection: conn,
	})
	if err != nil {
		log.Fatal(err)
	}
	clientFinalizer(c)
	reader := bufio.NewScanner(buf)
	var lines int
	var expectedLines = []string{
		"client: called finalizer on non-closed client: client not initialized",
		"client: called finalizer on non-closed client",
		"client: called finalizer on non-closed client: failed to close: " +
			"<nil> (connection), unexpected EOF (agent)",
	}
	for reader.Scan() {
		if reader.Text() != expectedLines[lines] {
			t.Error(reader.Text(), "!=", expectedLines[lines])
		}
		lines++
	}
	if reader.Err() != nil {
		t.Error(err)
	}
	if lines != 3 {
		t.Error("incorrect count of log lines:", lines)
	}
}
