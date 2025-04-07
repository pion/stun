// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

//go:build !js
// +build !js

package stun

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	errClientAlreadyStopped = errors.New("already stopped")
	errClientReadTimedOut   = errors.New("read timed out")
	errClientWriteTimedOut  = errors.New("write timed out")
	errClientCloseError     = errors.New("close error")
	errClientSetHandler     = errors.New("set handler error")
	errClientStart          = errors.New("start error")
	errClientAgentCantStop  = errors.New("agent does not want to stop")
	errClientStartRefused   = errors.New("start refused")
)

type TestAgent struct {
	h Handler
	e chan Event
}

func (n *TestAgent) SetHandler(h Handler) error {
	n.h = h

	return nil
}

func (n *TestAgent) Close() error {
	close(n.e)

	return nil
}

func (TestAgent) Collect(time.Time) error { return nil }

func (TestAgent) Process(*Message) error { return nil }

func (n *TestAgent) Start(id [TransactionIDSize]byte, _ time.Time) error {
	n.e <- Event{
		TransactionID: id,
	}

	return nil
}

func (n *TestAgent) Stop([TransactionIDSize]byte) error {
	return nil
}

type noopConnection struct{}

func (noopConnection) Write(b []byte) (int, error) {
	return len(b), nil
}

func (noopConnection) Read([]byte) (int, error) {
	time.Sleep(time.Millisecond)

	return 0, io.EOF
}

func (noopConnection) Close() error {
	return nil
}

func BenchmarkClient_Do(b *testing.B) {
	b.ReportAllocs()
	agent := &TestAgent{
		e: make(chan Event, 1000),
	}
	client, err := NewClient(noopConnection{},
		WithAgent(agent),
	)
	assert.NoError(b, err)
	defer func() {
		assert.NoError(b, client.Close())
	}()

	noopF := func(Event) {
		// pass
	}
	b.RunParallel(func(pb *testing.PB) {
		go func() {
			for e := range agent.e {
				agent.h(e)
			}
		}()
		m := New()
		m.NewTransactionID() //nolint:errcheck,gosec
		m.Encode()
		for pb.Next() {
			if err := client.Do(m, noopF); err != nil {
				b.Error(err)
			}
		}
	})
}

type testConnection struct {
	write      func([]byte) (int, error)
	read       func([]byte) (int, error)
	close      func() error
	b          []byte
	stopped    bool
	stoppedMux sync.Mutex
}

func (t *testConnection) Write(b []byte) (int, error) {
	return t.write(b)
}

func (t *testConnection) Close() error {
	if t.close != nil {
		return t.close()
	}
	t.stoppedMux.Lock()
	defer t.stoppedMux.Unlock()
	if t.stopped {
		return errClientAlreadyStopped
	}
	t.stopped = true

	return nil
}

func (t *testConnection) Read(b []byte) (int, error) {
	t.stoppedMux.Lock()
	defer t.stoppedMux.Unlock()
	if t.stopped {
		return 0, io.EOF
	}
	if t.read != nil {
		return t.read(b)
	}

	return copy(b, t.b), nil
}

func TestClosedOrPanic(t *testing.T) {
	closedOrPanic(nil)
	closedOrPanic(ErrAgentClosed)
	func() {
		defer func() {
			r, ok := recover().(error)
			assert.True(t, ok, "should be error")
			assert.ErrorIs(t, r, io.EOF)
		}()
		closedOrPanic(io.EOF)
	}()
}

