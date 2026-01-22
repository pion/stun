// SPDX-FileCopyrightText: 2026 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package stun

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAgent_ProcessInTransaction(t *testing.T) {
	msg := New()
	agent := NewAgent(func(e Event) {
		assert.NoError(t, e.Error, "got error")
		assert.True(t, e.Message.Equal(msg), "%s (got) != %s (expected)", e.Message, msg)
	})
	assert.NoError(t, msg.NewTransactionID())
	assert.NoError(t, agent.Start(msg.TransactionID, time.Time{}))
	assert.NoError(t, agent.Process(msg))
	assert.NoError(t, agent.Close())
}

func TestAgent_Process(t *testing.T) {
	msg := New()
	agent := NewAgent(func(e Event) {
		assert.NoError(t, e.Error, "got error")
		assert.True(t, e.Message.Equal(msg), "%s (got) != %s (expected)", e.Message, msg)
	})
	assert.NoError(t, msg.NewTransactionID())
	assert.NoError(t, agent.Process(msg))
	assert.NoError(t, agent.Close())
	assert.ErrorIs(t, agent.Process(msg), ErrAgentClosed)
}

func TestAgent_Start(t *testing.T) {
	agent := NewAgent(nil)
	id := NewTransactionID()
	deadline := time.Now().AddDate(0, 0, 1)
	assert.NoError(t, agent.Start(id, deadline), "failed to start transaction")
	assert.ErrorIs(t, agent.Start(id, deadline), ErrTransactionExists)
	assert.NoError(t, agent.Close())
	id = NewTransactionID()
	assert.ErrorIs(t, agent.Start(id, deadline), ErrAgentClosed)
	assert.ErrorIs(t, agent.SetHandler(nil), ErrAgentClosed)
}

func TestAgent_Stop(t *testing.T) {
	called := make(chan Event, 1)
	agent := NewAgent(func(e Event) {
		called <- e
	})
	assert.ErrorIs(t, agent.Stop(transactionID{}), ErrTransactionNotExists)
	id := NewTransactionID()
	timeout := time.Millisecond * 200
	assert.NoError(t, agent.Start(id, time.Now().Add(timeout)))
	assert.NoError(t, agent.Stop(id))
	select {
	case e := <-called:
		assert.ErrorIs(t, e.Error, ErrTransactionStopped)
	case <-time.After(timeout * 2):
		assert.Fail(t, "timed out")
	}
	assert.NoError(t, agent.Close())
	assert.ErrorIs(t, agent.Close(), ErrAgentClosed)
	assert.ErrorIs(t, agent.Stop(transactionID{}), ErrAgentClosed)
}

func TestAgent_GC(t *testing.T) { //nolint:cyclop
	agent := NewAgent(nil)
	shouldTimeOutID := make(map[transactionID]bool)
	deadline := time.Date(2027, time.November, 21,
		23, 0, 0, 0,
		time.UTC,
	)
	gcDeadline := deadline.Add(-time.Second)
	deadlineNotGC := gcDeadline.AddDate(0, 0, -1)
	agent.SetHandler(func(e Event) { //nolint:errcheck,gosec
		id := e.TransactionID
		shouldTimeOut, found := shouldTimeOutID[id]
		assert.True(t, found, "unexpected transaction ID")
		if shouldTimeOut {
			assert.ErrorIs(t, e.Error, ErrTransactionTimeOut, "%x should time out", id)
		} else {
			assert.False(t, errors.Is(e.Error, ErrTransactionTimeOut), "%x should not time out", id)
		}
	})
	for i := 0; i < 5; i++ {
		id := NewTransactionID()
		shouldTimeOutID[id] = false
		assert.NoError(t, agent.Start(id, deadline))
	}
	for i := 0; i < 5; i++ {
		id := NewTransactionID()
		shouldTimeOutID[id] = true
		assert.NoError(t, agent.Start(id, deadlineNotGC))
	}
	assert.NoError(t, agent.Collect(gcDeadline))
	assert.NoError(t, agent.Close())
	assert.ErrorIs(t, agent.Collect(gcDeadline), ErrAgentClosed)
}

func BenchmarkAgent_GC(b *testing.B) {
	agent := NewAgent(nil)
	deadline := time.Now().AddDate(0, 0, 1)
	for i := 0; i < agentCollectCap; i++ {
		assert.NoError(b, agent.Start(NewTransactionID(), deadline))
	}
	defer func() {
		assert.NoError(b, agent.Close())
	}()
	b.ReportAllocs()
	gcDeadline := deadline.Add(-time.Second)
	for i := 0; i < b.N; i++ {
		assert.NoError(b, agent.Collect(gcDeadline))
	}
}

func BenchmarkAgent_Process(b *testing.B) {
	agent := NewAgent(nil)
	deadline := time.Now().AddDate(0, 0, 1)
	for i := 0; i < 1000; i++ {
		assert.NoError(b, agent.Start(NewTransactionID(), deadline))
	}
	defer func() {
		assert.NoError(b, agent.Close())
	}()
	b.ReportAllocs()
	m := MustBuild(TransactionID)
	for i := 0; i < b.N; i++ {
		assert.NoError(b, agent.Process(m))
	}
}
