package stun

import (
	"testing"
	"time"
)

func TestAgent_ProcessInTransaction(t *testing.T) {
	m := New()
	a := NewAgent(AgentOptions{
		Handler: func(e AgentEvent) {
			t.Error("should not be called")
		},
	})
	if err := m.NewTransactionID(); err != nil {
		t.Fatal(err)
	}
	if err := a.Start(m.TransactionID, time.Time{}, func(e AgentEvent) {
		if e.Error != nil {
			t.Errorf("got error: %s", e.Error)
		}
		if !e.Message.Equal(m) {
			t.Errorf("%s (got) != %s (expected)", e.Message, m)
		}

	}); err != nil {
		t.Fatal(err)
	}
	if err := a.Process(m); err != nil {
		t.Error(err)
	}
	if err := a.Close(); err != nil {
		t.Error(err)
	}
}

func TestAgent_Process(t *testing.T) {
	m := New()
	a := NewAgent(AgentOptions{
		Handler: func(e AgentEvent) {
			if e.Error != nil {
				t.Errorf("got error: %s", e.Error)
			}
			if !e.Message.Equal(m) {
				t.Errorf("%s (got) != %s (expected)", e.Message, m)
			}
		},
	})
	if err := m.NewTransactionID(); err != nil {
		t.Fatal(err)
	}
	if err := a.Process(m); err != nil {
		t.Error(err)
	}
	if err := a.Close(); err != nil {
		t.Error(err)
	}
	if err := a.Process(m); err != ErrAgentClosed {
		t.Errorf("closed agent should return <%s>, but got <%s>",
			ErrAgentClosed, err,
		)
	}
}

func TestAgent_Start(t *testing.T) {
	a := NewAgent(AgentOptions{})
	id := NewTransactionID()
	deadline := time.Now().AddDate(0, 0, 1)
	if err := a.Start(id, deadline, noopHandler); err != nil {
		t.Errorf("failed to statt transaction: %s", err)
	}
	if err := a.Start(id, deadline, noopHandler); err != ErrTransactionExists {
		t.Errorf("duplicate start should return <%s>, got <%s>",
			ErrTransactionExists, err,
		)
	}
	if err := a.Close(); err != nil {
		t.Error(err)
	}
	id = NewTransactionID()
	if err := a.Start(id, deadline, noopHandler); err != ErrAgentClosed {
		t.Errorf("start on closed agent should return <%s>, got <%s>",
			ErrAgentClosed, err,
		)
	}
}

func TestAgent_Stop(t *testing.T) {
	a := NewAgent(AgentOptions{})
	if err := a.Stop(transactionID{}); err != ErrTransactionNotExists {
		t.Fatalf("unexpected error: %s, should be %s", err, ErrTransactionNotExists)
	}
	id := NewTransactionID()
	called := make(chan AgentEvent, 1)
	timeout := time.Millisecond * 200
	if err := a.Start(id, time.Now().Add(timeout), func(e AgentEvent) {
		called <- e
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.Stop(id); err != nil {
		t.Fatal(err)
	}
	select {
	case e := <-called:
		if e.Error != ErrTransactionStopped {
			t.Fatalf("unexpected error: %s, should be %s",
				e.Error, ErrTransactionStopped,
			)
		}
	case <-time.After(timeout * 2):
		t.Fatal("timed out")
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	if err := a.Stop(transactionID{}); err != ErrAgentClosed {
		t.Fatalf("unexpected error: %s, should be %s", err, ErrAgentClosed)
	}
}

var noopHandler = func(e AgentEvent) {}

func TestAgent_GC(t *testing.T) {
	a := NewAgent(AgentOptions{
		Handler: noopHandler,
	})
	shouldTimeOut := func(e AgentEvent) {
		if e.Error != ErrTransactionTimeOut {
			t.Errorf("should time out, but got <%s>", e.Error)
		}
	}
	shouldNotTimeOut := func(e AgentEvent) {
		if e.Error == ErrTransactionTimeOut {
			t.Error("should not time out")
		}
	}
	deadline := time.Date(2027, time.November, 21,
		23, 13, 34, 120021,
		time.UTC,
	)
	gcDeadline := deadline.Add(time.Second)
	deadlineNotGC := gcDeadline.AddDate(0, 0, 1)
	for i := 0; i < 5; i++ {
		if err := a.Start(NewTransactionID(), deadline, shouldTimeOut); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 5; i++ {
		if err := a.Start(NewTransactionID(), deadlineNotGC, shouldNotTimeOut); err != nil {
			t.Fatal(err)
		}
	}
	if err := a.Collect(gcDeadline); err != nil {
		t.Fatal(err)
	}
	if err := a.Close(); err != nil {
		t.Error(err)
	}
	if err := a.Collect(gcDeadline); err != ErrAgentClosed {
		t.Errorf("should <%s>, but got <%s>", ErrAgentClosed, err)
	}
}

func BenchmarkAgent_GC(b *testing.B) {
	a := NewAgent(AgentOptions{
		Handler: noopHandler,
	})
	deadline := time.Now().AddDate(0, 0, 1)
	for i := 0; i < agentCollectCap; i++ {
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
	gcDeadline := deadline.Add(-time.Second)
	for i := 0; i < b.N; i++ {
		if err := a.Collect(gcDeadline); err != nil {
			b.Fatal(err)
		}
	}
}

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
	m := MustBuild(TransactionID)
	for i := 0; i < b.N; i++ {
		if err := a.Process(m); err != nil {
			b.Fatal(err)
		}
	}
}