func TestClient_Start(t *testing.T) { //nolint:cyclop
	response := MustBuild(TransactionID, BindingSuccess)
	response.Encode()
	write := make(chan struct{}, 1)
	read := make(chan struct{}, 1)
	conn := &testConnection{
		b: response.Raw,
		read: func(i []byte) (int, error) {
			t.Log("waiting for read")
			select {
			case <-read:
				t.Log("reading")
				copy(i, response.Raw)

				return len(response.Raw), nil
			case <-time.After(time.Millisecond * 10):
				return 0, errClientReadTimedOut
			}
		},
		write: func(bytes []byte) (int, error) {
			t.Log("waiting for write")
			select {
			case <-write:
				t.Log("writing")

				return len(bytes), nil
			case <-time.After(time.Millisecond * 10):
				return 0, errClientWriteTimedOut
			}
		},
	}
	client, err := NewClient(conn)
	assert.NoError(t, err)
	defer func() {
		assert.NoError(t, client.Close())
		assert.Error(t, client.Close(), "second close should fail")
		assert.Error(t, client.Do(MustBuild(TransactionID), nil), "Do after Close should fail")
	}()
	msg := MustBuild(response, BindingRequest)
	t.Log("init")
	got := make(chan struct{})
	write <- struct{}{}
	t.Log("starting the first transaction")
	assert.NoError(t, client.Start(msg, func(event Event) {
		t.Log("got first transaction callback")
		assert.NoError(t, event.Error)
		got <- struct{}{}
	}))
	t.Log("starting the second transaction")
	assert.ErrorIs(t, client.Start(msg, func(Event) {
		assert.Fail(t, "should not be called")
	}), ErrTransactionExists)
	read <- struct{}{}
	select {
	case <-got:
		// pass
	case <-time.After(time.Millisecond * 10):
		assert.Fail(t, "timed out")
	}
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
	client, err := NewClient(conn)
	assert.NoError(t, err)
	defer func() {
		assert.NoError(t, client.Close())
		assert.Error(t, client.Close(), "second close should fail")
		assert.Error(t, client.Do(MustBuild(TransactionID), nil), "Do after Close should fail")
	}()
	m := MustBuild(
		NewTransactionIDSetter(response.TransactionID),
	)
	assert.NoError(t, client.Do(m, func(event Event) {
		assert.NoError(t, event.Error)
	}))
	m = MustBuild(TransactionID)
	assert.NoError(t, client.Do(m, nil))
}

func TestCloseErr_Error(t *testing.T) {
	for id, testCase := range []struct {
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
		assert.Equal(t, testCase.Out, testCase.Err.Error(), "[%d]: Error(%#v)", id, testCase.Err)
	}
}

func TestStopErr_Error(t *testing.T) {
	for id, testcase := range []struct {
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
		assert.Equal(t, testcase.Out, testcase.Err.Error(), "[%d]: Error(%#v)", id, testcase.Err)
	}
}

type errorAgent struct {
	startErr        error
	stopErr         error
	closeErr        error
	setHandlerError error
}

func (a errorAgent) SetHandler(Handler) error { return a.setHandlerError }

func (a errorAgent) Close() error { return a.closeErr }

func (errorAgent) Collect(time.Time) error { return nil }

func (errorAgent) Process(*Message) error { return nil }

func (a errorAgent) Start([TransactionIDSize]byte, time.Time) error {
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
	client, err := NewClient(conn,
		WithAgent(errorAgent{
			startErr: io.ErrUnexpectedEOF,
		}),
	)
	assert.NoError(t, err)
	defer func() {
		assert.NoError(t, client.Close())
	}()
	m := MustBuild(NewTransactionIDSetter(response.TransactionID))
	assert.NoError(t, client.Do(m, nil))
	assert.ErrorIs(t, client.Do(m, func(event Event) {
		assert.Error(t, event.Error, "error expected")
	}), io.ErrUnexpectedEOF)
}

func TestClientConnErr(t *testing.T) {
	conn := &testConnection{
		write: func([]byte) (int, error) {
			return 0, io.ErrClosedPipe
		},
	}
	client, err := NewClient(conn)
	assert.NoError(t, err)
	defer func() {
		assert.NoError(t, client.Close())
	}()
	m := MustBuild(TransactionID)
	assert.Error(t, client.Do(m, nil), "error expected")
	assert.Error(t, client.Do(m, NoopHandler()), "error expected")
}

func TestClientConnErrStopErr(t *testing.T) {
	conn := &testConnection{
		write: func([]byte) (int, error) {
			return 0, io.ErrClosedPipe
		},
	}
	client, err := NewClient(conn,
		WithAgent(errorAgent{
			stopErr: io.ErrUnexpectedEOF,
		}),
	)
	assert.NoError(t, err)
	defer func() {
		assert.NoError(t, client.Close())
	}()
	m := MustBuild(TransactionID)
	assert.Error(t, client.Do(m, NoopHandler()), "error expected")
}

