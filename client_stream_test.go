// SPDX-FileCopyrightText: 2026 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

//go:build !js

package stun

import (
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"

	"github.com/pion/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeResponseWithData builds an encoded Binding success response carrying
// a DATA attribute of dataLen bytes, so the whole message exceeds the
// client's 1024-byte read buffer.
func makeResponseWithData(t *testing.T, dataLen int) (*Message, []byte) {
	t.Helper()
	m := MustBuild(TransactionID, BindingSuccess)
	m.Add(AttrData, make([]byte, dataLen))

	return m, append([]byte{}, m.Raw...)
}

// startResponseServer accepts a single TCP connection, reads requestBytes
// (so client transactions are registered before the response arrives) and
// then writes the response, keeping the connection open briefly before
// closing it.
func startResponseServer(t *testing.T, requestBytes int, write func(net.Conn)) net.Addr {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0") //nolint:noctx
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, ln.Close())
	})
	go func() {
		conn, aErr := ln.Accept()
		if aErr != nil {
			return
		}
		if _, rErr := io.ReadFull(conn, make([]byte, requestBytes)); rErr != nil {
			return
		}
		write(conn)
		time.Sleep(100 * time.Millisecond)
		_ = conn.Close()
	}()

	return ln.Addr()
}

func newTCPClient(t *testing.T, addr net.Addr) *Client {
	t.Helper()
	conn, err := net.Dial("tcp", addr.String()) //nolint:noctx
	assert.NoError(t, err)
	client, err := NewClient(conn)
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, client.Close())
	})

	return client
}

// TestClientReadLargeMessageStream verifies that a STUN message larger than
// the 1024-byte read buffer is delivered intact over a TCP connection.
func TestClientReadLargeMessageStream(t *testing.T) {
	resp, raw := makeResponseWithData(t, 2000)
	addr := startResponseServer(t, messageHeaderSize, func(conn net.Conn) {
		_, _ = conn.Write(raw)
	})
	client := newTCPClient(t, addr)

	done := make(chan Event, 1)
	req := MustBuild(NewTransactionIDSetter(resp.TransactionID), BindingRequest)
	assert.NoError(t, client.Start(req, func(e Event) {
		done <- e
	}))

	select {
	case e := <-done:
		require.NoError(t, e.Error)
		require.NotNil(t, e.Message)
		data, err := e.Message.Get(AttrData)
		require.NoError(t, err)
		assert.Len(t, data, 2000, "large message must not be truncated")
	case <-time.After(time.Second):
		assert.FailNow(t, "large STUN message over TCP was never delivered")
	}
}

// TestClientReadFragmentedMessageStream verifies that a STUN message split
// across multiple TCP reads is reassembled instead of desynchronizing the
// stream.
func TestClientReadFragmentedMessageStream(t *testing.T) {
	resp, raw := makeResponseWithData(t, 2000)
	addr := startResponseServer(t, messageHeaderSize, func(conn net.Conn) {
		// Send the first half, wait so the client reads a partial message,
		// then send the rest.
		_, _ = conn.Write(raw[:len(raw)/2])
		time.Sleep(50 * time.Millisecond)
		_, _ = conn.Write(raw[len(raw)/2:])
	})
	client := newTCPClient(t, addr)

	done := make(chan Event, 1)
	req := MustBuild(NewTransactionIDSetter(resp.TransactionID), BindingRequest)
	assert.NoError(t, client.Start(req, func(e Event) {
		done <- e
	}))

	select {
	case e := <-done:
		require.NoError(t, e.Error)
		require.NotNil(t, e.Message)
		data, err := e.Message.Get(AttrData)
		require.NoError(t, err)
		assert.Len(t, data, 2000, "fragmented message must be reassembled intact")
	case <-time.After(time.Second):
		assert.FailNow(t, "fragmented STUN message over TCP was never delivered")
	}
}

