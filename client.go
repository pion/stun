package stun

import (
	"fmt"
	"net"
	"time"

	"github.com/pkg/errors"
)

var (
	maxMessageSize = 1280

	// ErrResponseTooBig is returned if more than maxMessageSize bytes are returned in the response
	// see https://tools.ietf.org/html/rfc5389#section-7 for the size limit
	ErrResponseTooBig = errors.New("received too much data")
)

// Client is a STUN client that sents STUN requests and receives STUN responses
type Client struct {
	conn net.Conn
}

// NewClient creates a configured STUN client
func NewClient(protocol, server string, deadline time.Duration) (*Client, error) {
	dialer := &net.Dialer{
		Timeout: deadline,
	}
	conn, err := dialer.Dial(protocol, server)
	if err != nil {
		return nil, err
	}
	err = conn.SetReadDeadline(time.Now().Add(deadline))
	if err != nil {
		return nil, err
	}
	err = conn.SetWriteDeadline(time.Now().Add(deadline))
	if err != nil {
		return nil, err
	}
	return &Client{
		conn: conn,
	}, nil
}

// LocalAddr returns local address of the client
func (c *Client) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// Close disconnects the client
func (c *Client) Close() error {
	return c.conn.Close()
}

// Request executes a STUN request against the clients server
func (c *Client) Request() (*Message, error) {
	return request(c.conn.Read, c.conn.Write)
}

// GetMappedAddressUDP initiates a stun requests to serverAddr using conn, reads the response and returns
// the XorAddress returned by the stun server via the AttrXORMappedAddress attribute
func GetMappedAddressUDP(conn *net.UDPConn, serverAddr net.Addr, deadline time.Duration) (*XorAddress, error) {
	var err error
	if deadline > 0 {
		err = conn.SetReadDeadline(time.Now().Add(deadline))
		if err != nil {
			return nil, err
		}
		err = conn.SetWriteDeadline(time.Now().Add(deadline))
		if err != nil {
			return nil, err
		}
	}

	resp, err := request(
		conn.Read,
		func(b []byte) (int, error) {
			return conn.WriteTo(b, serverAddr)
		},
	)
	if err != nil {
		return nil, err
	}

	if deadline > 0 {
		err = conn.SetReadDeadline(time.Time{})
		if err != nil {
			return nil, err
		}
		err = conn.SetWriteDeadline(time.Time{})
		if err != nil {
			return nil, err
		}
	}

	attr, ok := resp.GetOneAttribute(AttrXORMappedAddress)
	if !ok {
		return nil, fmt.Errorf("got response from STUN server that did not contain XORAddress")
	}

	addr := &XorAddress{}
	if err = addr.Unpack(resp, attr); err != nil {
		return nil, fmt.Errorf("failed to unpack STUN XorAddress response: %v", err)
	}

	return addr, nil
}

func request(read func([]byte) (int, error), write func([]byte) (int, error)) (*Message, error) {
	req, err := Build(ClassRequest, MethodBinding, GenerateTransactionID())
	if err != nil {
		return nil, err
	}

	_, err = write(req.Pack())
	if err != nil {
		return nil, err
	}

	bs := make([]byte, maxMessageSize)
	n, err := read(bs)
	if err != nil {
		return nil, err
	}
	if n > maxMessageSize {
		return nil, ErrResponseTooBig
	}

	return NewMessage(bs[:n])
}
