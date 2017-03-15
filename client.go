package stun

import (
	"errors"
	"net"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// MultiplexFunc should return true if multiplex client is interested in b.
type MultiplexFunc func(b []byte) bool

type multiplexClient struct {
	f MultiplexFunc
	c func([]byte, net.Addr)
}

// Multiplexer implements multiplexing on connection.
type Multiplexer struct {
	conn    *net.UDPConn
	clients []multiplexClient
}

// Add appends new multiplexer client and function. If f returns true, c will be called with
// remote address and received data.
func (m *Multiplexer) Add(f MultiplexFunc, c func([]byte, net.Addr)) {
	m.clients = append(m.clients, multiplexClient{
		f: f,
		c: c,
	})
}

// ClientMultiplexer wraps methods that are needed to multiples STUN and other protocols
// on one connection.
type ClientMultiplexer interface {
	Add(f MultiplexFunc, c func([]byte, net.Addr))
	WriteTo(b []byte, addr net.Addr) (int, error)
}

type clientPacket struct {
	b    []byte
	addr net.Addr
}

// Client implements STUN Client to some remote server.
// Zero value will dork, but Do and Indicate calls are valid
// only after at least one Dial. Call Close to stop internal workers
// and reset state to same as zero value.
type Client struct {
	conn         *net.UDPConn
	addr         net.Addr
	m            ClientMultiplexer
	timeout      time.Duration
	transactions map[transactionID]*clientTransaction
	packets      chan clientPacket
	t            sync.Mutex
	initialized  bool
}

func (c *Client) worker() {
	m := new(Message)
	for p := range c.packets {
		m.Reset()
		m.Raw = p.b
		c.processMessage(m, c.addr)
	}
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

type transactionID [transactionIDSize]byte

var tPool = &sync.Pool{
	New: func() interface{} {
		return &clientTransaction{
			res:   make(chan response),
			close: make(chan interface{}),
		}
	},
}

func (c *Client) addTransaction(id transactionID) *clientTransaction {
	t := tPool.Get().(*clientTransaction)
	c.t.Lock()
	if c.transactions == nil {
		// Lazily initializing map.
		c.transactions = make(map[transactionID]*clientTransaction)
	}
	t.id = id
	c.transactions[id] = t
	c.t.Unlock()
	return t
}

type response struct {
	m   *Message
	err error
}

func (c *Client) closeTransaction(t *clientTransaction) {
	c.t.Lock()
	delete(c.transactions, t.id)
	t.close <- nil
	c.t.Unlock()
}

func (c *Client) processMessage(m *Message, addr net.Addr) {
	r := response{
		err: m.Decode(),
		m:   m,
	}
	c.t.Lock()
	t, ok := c.transactions[m.TransactionID]
	c.t.Unlock()
	if !ok {
		// If transaction is closed, ok should always be true.
		return
	}
	t.res <- r
	<-t.close
	tPool.Put(t)
}

func (c Client) writeTo(b []byte, addr net.Addr) (int, error) {
	if c.m != nil {
		return c.m.WriteTo(b, addr)
	}
	if c.conn != nil {
		return c.conn.WriteTo(b, addr)
	}
	return 0, errors.New("client not initialized")
}

const defaultTimeout = time.Second * 2

// Read reads and processes message from internal connection.
func (c *Client) Read() error {
	b := make([]byte, 1024)
	n, addr, err := c.conn.ReadFrom(b)
	if err != nil {
		return err
	}
	c.packets <- clientPacket{
		addr: addr,
		b:    b[:n],
	}
	return nil
}

func (c *Client) init() {
	if c.initialized {
		return
	}
	c.packets = make(chan clientPacket)
	c.spinWorkers(8)
}

func (c *Client) spinWorkers(n int) {
	for i := 0; i < n; i++ {
		go c.worker()
	}
}

// Indicate writes req to remote address.
func (c *Client) Indicate(req *Message) error {
	_, err := c.writeTo(req.Raw, c.addr)
	return err
}

// ErrTimedOut means that client did not received message in timeout window.
var ErrTimedOut = errors.New("timed out")

var timerPool = &sync.Pool{
	New: func() interface{} {
		return time.NewTimer(time.Second)
	},
}

func getTimer(d time.Duration) *time.Timer {
	t := timerPool.Get().(*time.Timer)
	t.Reset(d)
	return t
}

func putTimer(t *time.Timer) {
	t.Stop()
	timerPool.Put(t)
}

type clientTransaction struct {
	id    transactionID
	res   chan response
	close chan interface{}
	l     sync.Mutex
}

// Do performs request-response routine handling timeouts and retransmissions.
func (c *Client) Do(req *Message, handler func(*Message) error) error {
	// TODO(ar): handle RTO

	t := c.addTransaction(req.TransactionID)
	defer c.closeTransaction(t)

	if _, err := c.writeTo(req.Raw, c.addr); err != nil {
		return err
	}

	timeout := getTimer(defaultTimeout)
	defer putTimer(timeout)

	select {
	case <-timeout.C:
		return ErrTimedOut
	case r := <-t.res:
		if r.err != nil {
			return r.err
		}
		return handler(r.m)
	}
}

// Multiplex sets m as multiplexer to client and
// calls multiplexer Add method to multiplex STUN
// messages via c.
func (c *Client) Multiplex(m ClientMultiplexer) {
	m.Add(IsMessage, func(b []byte, addr net.Addr) {
		c.packets <- clientPacket{
			b:    b,
			addr: addr,
		}
	})
	c.m = m
}

// Dial creates new client to server under provided URI and returns it.
// Exaple uri is "stun:some-stun.com?proto=udp". Default port is used if no port specified.
// Currently only supports udp proto.
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

// Close stops internal workers and resets state of client.
func (c *Client) Close() error {
	c.t.Lock()
	close(c.packets)
	c.addr = nil
	c.initialized = false
	c.m = nil
	c.conn = nil
	c.transactions = nil
	c.t.Unlock()

	return nil
}

// Dial sets current server address. If no underlying connection
// is set, net.ListenUDP will create one.
func (c *Client) Dial(addr net.Addr) error {
	c.init()
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
