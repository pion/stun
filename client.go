package stun

import (
	"net"
	"time"

	"github.com/pkg/errors"
)

// SameTransaction returns true of a and b have same Transaction ID.
// DEPRECATED: remove usage.
func SameTransaction(a *Message, b *Message) bool {
	return a.TransactionID == b.TransactionID
}

// Defaults for Client fields.
const (
	DefaultClientRetries  = 9
	DefaultMaxTimeout     = 2 * time.Second
	DefaultInitialTimeout = 1 * time.Millisecond
)

// DefaultClient is Client with defaults that are close
// to RFC recommendations.
var DefaultClient = Client{}

// Client implements STUN client.
type Client struct {
	Retries        int
	MaxTimeout     time.Duration
	InitialTimeout time.Duration

	addr *net.UDPAddr
}

func (c Client) getRetries() int {
	if c.Retries == 0 {
		return DefaultClientRetries
	}
	return c.Retries
}

func (c Client) getMaxTimeout() time.Duration {
	if c.MaxTimeout == 0 {
		return DefaultMaxTimeout
	}
	return c.MaxTimeout
}

func (c Client) getInitialTimeout() time.Duration {
	if c.InitialTimeout == 0 {
		return DefaultInitialTimeout
	}
	return c.InitialTimeout
}

func (c *Client) getAddr() (*net.UDPAddr, error) {
	var (
		err  error
		addr *net.UDPAddr
	)
	if c.addr != nil {
		return c.addr, nil
	}
	addr, err = net.ResolveUDPAddr("udp", "0.0.0.0:0")
	if err == nil {
		c.addr = addr
	}
	return c.addr, err
}

// Request is wrapper on message and target server address.
type Request struct {
	Message *Message
	Target  string
}

// Response is message returned from STUN server.
type Response struct {
	Message *Message
}

// ResponseHandler is handler executed if response is received.
type ResponseHandler func(r Response) error

const timeoutGrowthRate = 2

// loop tries to send r on conn and get Response, passing it to handler.
func (c Client) loop(conn *net.UDPConn, r Request, h ResponseHandler) error {
	var (
		timeout    = c.getInitialTimeout()
		maxTimeout = c.getMaxTimeout()
		maxRetries = c.getRetries()
		message    = AcquireMessage()

		err      error
		deadline time.Time
	)
	defer ReleaseMessage(message)
	for i := 0; i < maxRetries; i++ {
		if _, err = r.Message.WriteTo(conn); err != nil {
			return errors.Wrap(err, "failed to write")
		}

		deadline = time.Now().Add(timeout)
		if err = conn.SetReadDeadline(deadline); err != nil {
			return errors.Wrap(err, "failed to set deadline")
		}

		if timeout < maxTimeout {
			timeout *= timeoutGrowthRate
		}

		message.Reset()
		if _, err = message.ReadFrom(conn); err != nil {
			if _, ok := err.(net.Error); ok {
				continue
			}
			return errors.Wrap(err, "network failed")
		}
		if SameTransaction(message, r.Message) {
			return h(Response{
				Message: message,
			})
		}
	}
	return errors.Wrap(err, "max retries reached")
}

// Do performs request and passing response to handler. If error occurs
// during request, Do returns it, not calling the handler.
// Do returns any error that is returned by handler.
// Response is only valid during handler execution.
//
// Never store Response, Message pointer or any values obtained from
// Message getters, copy message and use it if needed.
func (c Client) Do(request Request, h ResponseHandler) error {
	var (
		targetAddr *net.UDPAddr
		clientAddr *net.UDPAddr
		conn       *net.UDPConn
		err        error
	)
	// initializing connection
	if targetAddr, err = net.ResolveUDPAddr("udp", request.Target); err != nil {
		return errors.Wrap(err, "failed to resolve")
	}
	if clientAddr, err = c.getAddr(); err != nil {
		return errors.Wrap(err, "failed to get local addr")
	}
	if conn, err = net.DialUDP("udp", clientAddr, targetAddr); err != nil {
		return errors.Wrap(err, "failed to dial")
	}
	return c.loop(conn, request, h)
}