func TestCallbackWaitHandler_setCallback(t *testing.T) {
	c := callbackWaitHandler{}
	defer func() {
		err := recover()
		assert.NotNil(t, err, "should panic")
	}()
	c.setCallback(nil)
}

func TestCallbackWaitHandler_HandleEvent(t *testing.T) {
	c := &callbackWaitHandler{
		cond: sync.NewCond(new(sync.Mutex)),
	}
	defer func() {
		err := recover()
		assert.NotNil(t, err, "should panic")
	}()
	c.HandleEvent(Event{})
}

func TestNewClientNoConnection(t *testing.T) {
	c, err := NewClient(nil)
	assert.Nil(t, c, "c should be nil")
	assert.ErrorIs(t, err, ErrNoConnection, "bad error")
}

func TestDial(t *testing.T) {
	c, err := Dial("udp4", "localhost:3458")
	assert.NoError(t, err)
	defer func() {
		assert.NoError(t, c.Close())
	}()
}

func TestDialURI(t *testing.T) {
	u, err := ParseURI("stun:localhost")
	assert.NoError(t, err)
	c, err := DialURI(u, &DialConfig{})
	assert.NoError(t, err)
	defer func() {
		assert.NoError(t, c.Close())
	}()
}

func TestDialError(t *testing.T) {
	_, err := Dial("bad?network", "?????")
	assert.Error(t, err, "error expected")
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
	c, err := NewClient(conn,
		WithAgent(errorAgent{
			closeErr: io.ErrUnexpectedEOF,
		}),
	)
	assert.NoError(t, err)
	defer func() {
		err, ok := c.Close().(CloseErr) //nolint:errorlint
		assert.True(t, ok, "should be CloseErr")
		assert.ErrorIs(t, err.AgentErr, io.ErrUnexpectedEOF, "unexpected close err")
	}()
}

func TestWithNoConnClose(t *testing.T) {
	response := MustBuild(TransactionID, BindingSuccess)
	response.Encode()
	closeErr := errClientCloseError
	conn := &testConnection{
		b: response.Raw,
		close: func() error {
			return closeErr
		},
	}
	c, err := NewClient(conn,
		WithAgent(errorAgent{
			closeErr: nil,
		}),
		WithNoConnClose(),
	)
	assert.NoError(t, err)
	assert.NoError(t, c.Close(), "unexpected non-nil error")
}

type gcWaitAgent struct {
	gc chan struct{}
}

func (a *gcWaitAgent) SetHandler(Handler) error {
	return nil
}

func (a *gcWaitAgent) Stop([TransactionIDSize]byte) error {
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

func (a *gcWaitAgent) Process(*Message) error {
	return nil
}

func (a *gcWaitAgent) Start([TransactionIDSize]byte, time.Time) error {
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
	c, err := NewClient(conn,
		WithAgent(agent),
		WithTimeoutRate(time.Millisecond),
	)
	assert.NoError(t, err)
	defer func() {
		assert.NoError(t, c.Close())
	}()
	select {
	case <-agent.gc:
	case <-time.After(time.Millisecond * 200):
		assert.Fail(t, "timed out")
	}
}

func TestClientCheckInit(t *testing.T) {
	assert.ErrorIs(t, (&Client{}).Indicate(nil), ErrClientNotInitialized)
	assert.ErrorIs(t, (&Client{}).Do(nil, nil), ErrClientNotInitialized)
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
		write: func([]byte) (int, error) {
			return 0, io.ErrClosedPipe
		},
	}
	client, err := NewClient(conn)
	assert.NoError(t, err)
	clientFinalizer(client)
	clientFinalizer(client)
	response := MustBuild(TransactionID, BindingSuccess)
	response.Encode()
	conn = &testConnection{
		b: response.Raw,
		write: func(bytes []byte) (int, error) {
			return len(bytes), nil
		},
	}
	client, err = NewClient(conn,
		WithAgent(errorAgent{
			closeErr: io.ErrUnexpectedEOF,
		}),
	)
	assert.NoError(t, err)
	clientFinalizer(client)
	reader := bufio.NewScanner(buf)
	var lines int
	expectedLines := []string{
		"client: called finalizer on non-closed client: client not initialized",
		"client: called finalizer on non-closed client",
		"client: called finalizer on non-closed client: failed to close: " +
			"<nil> (connection), unexpected EOF (agent)",
	}
	for reader.Scan() {
		assert.Equal(t, expectedLines[lines], reader.Text())
		lines++
	}
	assert.NoError(t, reader.Err())
	assert.Equal(t, 3, lines, "incorrect count of log lines")
}