// TestClientReadCoalescedMessagesStream verifies that two STUN messages
// arriving in a single TCP read are both delivered to their transactions.
func TestClientReadCoalescedMessagesStream(t *testing.T) {
	resp1, raw1 := makeResponseWithData(t, 100)
	resp2, raw2 := makeResponseWithData(t, 100)
	addr := startResponseServer(t, 2*messageHeaderSize, func(conn net.Conn) {
		_, _ = conn.Write(append(raw1, raw2...))
	})
	client := newTCPClient(t, addr)

	// The read loop reuses the Message buffer, so each callback must copy
	// the data it needs before returning.
	type result struct {
		err error
		raw []byte
	}
	done := make(chan result, 2)
	req1 := MustBuild(NewTransactionIDSetter(resp1.TransactionID), BindingRequest)
	req2 := MustBuild(NewTransactionIDSetter(resp2.TransactionID), BindingRequest)
	handler := func(e Event) {
		var raw []byte
		if e.Message != nil {
			raw = append([]byte{}, e.Message.Raw...)
		}
		done <- result{err: e.Error, raw: raw}
	}
	assert.NoError(t, client.Start(req1, handler))
	assert.NoError(t, client.Start(req2, handler))

	for range 2 {
		select {
		case r := <-done:
			require.NoError(t, r.err)
			m := &Message{}
			require.NoError(t, Decode(r.raw, m))
			_, err := m.Get(AttrData)
			require.NoError(t, err)
		case <-time.After(time.Second):
			assert.FailNow(t, "coalesced STUN messages over TCP were not both delivered")
		}
	}
}

// TestClientReadLargeMessageDatagram verifies that a large STUN message on a
// datagram-style connection is not truncated by the read buffer.
func TestClientReadLargeMessageDatagram(t *testing.T) {
	resp, raw := makeResponseWithData(t, 2000)
	server, err := net.ListenPacket("udp", "127.0.0.1:0") //nolint:noctx
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, server.Close())
	})
	go func() {
		buf := make([]byte, 1500)
		n, addr, rErr := server.ReadFrom(buf)
		if rErr != nil || n <= 0 {
			return
		}
		_, _ = server.WriteTo(raw, addr)
	}()
	conn, err := net.Dial("udp", server.LocalAddr().String()) //nolint:noctx
	require.NoError(t, err)
	client, err := NewClient(conn)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, client.Close())
	})

	done := make(chan Event, 1)
	req := MustBuild(NewTransactionIDSetter(resp.TransactionID), BindingRequest)
	require.NoError(t, client.Start(req, func(e Event) {
		done <- e
	}))

	select {
	case e := <-done:
		require.NoError(t, e.Error)
		require.NotNil(t, e.Message)
		data, err := e.Message.Get(AttrData)
		require.NoError(t, err)
		assert.Len(t, data, 2000, "datagram message must not be truncated")
	case <-time.After(time.Second):
		assert.FailNow(t, "large STUN message on datagram connection was never delivered")
	}
}

// TestClientReadStrictTypeZeroStream verifies that a stream message with a
// reserved message type 0 is rejected in strict mode and the in-flight
// transaction times out.
func TestClientReadStrictTypeZeroStream(t *testing.T) {
	// A 20-byte message with type 0 (reserved), a valid magic cookie and
	// zero attributes.
	raw := make([]byte, messageHeaderSize)
	binary.BigEndian.PutUint16(raw[0:2], 0)
	binary.BigEndian.PutUint32(raw[4:8], magicCookie)
	addr := startResponseServer(t, messageHeaderSize, func(conn net.Conn) {
		_, _ = conn.Write(raw)
	})
	conn, err := net.Dial("tcp", addr.String()) //nolint:noctx
	require.NoError(t, err)
	client, err := NewClient(conn,
		WithStrictMode(true),
		WithLoggerFactory(logging.NewDefaultLoggerFactory()),
		WithRTO(100*time.Millisecond),
	)
	require.NoError(t, err)
	WithNoRetransmit(client)
	t.Cleanup(func() {
		assert.NoError(t, client.Close())
	})

	done := make(chan Event, 1)
	req := MustBuild(TransactionID, BindingRequest)
	require.NoError(t, client.Start(req, func(e Event) {
		done <- e
	}))

	select {
	case e := <-done:
		require.ErrorIs(t, e.Error, ErrTransactionTimeOut,
			"reserved message type must be rejected, leaving the transaction to time out")
	case <-time.After(time.Second):
		assert.FailNow(t, "transaction was not terminated after reserved type message")
	}
}

