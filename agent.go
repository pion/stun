package stun

import (
	"errors"
	"net"
	"sync"
	"time"
)

// AgentOptions are required to initialize Agent.
type AgentOptions struct {
	Handler AgentFn // Default handler, can be nil.
}

// NewAgent initializes and returns new Agent from options.
func NewAgent(o AgentOptions) *Agent {
	a := &Agent{
		transactions: make(map[transactionID]agentTransaction),
		zeroHandler:  o.Handler,
	}
	return a
}

// Agent is low-level abstraction over transactions.
type Agent struct {
	transactions map[transactionID]agentTransaction
	closed       bool
	mux          sync.Mutex // protects transactions and closed
	zeroHandler  AgentFn    // handles non-registered transactions if set
}

// AgentFn is called on transaction state change.
// Usage of e is valid only during call, user must
// copy needed fields explicitly.
type AgentFn func(e AgentEvent)

// AgentEvent is set of arguments passed to AgentFn, describing
// an transaction event.
type AgentEvent struct {
	RAddr   net.Addr
	LAddr   net.Addr
	Message *Message
	Error   error
}

type agentTransaction struct {
	id       transactionID
	deadline time.Time
	f        AgentFn
}

var (
	// ErrTransactionStopped indicates that transaction was manually stopped.
	ErrTransactionStopped = errors.New("transaction is stopped")
	// ErrTransactionNotExists indicates that agent failed to find transaction.
	ErrTransactionNotExists = errors.New("transaction not exists")
	// ErrTransactionExists indicates that transaction with same id is already
	// registered.
	ErrTransactionExists = errors.New("transaction exists with same id")
)

// Stop stops transaction by id with ErrTransactionStopped.
func (a *Agent) Stop(id transactionID) error {
	a.mux.Lock()
	if a.closed {
		a.mux.Unlock()
		return ErrAgentClosed
	}
	t, exists := a.transactions[id]
	delete(a.transactions, id)
	a.mux.Unlock()
	if !exists {
		return ErrTransactionNotExists
	}
	t.f(AgentEvent{
		Error: ErrTransactionStopped,
	})
	return nil
}

// ErrAgentClosed indicates that agent is in closed state and is unable
// to handle transactions.
var ErrAgentClosed = errors.New("agent is closed")

// Start registers transaction with provided id, deadline and callback.
// Could return ErrAgentClosed, ErrTransactionExists.
// Callback f is guaranteed to be eventually called. See AgentFn for
// callback processing constraints.
func (a *Agent) Start(id transactionID, deadline time.Time, f AgentFn) error {
	a.mux.Lock()
	defer a.mux.Unlock()
	if a.closed {
		return ErrAgentClosed
	}
	_, exists := a.transactions[id]
	if exists {
		return ErrTransactionExists
	}
	a.transactions[id] = agentTransaction{
		id:       id,
		f:        f,
		deadline: deadline,
	}
	return nil
}

// agentGCInitCap is initial capacity for Agent.garbageCollect slices,
// sufficient to make function zero-alloc in most cases.
const agentGCInitCap = 100

// ErrTransactionTimeOut indicates that transaction has reached deadline.
var ErrTransactionTimeOut = errors.New("transaction is timed out")

// garbageCollect terminates all timed out transactions.
func (a *Agent) garbageCollect(gcTime time.Time) {
	toCall := make([]AgentFn, 0, agentGCInitCap)
	toRemove := make([]transactionID, 0, agentGCInitCap)
	a.mux.Lock()
	if a.closed {
		// Doing nothing if agent is closed.
		// All transactions should be already closed
		// during Close() call.
		a.mux.Unlock()
		return
	}
	// Adding all transactions with deadline before gcTime
	// to toCall and toRemove slices.
	// No allocs if there are less than agentGCInitCap
	// timed out transactions.
	for id, t := range a.transactions {
		if t.deadline.Before(gcTime) {
			toRemove = append(toRemove, id)
			toCall = append(toCall, t.f)
		}
	}
	// Un-registering timed out transactions.
	for _, id := range toRemove {
		delete(a.transactions, id)
	}
	// Calling callbacks does not require locked mutex,
	// reducing lock time.
	a.mux.Unlock()
	// Sending ErrTransactionTimeOut to all callbacks, blocking
	// garbageCollect until last one.
	event := AgentEvent{
		Error: ErrTransactionTimeOut,
	}
	for _, f := range toCall {
		f(event)
	}
}

// AgentProcessArgs is set of arguments passed to Agent.Process.
type AgentProcessArgs struct {
	Message *Message
}

// Process incoming message.
// Blocks until handler returns.
func (a *Agent) Process(args AgentProcessArgs) error {
	m := args.Message
	a.mux.Lock()
	if a.closed {
		a.mux.Unlock()
		return ErrAgentClosed
	}
	t, ok := a.transactions[m.TransactionID]
	delete(a.transactions, m.TransactionID)
	a.mux.Unlock()
	event := AgentEvent{
		Message: m,
	}
	if ok {
		t.f(event)
	} else if a.zeroHandler != nil {
		a.zeroHandler(event)
	}
	return nil
}

// Close terminated all transactions with ErrAgentClosed and renders Agent to
// closed state.
func (a *Agent) Close() error {
	e := AgentEvent{
		Error: ErrAgentClosed,
	}
	a.mux.Lock()
	for _, t := range a.transactions {
		t.f(e)
	}
	a.transactions = nil
	a.closed = true
	a.mux.Unlock()
	return nil
}

type transactionID [transactionIDSize]byte