func TestCallbackWaitHandler(*testing.T) {
	h := callbackWaitHandlerPool.Get().(*callbackWaitHandler) //nolint:forcetypeassert
	for i := 0; i < 100; i++ {
		h.setCallback(func(Event) {})
		go func() {
			time.Sleep(time.Microsecond * 100)
			h.HandleEvent(Event{})
		}()
		h.wait()
	}
}

type manualCollector struct {
	f func(t time.Time)
}

func (m *manualCollector) Collect(t time.Time) {
	m.f(t)
}

func (m *manualCollector) Start(_ time.Duration, f func(t time.Time)) error {
	m.f = f

	return nil
}

func (m *manualCollector) Close() error {
	return nil
}

type manualClock struct {
	mux     sync.Mutex
	current time.Time
}

func (m *manualClock) Add(d time.Duration) time.Time {
	m.mux.Lock()
	v := m.current.Add(d)
	m.current = v
	m.mux.Unlock()

	return v
}

func (m *manualClock) Now() time.Time {
	m.mux.Lock()
	defer m.mux.Unlock()

	return m.current
}

type manualAgent struct {
	start   func(id [TransactionIDSize]byte, deadline time.Time) error
	stop    func(id [TransactionIDSize]byte) error
	process func(m *Message) error
	h       Handler
}

func (n *manualAgent) SetHandler(h Handler) error {
	n.h = h

	return nil
}

func (n *manualAgent) Close() error {
	return nil
}

func (manualAgent) Collect(time.Time) error { return nil }

func (n *manualAgent) Process(m *Message) error {
	if n.process != nil {
		return n.process(m)
	}

	return nil
}

func (n *manualAgent) Start(id [TransactionIDSize]byte, deadline time.Time) error {
	return n.start(id, deadline)
}

func (n *manualAgent) Stop(id [TransactionIDSize]byte) error {
	if n.stop != nil {
		return n.stop(id)
	}

	return nil
}

func TestClientRetransmission(t *testing.T) {
	response := MustBuild(TransactionID, BindingSuccess)
	response.Encode()
	connL, connR := net.Pipe()
	defer func() {
		assert.NoError(t, connR.Close())
	}()
	collector := new(manualCollector)
	clock := &manualClock{current: time.Now()}
	agent := &manualAgent{}
	attempt := 0
	agent.start = func(id [TransactionIDSize]byte, _ time.Time) error {
		if attempt == 0 {
			attempt++
			go agent.h(Event{
				TransactionID: id,
				Error:         ErrTransactionTimeOut,
			})
		} else {
			go agent.h(Event{
				TransactionID: id,
				Message:       response,
			})
		}

		return nil
	}
	client, err := NewClient(connR,
		WithAgent(agent),
		WithClock(clock),
		WithCollector(collector),
		WithRTO(time.Millisecond),
	)
	assert.NoError(t, err)
	client.SetRTO(time.Second)
	gotReads := make(chan struct{})
	go func() {
		buf := make([]byte, 1500)
		readN, readErr := connL.Read(buf)
		assert.NoError(t, readErr)
		assert.True(t, IsMessage(buf[:readN]), "should be STUN")
		readN, readErr = connL.Read(buf)
		assert.NoError(t, readErr)
		assert.True(t, IsMessage(buf[:readN]), "should be STUN")
		gotReads <- struct{}{}
	}()
	assert.NoError(t, client.Do(MustBuild(response, BindingRequest), func(event Event) {
		assert.NoError(t, event.Error, "failed")
	}))
	<-gotReads
}

