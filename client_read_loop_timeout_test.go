// SPDX-FileCopyrightText: 2026 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

//go:build !js

package stun

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestReadLoopExitTimesOutTransactions verifies that after the read loop
// exits due to a connection error, in-flight transactions are still
// terminated by the timeout collector (the client does not go silent).
func TestReadLoopExitTimesOutTransactions(t *testing.T) {
	conn := &countingReadConn{}
	client, err := NewClient(conn,
		WithRTO(time.Millisecond*20),
		WithTimeoutRate(time.Millisecond*5),
	)
	assert.NoError(t, err)
	defer func() {
		assert.NoError(t, client.Close())
	}()

	m := MustBuild(TransactionID, BindingRequest)
	m.Encode()
	done := make(chan Event, 1)
	assert.NoError(t, client.Start(m, func(e Event) {
		done <- e
	}))

	select {
	case e := <-done:
		assert.ErrorIs(t, e.Error, ErrTransactionTimeOut,
			"transaction should time out after read loop exits")
	case <-time.After(time.Second):
		assert.FailNow(t, "transaction never timed out after connection failure")
	}

	// Read loop must have exited (one Read call consumed the EOF).
	assert.Equal(t, int64(1), conn.reads.Load())
}
