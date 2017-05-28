package stun

import (
	"errors"
	"net"
	"net/url"
	"strconv"
	"sync"
	"time"
)

type AgentOptions struct {
	Handler AgentFn // Default handler, can be nil.
}

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
	zeroHandler  AgentFn
	mux          sync.Mutex // protects transactions and closed
}

// AgentFn is called on incoming message.
// Usage of message is valid only during call.
type AgentFn func(e AgentEvent)

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
		return errors.New("not exists")
	}
	t.f(AgentEvent{
		Error: errors.New("stopped"),
	})
	return nil
}

var ErrAgentClosed = errors.New("agent is closed")

func (a *Agent) Start(id transactionID, deadline time.Time, f AgentFn) error {
	a.mux.Lock()
	defer a.mux.Unlock()
	if a.closed {
		return ErrAgentClosed
	}
	_, exists := a.transactions[id]
	if exists {
		return errors.New("already exists")
	}
	a.transactions[id] = agentTransaction{
		id:       id,
		f:        f,
		deadline: deadline,
	}
	return nil
}

func (a *Agent) garbageCollect(deadline time.Time) {
	var (
		toCall   []AgentFn
		toRemove []transactionID
	)
	a.mux.Lock()
	if a.closed {
		a.mux.Unlock()
		return
	}
	for id, t := range a.transactions {
		if t.deadline.After(deadline) {
			toRemove = append(toRemove, id)
			toCall = append(toCall, t.f)
		}
	}
	for _, id := range toRemove {
		delete(a.transactions, id)
	}
	a.mux.Unlock()
	event := AgentEvent{
		Error: errors.New("timed out"),
	}
	for _, f := range toCall {
		f(event)
	}
}

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

// ClientMultiplexer wraps methods that are needed to multiplex STUN and other protocols
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
	readErr      error
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
			res: make(chan response),
		}
	},
}

func (c *Client) timeOuter() {
	ticker := time.NewTicker(time.Millisecond * 5)
	for now := range ticker.C {
		c.t.Lock()
		toRemove := make([]*clientTransaction, 0, 100)
		for _, t := range c.transactions {
			if t.deadline.After(now) {
				continue
			}
			toRemove = append(toRemove, t)
		}
		for _, t := range toRemove {
			t.res <- response{
				err: ErrTimedOut,
			}
		}
		c.t.Unlock()
	}
}

func (c *Client) addTransaction(id transactionID) *clientTransaction {
	t := tPool.Get().(*clientTransaction)
	t.deadline = time.Now().Add(defaultTimeout)
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
	c.t.Unlock()
	tPool.Put(t)
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
}

func (c *Client) writeTo(b []byte, addr net.Addr) (int, error) {
	if c.m != nil {
		return c.m.WriteTo(b, addr)
	}
	if c.conn != nil {
		return c.conn.Write(b)
	}
	return 0, errors.New("client not initialized")
}

const defaultTimeout = time.Second * 2

// Read reads and processes message from internal connection.
func (c *Client) Read() error {
	c.t.Lock()
	c.readErr = nil
	c.t.Unlock()
	b := make([]byte, 1024)
	n, addr, err := c.conn.ReadFrom(b)
	if err != nil {
		c.t.Lock()
		c.readErr = err
		c.t.Unlock()
		return err
	}
	c.packets <- clientPacket{
		addr: addr,
		b:    b[:n],
	}
	return nil
}

// ReadUntilClosed calls Read until it errors. If it errors after
// c.Close call, nil error is returned.
func (c *Client) ReadUntilClosed() error {
	for {
		err := c.Read()
		if err == nil {
			continue
		}
		c.t.Lock()
		if !c.initialized {
			err = nil
		}
		c.t.Unlock()
		return err
	}
}

func (c *Client) init() {
	if c.initialized {
		return
	}
	c.packets = make(chan clientPacket)
	c.spinWorkers(8)
	go c.timeOuter()
	c.initialized = true
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

type clientTransaction struct {
	id       transactionID
	res      chan response
	deadline time.Time
	l        sync.Mutex
}

func (c *Client) LocalAddr() net.Addr {
	if c.conn != nil {
		return c.conn.LocalAddr()
	}
	return nil
}

// Do performs request-response routine handling timeouts and retransmissions.
func (c *Client) Do(req *Message, handler func(*Message) error) error {
	// TODO(ar): handle RTO
	if c.readErr != nil {
		return c.readErr
	}

	t := c.addTransaction(req.TransactionID)
	defer c.closeTransaction(t)

	if _, err := c.writeTo(req.Raw, c.addr); err != nil {
		return err
	}

	r := <-t.res
	if r.err != nil {
		return r.err
	}
	return handler(r.m)
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
	if !c.initialized {
		c.t.Unlock()
		return nil
	}
	close(c.packets)
	c.addr = nil
	c.initialized = false
	c.m = nil
	c.conn.Close()
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
	if udpAddr, ok := addr.(*net.UDPAddr); ok {
		conn, err := net.DialUDP("udp", nil, udpAddr)
		if err != nil {
			return err
		}
		c.conn = conn
	} else {
		conn, err := net.ListenUDP("udp", nil)
		if err != nil {
			return err
		}
		c.conn = conn
	}

	return nil
}