// streamReaderConnection is a Connection whose LocalAddr reports a TCP
// address, so the client treats it as a stream. Each Read call returns the
// next chunk followed by the configured error.
type streamReaderConnection struct {
	chunks [][]byte
	readAt int
	err    error
}

func (c *streamReaderConnection) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1}
}

func (c *streamReaderConnection) Read(b []byte) (int, error) {
	if c.readAt < len(c.chunks) {
		n := copy(b, c.chunks[c.readAt])
		c.readAt++

		return n, c.err
	}

	return 0, io.EOF
}

func (c *streamReaderConnection) Write(b []byte) (int, error) {
	return len(b), nil
}

func (c *streamReaderConnection) Close() error {
	return nil
}

// TestClientReadFinalMessageOnEOFStream verifies that a complete message
// delivered together with io.EOF is still handed to the transaction.
func TestClientReadFinalMessageOnEOFStream(t *testing.T) {
	resp, raw := makeResponseWithData(t, 100)
	conn := &streamReaderConnection{
		chunks: [][]byte{raw},
		err:    io.EOF,
	}
	client, err := NewClient(conn)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, client.Close())
	})

	done := make(chan Event, 1)
	req := MustBuild(NewTransactionIDSetter(resp.TransactionID), BindingRequest)
	require.NoError(t, client.Start(req, func(e Event) {
		done <- e
	}))

	select {
	case e := <-done:
		require.NoError(t, e.Error)
		require.NotNil(t, e.Message)
	case <-time.After(time.Second):
		assert.FailNow(t, "message delivered together with EOF was not processed")
	}
}

// TestClientReadMalformedStream verifies that garbage on a stream stops the
// read loop, leaving the in-flight transaction to time out.
func TestClientReadMalformedStream(t *testing.T) {
	// A 20-byte "message" with an invalid magic cookie.
	raw := make([]byte, messageHeaderSize)
	binary.BigEndian.PutUint32(raw[4:8], 0xdeadbeef)
	conn := &streamReaderConnection{
		chunks: [][]byte{raw},
	}
	client, err := NewClient(conn,
		WithLoggerFactory(logging.NewDefaultLoggerFactory()),
		WithRTO(100*time.Millisecond),
	)
	require.NoError(t, err)
	WithNoRetransmit(client)
	t.Cleanup(func() {
		assert.NoError(t, client.Close())
	})

	done := make(chan Event, 1)
	req := MustBuild(TransactionID, BindingRequest)
	require.NoError(t, client.Start(req, func(e Event) {
		done <- e
	}))

	select {
	case e := <-done:
		require.ErrorIs(t, e.Error, ErrTransactionTimeOut)
	case <-time.After(time.Second):
		assert.FailNow(t, "transaction was not terminated after malformed stream data")
	}
}

// nilAddrConnection reports a nil local address, so the client must fall
// back to datagram handling without crashing.
type nilAddrConnection struct {
	testConnection
}

func (c *nilAddrConnection) LocalAddr() net.Addr {
	return nil
}

// TestClientReadNilLocalAddr verifies that a connection with a nil local
// address is handled like a datagram connection.
func TestClientReadNilLocalAddr(t *testing.T) {
	resp, raw := makeResponseWithData(t, 100)
	conn := &nilAddrConnection{
		testConnection: testConnection{
			b: raw,
			write: func(b []byte) (int, error) {
				return len(b), nil
			},
		},
	}
	client, err := NewClient(conn)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, client.Close())
	})

	done := make(chan Event, 1)
	req := MustBuild(NewTransactionIDSetter(resp.TransactionID), BindingRequest)
	require.NoError(t, client.Start(req, func(e Event) {
		done <- e
	}))

	select {
	case e := <-done:
		require.NoError(t, e.Error)
		require.NotNil(t, e.Message)
	case <-time.After(time.Second):
		assert.FailNow(t, "datagram fallback for nil local address was not delivered")
	}
}
