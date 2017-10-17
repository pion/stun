package stun

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// Dial connects to the address on the named network and then
// initializes Client on that connection, returning error if any.
func Dial(network, address string) (*Client, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return NewClient(ClientOptions{
		Connection: conn,
	}), nil
}

// ClientOptions are used to initialize Client.
type ClientOptions struct {
	Agent       ClientAgent
	Connection  Connection
	TimeoutRate time.Duration // defaults to 100 ms
}

const defaultTimeoutRate = time.Millisecond * 100

// NewClient initializes new Client from provided options,
// starting internal goroutines and using default options fields
// if necessary. Call Close method after using Client to release
// resources.
func NewClient(options ClientOptions) *Client {
	c := &Client{
		close:  make(chan struct{}),
		c:      options.Connection,
		a:      options.Agent,
		gcRate: options.TimeoutRate,
	}
	if c.a == nil {
		c.a = NewAgent(AgentOptions{})
	}
	if c.gcRate == 0 {
		c.gcRate = defaultTimeoutRate
	}
	c.wg.Add(2)
	go c.readUntilClosed()
	go c.collectUntilClosed()
	return c
}

// Connection wraps Reader, Writer and Closer interfaces.
type Connection interface {
	io.Reader
	io.Writer
	io.Closer
}

// ClientAgent is Agent implementation that is used by Client to
// process transactions.
type ClientAgent interface {
	Process(*Message) error
	Close() error
	Start(id [TransactionIDSize]byte, deadline time.Time, f AgentFn) error
	Stop(id [TransactionIDSize]byte) error
	Collect(time.Time) error
}

// Client simulates "connection" to STUN server.
type Client struct {
	a         ClientAgent
	c         Connection
	close     chan struct{}
	closed    bool
	closedMux sync.RWMutex
	gcRate    time.Duration
	wg        sync.WaitGroup
}

// StopErr occurs when Client fails to stop transaction while
// processing error.
type StopErr struct {
	Err   error // value returned by Stop()
	Cause error // error that caused Stop() call
}

func (e StopErr) Error() string {
	return fmt.Sprintf("error while stopping due to %s: %s",
		sprintErr(e.Cause), sprintErr(e.Err),
	)
}

// CloseErr indicates client close failure.
type CloseErr struct {
	AgentErr      error
	ConnectionErr error
}

func sprintErr(err error) string {
	if err == nil {
		return "<nil>"
	}
	return err.Error()
}

func (c CloseErr) Error() string {
	return fmt.Sprintf("failed to close: %s (connection), %s (agent)",
		sprintErr(c.ConnectionErr), sprintErr(c.AgentErr),
	)
}

func (c *Client) readUntilClosed() {
	defer c.wg.Done()
	m := new(Message)
	m.Raw = make([]byte, 1024)
	for {
		select {
		case <-c.close:
			return
		default:
		}
		_, err := m.ReadFrom(c.c)
		if err == nil {
			if pErr := c.a.Process(m); pErr == ErrAgentClosed {
				return
			}
		}
	}
}

func closedOrPanic(err error) {
	if err == nil || err == ErrAgentClosed {
		return
	}
	panic(err)
}

func (c *Client) collectUntilClosed() {
	t := time.NewTicker(c.gcRate)
	defer c.wg.Done()
	for {
		select {
		case <-c.close:
			t.Stop()
			return
		case gcTime := <-t.C:
			closedOrPanic(c.a.Collect(gcTime))
		}
	}
}

// ErrClientClosed indicates that client is closed.
var ErrClientClosed = errors.New("client is closed")

// Close stops internal connection and agent, returning CloseErr on error.
func (c *Client) Close() error {
	c.closedMux.Lock()
	if c.closed {
		c.closedMux.Unlock()
		return ErrClientClosed
	}
	c.closed = true
	c.closedMux.Unlock()
	agentErr := c.a.Close()
	connErr := c.c.Close()
	close(c.close)
	c.wg.Wait()
	if agentErr == nil && connErr == nil {
		return nil
	}
	return CloseErr{
		AgentErr:      agentErr,
		ConnectionErr: connErr,
	}
}

// Indicate sends indication m to server. Shorthand to Start call
// with zero deadline and callback.
func (c *Client) Indicate(m *Message) error {
	return c.Start(m, time.Time{}, nil)
}

// Do is Start wrapper that waits until callback is called. If no callback
// provided, Indicate is called instead.
//
// Do has memory allocation overhead due to blocking, see BenchmarkClient_Do.
// Use Start for zero overhead.
func (c *Client) Do(m *Message, d time.Time, f func(AgentEvent)) error {
	if f == nil {
		return c.Indicate(m)
	}
	cond := sync.NewCond(new(sync.Mutex))
	processed := false
	wrapper := func(e AgentEvent) {
		f(e)
		cond.L.Lock()
		processed = true
		cond.Broadcast()
		cond.L.Unlock()
	}
	if err := c.Start(m, d, wrapper); err != nil {
		return err
	}
	cond.L.Lock()
	for !processed {
		cond.Wait()
	}
	cond.L.Unlock()
	return nil
}

// Start starts transaction (if f set) and writes message to server, callback
// is called asynchronously.
func (c *Client) Start(m *Message, d time.Time, f func(AgentEvent)) error {
	c.closedMux.RLock()
	closed := c.closed
	c.closedMux.RUnlock()
	if closed {
		return ErrClientClosed
	}
	if f != nil {
		// Starting transaction only if f is set. Useful for indications.
		if err := c.a.Start(m.TransactionID, d, f); err != nil {
			return err
		}
	}
	_, err := m.WriteTo(c.c)
	if err != nil && f != nil {
		// Stopping transaction instead of waiting until deadline.
		if stopErr := c.a.Stop(m.TransactionID); stopErr != nil {
			return StopErr{
				Err:   stopErr,
				Cause: err,
			}
		}
	}
	return err
}
