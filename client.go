package stun

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

func Dial(network, address string) (*Client, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return NewClient(ClientOptions{
		Connection: conn,
	}), nil
}

type ClientOptions struct {
	AgentOptions
	Connection  Connection
	TimeoutRate time.Duration
}

const defaultTimeoutRate = time.Millisecond * 100

func NewClient(options ClientOptions) *Client {
	a := NewAgent(options.AgentOptions)
	c := &Client{
		close:  make(chan struct{}),
		c:      options.Connection,
		a:      a,
		gcRate: options.TimeoutRate,
	}
	if c.gcRate == 0 {
		c.gcRate = defaultTimeoutRate
	}
	c.wg.Add(2)
	go c.readUntilClosed()
	go c.collectUntilClosed()
	return c
}

type Connection interface {
	io.Reader
	io.Writer
	io.Closer
}

// Client simulates "connection" to STUN server.
type Client struct {
	a         *Agent
	c         Connection
	close     chan struct{}
	closed    bool
	closedMux sync.Mutex
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
		e.Cause, e.Err,
	)
}

// CloseErr indicates client close failure.
type CloseErr struct {
	AgentErr      error
	ConnectionErr error
}

func (c CloseErr) Error() string {
	return fmt.Sprintf("failed to close: %s (connection), %s (agent)",
		c.ConnectionErr, c.AgentErr,
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
	if err == nil {
		return
	}
	if err == ErrAgentClosed {
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

// Do starts transaction (if f set) and writes message to server.
func (c *Client) Do(m *Message, d time.Time, f func(AgentEvent)) error {
	if f != nil {
		// Starting transaction only if f is set. Useful for indications.
		if err := c.a.Start(m.TransactionID, d, f); err != nil {
			fmt.Println("failed to Start()")
			return err
		}
	}
	_, err := m.WriteTo(c.c)
	if err != nil {
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
