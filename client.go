package stun

import (
	"errors"
	"net"
	"net/url"
	"strconv"
	"sync"
	"time"
)

type MultiplexFunc func([]byte) bool

type multiplexClient struct {
	f MultiplexFunc
	c func([]byte, net.Addr)
}

type Multiplexer struct {
	conn    *net.UDPConn
	clients []multiplexClient
}

func (m *Multiplexer) Add(f MultiplexFunc, c func([]byte, net.Addr)) {
	m.clients = append(m.clients, multiplexClient{
		f: f,
		c: c,
	})
}

type ClientMultiplexer interface {
	Add(f MultiplexFunc, c func([]byte, net.Addr))
	WriteTo(b []byte, addr net.Addr) (int, error)
}

type Client struct {
	conn         *net.UDPConn
	addr         net.Addr
	m            ClientMultiplexer
	timeout      time.Duration
	transactions map[[transactionIDSize]byte]func(*Message)
}

func (m *Multiplexer) Read(b []byte) (int, error) {
	n, addr, err := m.conn.ReadFrom(b)
	if err != nil {
		return n, err
	}
	for _, c := range m.clients {
		if !c.f(b) {
			continue
		}
		c.c(b, addr)
	}
	return n, nil
}

func (c *Client) processMessage(b []byte, addr net.Addr) {
	m := new(Message)
	m.Raw = b
	if err := m.Decode(); err != nil {
		return
	}
	f, ok := c.transactions[m.TransactionID]
	if !ok {
		return
	}
	f(m)
}

func (c Client) writeTo(b []byte, addr net.Addr) (int, error) {
	if c.m != nil {
		return c.m.WriteTo(b, addr)
	}
	if c.conn != nil {
		return c.conn.WriteTo(b, addr)
	}
	return 0, errors.New("wtf")
}

const defaultTimeout = time.Second * 2

// Read reads and processes message from internal connection.
func (c *Client) Read() error {
	b := make([]byte, 1024)
	n, addr, err := c.conn.ReadFrom(b)
	if err != nil {
		return err
	}
	c.processMessage(b[:n], addr)
	return nil
}

func (c *Client) Do(req *Message, handler func(*Message) error) error {
	var (
		response = make(chan *Message)
	)
	wg := new(sync.WaitGroup)
	if c.transactions == nil {
		c.transactions = make(map[[transactionIDSize]byte]func(*Message))
	}
	c.transactions[req.TransactionID] = func(res *Message) {
		response <- res
		wg.Wait()
	}
	defer delete(c.transactions, req.TransactionID)
	defer close(response)
	if _, err := c.writeTo(req.Raw, c.addr); err != nil {
		return err
	}
	d := c.timeout
	if d == 0 {
		d = defaultTimeout
	}
	timeout := time.NewTimer(d)
	select {
	case <-timeout.C:
		return errors.New("timed out")
	case r := <-response:
		wg.Add(1)
		err := handler(r)
		wg.Done()
		return err
	}
}

// Multiplex sets m as multiplexer to client and
// calls multiplexer Add method to multiplex STUN
// messages via c.
func (c *Client) Multiplex(m ClientMultiplexer) {
	m.Add(IsMessage, c.processMessage)
	c.m = m
}

func Dial(uri string) (*Client, error) {
	// stun:a1.cydev.ru[:3134]?proto=udp
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	var (
		host = u.Opaque
		port = strconv.Itoa(DefaultPort)
	)
	hostParsed, portParsed, err := net.SplitHostPort(u.Opaque)
	if err == nil {
		host = hostParsed
		port = portParsed
	}
	addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(host, port))
	if err != nil {
		return nil, err
	}
	c := new(Client)
	if err := c.Dial(addr); err != nil {
		return nil, err
	}
	return c, nil
}

// Dial sets current server address. If no underlying connection
// is set, net.ListenUDP will create one.
func (c *Client) Dial(addr net.Addr) error {
	c.addr = addr
	if c.m != nil {
		// Using multiplexer.
		return nil
	}
	if c.conn != nil {
		// Using provided connection.
		return nil
	}
	// Using null addr.
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}
