// SPDX-FileCopyrightText: 2026 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

//go:build !js

package stun

import (
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pion/logging"
	"github.com/stretchr/testify/assert"
)

// countingReadConn counts Read calls and always fails with io.EOF,
// simulating a peer that closed the connection.
type countingReadConn struct {
	reads atomic.Int64
}

func (c *countingReadConn) Read([]byte) (int, error) {
	c.reads.Add(1)

	return 0, io.EOF
}

func (c *countingReadConn) Write(b []byte) (int, error) {
	return len(b), nil
}

func (c *countingReadConn) Close() error {
	return nil
}

// TestReadLoopExitsOnEOF verifies that the client read loop exits after a
// permanent read error instead of busy-spinning on a failing connection.
//
// Regression test: previously the read loop ignored all read errors, so a
// closed connection caused the goroutine to spin at 100% CPU calling Read
// millions of times until the client was closed.
func TestReadLoopExitsOnEOF(t *testing.T) {
	for _, loggerFactory := range []logging.LoggerFactory{nil, logging.NewDefaultLoggerFactory()} {
		conn := &countingReadConn{}
		client, err := NewClient(conn, WithLoggerFactory(loggerFactory))
		assert.NoError(t, err)

		// Give the read loop time to consume the EOF.
		time.Sleep(100 * time.Millisecond)
		first := conn.reads.Load()

		// After the permanent read error the loop must have exited, so the
		// number of Read calls must not keep growing.
		time.Sleep(100 * time.Millisecond)
		second := conn.reads.Load()

		assert.NoError(t, client.Close())
		t.Logf("Read calls (logger=%v): first window=%d, second window=%d",
			loggerFactory != nil, first, second-first)
		assert.LessOrEqual(t, second, first+1,
			"read loop busy-spins after permanent read error (Read called %d more times)",
			second-first)
	}
}