func testClientDoConcurrent(t *testing.T, concurrency int) { //nolint:cyclop
	t.Helper()

	response := MustBuild(TransactionID, BindingSuccess)
	response.Encode()
	connL, connR := net.Pipe()
	defer func() {
		assert.NoError(t, connR.Close())
	}()
	collector := new(manualCollector)
	clock := &manualClock{current: time.Now()}
	agent := &manualAgent{}
	agent.start = func(id [TransactionIDSize]byte, _ time.Time) error {
		go agent.h(Event{
			TransactionID: id,
			Message:       response,
		})

		return nil
	}
	client, err := NewClient(connR,
		WithAgent(agent),
		WithClock(clock),
		WithCollector(collector),
	)
	assert.NoError(t, err)
	client.SetRTO(time.Second)
	conns := new(sync.WaitGroup)
	wg := new(sync.WaitGroup)
	for i := 0; i < concurrency; i++ {
		conns.Add(1)
		go func() {
			defer conns.Done()
			buf := make([]byte, 1500)
			for {
				readN, readErr := connL.Read(buf)
				if readErr != nil {
					if errors.Is(readErr, io.EOF) {
						break
					}
					assert.NoError(t, readErr)
				}
				assert.True(t, IsMessage(buf[:readN]), "should be STUN")
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			assert.NoError(t, client.Do(MustBuild(TransactionID, BindingRequest), func(event Event) {
				assert.NoError(t, event.Error, "failed")
			}))
		}()
	}
	wg.Wait()
	assert.NoError(t, connR.Close())
	conns.Wait()
}

func TestClient_DoConcurrent(t *testing.T) {
	for _, concurrency := range []int{
		1, 5, 10, 25, 100, 500,
	} {
		concurrency := concurrency
		t.Run(fmt.Sprintf("%d", concurrency), func(t *testing.T) {
			testClientDoConcurrent(t, concurrency)
		})
	}
}

type errorCollector struct {
	startError error
	closeError error
}

func (c errorCollector) Start(time.Duration, func(now time.Time)) error {
	return c.startError
}

func (c errorCollector) Close() error { return c.closeError }

func TestNewClient(t *testing.T) {
	t.Run("SetCallbackError", func(t *testing.T) {
		setHandlerError := errClientSetHandler
		_, createErr := NewClient(noopConnection{},
			WithAgent(&errorAgent{
				setHandlerError: setHandlerError,
			}),
		)
		assert.ErrorIs(t, createErr, setHandlerError, "unexpected error returned")
	})
	t.Run("CollectorStartError", func(t *testing.T) {
		startError := errClientStart
		_, createErr := NewClient(noopConnection{},
			WithAgent(&TestAgent{}),
			WithCollector(errorCollector{
				startError: startError,
			}),
		)
		assert.ErrorIs(t, createErr, startError, "unexpected error returned")
	})
}

func TestClient_Close(t *testing.T) {
	t.Run("CollectorCloseError", func(t *testing.T) {
		closeErr := errClientStart
		c, createErr := NewClient(noopConnection{},
			WithCollector(errorCollector{
				closeError: closeErr,
			}),
			WithAgent(&TestAgent{}),
		)
		assert.NoError(t, createErr, "unexpected create error returned")
		gotCloseErr := c.Close()
		assert.ErrorIs(t, gotCloseErr, closeErr, "unexpected close error returned")
	})
}

func TestClientDefaultHandler(t *testing.T) {
	agent := &TestAgent{
		e: make(chan Event),
	}
	id := NewTransactionID()
	handlerCalled := make(chan struct{})
	called := false
	client, createErr := NewClient(noopConnection{},
		WithAgent(agent),
		WithHandler(func(e Event) {
			assert.False(t, called, "should not be called twice")
			called = true
			assert.Equal(t, id, e.TransactionID, "wrong transaction ID")
			handlerCalled <- struct{}{}
		}),
	)
	assert.NoError(t, createErr)
	go func() {
		agent.h(Event{
			TransactionID: id,
		})
	}()
	select {
	case <-handlerCalled:
		// pass
	case <-time.After(time.Millisecond * 100):
		assert.Fail(t, "timed out")
	}
	assert.NoError(t, client.Close())
	// Handler call should be ignored.
	agent.h(Event{})
}

func TestClientClosedStart(t *testing.T) {
	a := &TestAgent{
		e: make(chan Event),
	}
	c, createErr := NewClient(noopConnection{},
		WithAgent(a),
	)
	assert.NoError(t, createErr)
	assert.NoError(t, c.Close())
	assert.ErrorIs(t, c.start(&clientTransaction{}), ErrClientClosed)
}

func TestWithNoRetransmit(t *testing.T) {
	response := MustBuild(TransactionID, BindingSuccess)
	response.Encode()
	connL, connR := net.Pipe()
	defer func() {
		assert.NoError(t, connL.Close())
	}()
	collector := new(manualCollector)
	clock := &manualClock{current: time.Now()}
	agent := &manualAgent{}
	attempt := 0
	agent.start = func(id [TransactionIDSize]byte, _ time.Time) error {
		if attempt == 0 {
			attempt++
			go agent.h(Event{
				TransactionID: id,
				Error:         ErrTransactionTimeOut,
			})
		} else {
			assert.Fail(t, "there should be no second attempt")
			go agent.h(Event{
				TransactionID: id,
				Error:         ErrTransactionTimeOut,
			})
		}

		return nil
	}
	client, err := NewClient(connR,
		WithAgent(agent),
		WithClock(clock),
		WithCollector(collector),
		WithRTO(0),
		WithNoRetransmit,
	)
	assert.NoError(t, err)
	gotReads := make(chan struct{})
	go func() {
		buf := make([]byte, 1500)
		readN, readErr := connL.Read(buf)
		assert.NoError(t, readErr)
		assert.True(t, IsMessage(buf[:readN]), "should be STUN")
		gotReads <- struct{}{}
	}()
	assert.NoError(t, client.Do(MustBuild(response, BindingRequest), func(event Event) {
		assert.ErrorIs(t, event.Error, ErrTransactionTimeOut, "unexpected error")
	}))
	<-gotReads
}

type callbackClock func() time.Time

func (c callbackClock) Now() time.Time {
	return c()
}

func TestClientRTOStartErr(t *testing.T) { //nolint:cyclop
	response := MustBuild(TransactionID, BindingSuccess)
	response.Encode()
	connL, connR := net.Pipe()
	defer func() {
		assert.NoError(t, connL.Close())
	}()
	collector := new(manualCollector)
	shouldWait := false
	shouldWaitMux := new(sync.RWMutex)
	clockWait := make(chan struct{})
	clockLocked := make(chan struct{})
	clock := callbackClock(func() time.Time {
		shouldWaitMux.RLock()
		waiting := shouldWait
		t.Log("waiting:", waiting)
		time.Sleep(time.Millisecond * 100)
		shouldWaitMux.RUnlock()
		if waiting {
			t.Log("clock waiting for log ack")
			clockLocked <- struct{}{}
			t.Log("clock waiting for unlock")
			<-clockWait
			t.Log("clock returned after waiting")
		} else {
			t.Log("clock returned")
		}

		return time.Now()
	})
	agent := &manualAgent{}
	attempt := 0
	gotReads := make(chan struct{})
	var (
		client         *Client
		startClientErr error
	)
	agent.start = func(id [TransactionIDSize]byte, _ time.Time) error {
		t.Log("start", attempt)
		if attempt == 0 {
			attempt++
			go agent.h(Event{
				TransactionID: id,
				Error:         ErrTransactionTimeOut,
			})
		} else {
			go func() {
				<-gotReads
				shouldWaitMux.Lock()
				shouldWait = true
				shouldWaitMux.Unlock()
				go agent.h(Event{
					TransactionID: id,
					Error:         ErrTransactionTimeOut,
				})
				t.Log("clock locked")
				<-clockLocked
				t.Log("closing client")
				assert.NoError(t, client.Close())
				t.Log("client closed, unlocking clock")
				clockWait <- struct{}{}
				t.Log("clock unlocked")
			}()
		}

		return nil
	}
	client, startClientErr = NewClient(connR,
		WithAgent(agent),
		WithClock(clock),
		WithCollector(collector),
		WithRTO(time.Millisecond),
	)
	assert.NoError(t, startClientErr)
	go func() {
		buf := make([]byte, 1500)
		readN, readErr := connL.Read(buf)
		assert.NoError(t, readErr)
		assert.True(t, IsMessage(buf[:readN]), "should be STUN")
		readN, readErr = connL.Read(buf)
		assert.NoError(t, readErr)
		assert.True(t, IsMessage(buf[:readN]), "should be STUN")
		gotReads <- struct{}{}
	}()
	t.Log("starting")
	done := make(chan struct{})
	go func() {
		assert.NoError(t, client.Do(MustBuild(response, BindingRequest), func(event Event) {
			assert.ErrorIs(t, event.Error, ErrClientClosed)
		}))
		done <- struct{}{}
	}()
	select {
	case <-done:
		// ok
	case <-time.After(time.Second * 5):
		assert.Fail(t, "timeout")
	}
}

func TestClientRTOWriteErr(t *testing.T) { //nolint:cyclop
	response := MustBuild(TransactionID, BindingSuccess)
	response.Encode()
	connL, connR := net.Pipe()
	defer func() {
		assert.NoError(t, connL.Close())
	}()
	collector := new(manualCollector)
	shouldWait := false
	shouldWaitMux := new(sync.RWMutex)
	clockWait := make(chan struct{})
	clockLocked := make(chan struct{})
	clock := callbackClock(func() time.Time {
		shouldWaitMux.RLock()
		waiting := shouldWait
		t.Log("waiting:", waiting)
		time.Sleep(time.Millisecond * 100)
		shouldWaitMux.RUnlock()
		if waiting {
			t.Log("clock waiting for log ack")
			clockLocked <- struct{}{}
			t.Log("clock waiting for unlock")
			<-clockWait
			t.Log("clock returned after waiting")
		} else {
			t.Log("clock returned")
		}

		return time.Now()
	})
	agent := &manualAgent{}
	attempt := 0
	gotReads := make(chan struct{})
	var (
		client         *Client
		startClientErr error
	)
	agentStopErr := errClientAgentCantStop
	agent.stop = func([TransactionIDSize]byte) error {
		return agentStopErr
	}
	agent.start = func(id [TransactionIDSize]byte, _ time.Time) error {
		t.Log("start", attempt)
		if attempt == 0 {
			attempt++
			go agent.h(Event{
				TransactionID: id,
				Error:         ErrTransactionTimeOut,
			})
		} else {
			go func() {
				<-gotReads
				shouldWaitMux.Lock()
				shouldWait = true
				shouldWaitMux.Unlock()
				go agent.h(Event{
					TransactionID: id,
					Error:         ErrTransactionTimeOut,
				})
				t.Log("clock locked")
				<-clockLocked
				t.Log("closing connection")
				assert.NoError(t, connL.Close())
				t.Log("connection closed, unlocking clock")
				clockWait <- struct{}{}
				t.Log("clock unlocked")
			}()
		}

		return nil
	}
	client, startClientErr = NewClient(connR,
		WithAgent(agent),
		WithClock(clock),
		WithCollector(collector),
		WithRTO(time.Millisecond),
	)
	assert.NoError(t, startClientErr)
	go func() {
		buf := make([]byte, 1500)
		readN, readErr := connL.Read(buf)
		assert.NoError(t, readErr)
		assert.True(t, IsMessage(buf[:readN]), "should be STUN")
		readN, readErr = connL.Read(buf)
		assert.NoError(t, readErr)
		assert.True(t, IsMessage(buf[:readN]), "should be STUN")
		gotReads <- struct{}{}
	}()
	t.Log("starting")
	done := make(chan struct{})
	go func() {
		assert.NoError(t, client.Do(MustBuild(response, BindingRequest), func(event Event) {
			var e StopErr
			assert.ErrorAs(t, event.Error, &e)
			assert.ErrorIs(t, e.Err, agentStopErr, "incorrect agent error")
			assert.ErrorIs(t, e.Cause, io.ErrClosedPipe, "incorrect connection error")
		}))
		done <- struct{}{}
	}()
	select {
	case <-done:
		// ok
	case <-time.After(time.Second * 5):
		assert.Fail(t, "timeout")
	}
}

func TestClientRTOAgentErr(t *testing.T) {
	response := MustBuild(TransactionID, BindingSuccess)
	response.Encode()
	connL, connR := net.Pipe()
	defer func() {
		assert.NoError(t, connL.Close())
	}()
	collector := new(manualCollector)
	clock := callbackClock(time.Now)
	agent := &manualAgent{}
	attempt := 0
	gotReads := make(chan struct{})
	var (
		client         *Client
		startClientErr error
	)
	agentStartErr := errClientStartRefused
	agent.start = func(id [TransactionIDSize]byte, _ time.Time) error {
		t.Log("start", attempt)
		if attempt == 0 {
			attempt++
			go agent.h(Event{
				TransactionID: id,
				Error:         ErrTransactionTimeOut,
			})
		} else {
			return agentStartErr
		}

		return nil
	}
	client, startClientErr = NewClient(connR,
		WithAgent(agent),
		WithClock(clock),
		WithCollector(collector),
		WithRTO(time.Millisecond),
	)
	assert.NoError(t, startClientErr)
	go func() {
		buf := make([]byte, 1500)
		readN, readErr := connL.Read(buf)
		assert.NoError(t, readErr)
		assert.True(t, IsMessage(buf[:readN]), "should be STUN")
		gotReads <- struct{}{}
	}()
	t.Log("starting")
	assert.NoError(t, client.Do(MustBuild(response, BindingRequest), func(event Event) {
		assert.ErrorIs(t, event.Error, agentStartErr)
	}))
	select {
	case <-gotReads:
		// ok
	case <-time.After(time.Second * 5):
		assert.Fail(t, "reads timeout")
	}
}

func TestClient_HandleProcessError(t *testing.T) {
	response := MustBuild(TransactionID, BindingSuccess)
	response.Encode()
	connL, connR := net.Pipe()
	defer func() {
		assert.NoError(t, connL.Close())
	}()
	collector := new(manualCollector)
	clock := callbackClock(time.Now)
	agent := &manualAgent{}
	gotWrites := make(chan struct{})
	processCalled := make(chan struct{}, 1)
	agent.process = func(*Message) error {
		processCalled <- struct{}{}

		return ErrAgentClosed
	}
	client, startClientErr := NewClient(connR,
		WithAgent(agent),
		WithClock(clock),
		WithCollector(collector),
		WithRTO(time.Millisecond),
	)
	assert.NoError(t, startClientErr)
	go func() {
		_, readErr := connL.Write(response.Raw)
		assert.NoError(t, readErr)
		gotWrites <- struct{}{}
	}()
	t.Log("starting")
	select {
	case <-gotWrites:
		// ok
	case <-time.After(time.Second * 5):
		assert.Fail(t, "reads timeout")
	}
	assert.NoError(t, client.Close())
}

func TestClientImmediateTimeout(t *testing.T) {
	response := MustBuild(TransactionID, BindingSuccess)
	connL, connR := net.Pipe()
	defer func() {
		assert.NoError(t, connL.Close())
	}()
	collector := new(manualCollector)
	clock := &manualClock{current: time.Now()}
	rto := time.Second * 1
	agent := &manualAgent{}
	attempt := 0
	agent.start = func(id [TransactionIDSize]byte, deadline time.Time) error {
		if attempt == 0 {
			assert.False(t, deadline.Before(clock.current.Add(rto/2)), "deadline too fast")
			attempt++
			go agent.h(Event{
				TransactionID: id,
				Message:       response,
			})
		} else {
			assert.Fail(t, "there should be no second attempt")
			go agent.h(Event{
				TransactionID: id,
				Error:         ErrTransactionTimeOut,
			})
		}

		return nil
	}
	client, err := NewClient(connR,
		WithAgent(agent),
		WithClock(clock),
		WithCollector(collector),
		WithRTO(rto),
	)
	assert.NoError(t, err)
	gotReads := make(chan struct{})
	go func() {
		buf := make([]byte, 1500)
		readN, readErr := connL.Read(buf)
		assert.NoError(t, readErr)
		assert.True(t, IsMessage(buf[:readN]), "should be STUN")
		gotReads <- struct{}{}
	}()
	client.Start(MustBuild(response, BindingRequest), func(e Event) { //nolint:errcheck,gosec
		assert.NoError(t, e.Error, "unexpected error")
	})
	<-gotReads
}
