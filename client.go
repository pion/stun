// SPDX-FileCopyrightText: 2026 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package stun

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pion/dtls/v3"
	"github.com/pion/logging"
	"github.com/pion/transport/v4"
	"github.com/pion/transport/v4/stdnet"
)

// ErrUnsupportedURI is an error thrown if the user passes an unsupported STUN or TURN URI.
var ErrUnsupportedURI = fmt.Errorf("invalid schema or transport")

// Dial connects to the address on the named network and then
// initializes Client on that connection, returning error if any.
func Dial(network, address string) (*Client, error) {
	conn, err := net.Dial(network, address) //nolint: noctx
	if err != nil {
		return nil, err
	}

	return NewClient(conn)
}

// DialConfig is used to pass configuration to DialURI().
type DialConfig struct {
	DTLSOptions []dtls.ClientOption
	TLSConfig   tls.Config

	Net transport.Net
}

// DialURI connect to the STUN/TURN URI and then
// initializes Client on that connection, returning error if any.
func DialURI(uri *URI, cfg *DialConfig) (*Client, error) { //nolint:cyclop
	var conn Connection
	var err error

	nw := cfg.Net
	if nw == nil {
		nw, err = stdnet.NewNet()
		if err != nil {
			return nil, fmt.Errorf("failed to create net: %w", err)
		}
	}

	addr := net.JoinHostPort(uri.Host, strconv.Itoa(uri.Port))

	switch {
	case uri.Scheme == SchemeTypeSTUN:
		if conn, err = nw.Dial("udp", addr); err != nil {
			return nil, fmt.Errorf("failed to listen: %w", err)
		}

	case uri.Scheme == SchemeTypeTURN:
		network := "udp" //nolint:goconst
		if uri.Proto == ProtoTypeTCP {
			network = "tcp" //nolint:goconst
		}

		if conn, err = nw.Dial(network, addr); err != nil {
			return nil, fmt.Errorf("failed to dial: %w", err)
		}

	case uri.Scheme == SchemeTypeTURNS && uri.Proto == ProtoTypeUDP:
		udpAddr, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve UDPAddr: %w", err)
		}

		udpConn, err := nw.DialUDP("udp", nil, udpAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to dial: %w", err)
		}

		dtlsOptions := append([]dtls.ClientOption{}, cfg.DTLSOptions...)
		dtlsOptions = append(dtlsOptions, dtls.WithServerName(uri.Host))
		dtlsConn, err := dtls.ClientWithOptions(udpConn, udpConn.RemoteAddr(), dtlsOptions...)
		if err != nil {
			_ = udpConn.Close()

			return nil, fmt.Errorf("failed to connect to '%s': %w", addr, err)
		}
		conn = dtlsConn

	case (uri.Scheme == SchemeTypeTURNS || uri.Scheme == SchemeTypeSTUNS) && uri.Proto == ProtoTypeTCP:
		tlsCfg := cfg.TLSConfig //nolint:govet, copylocks
		tlsCfg.ServerName = uri.Host

		tcpConn, err := nw.Dial("tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("failed to dial: %w", err)
		}

		conn = tls.Client(tcpConn, &tlsCfg)

	default:
		return nil, ErrUnsupportedURI
	}

	return NewClient(conn)
}

// ErrNoConnection means that ClientOptions.Connection is nil.
var ErrNoConnection = errors.New("no connection provided")

// ClientOption sets some client option.
type ClientOption func(c *Client)

// WithHandler sets client handler which is called if Agent emits the Event
// with TransactionID that is not currently registered by Client.
// Useful for handling Data indications from TURN server.
func WithHandler(h Handler) ClientOption {
	return func(c *Client) {
		c.handler = h
	}
}

// WithRTO sets client RTO as defined in STUN RFC.
func WithRTO(rto time.Duration) ClientOption {
	return func(c *Client) {
		c.rto = int64(rto)
	}
}

// WithClock sets Clock of client, the source of current time.
// Also clock is passed to default collector if set.
func WithClock(clock Clock) ClientOption {
	return func(c *Client) {
		c.clock = clock
	}
}

// WithTimeoutRate sets RTO timer minimum resolution.
func WithTimeoutRate(d time.Duration) ClientOption {
	return func(c *Client) {
		c.rtoRate = d
	}
}

// WithAgent sets client STUN agent.
//
// Defaults to agent implementation in current package,
// see agent.go.
func WithAgent(a ClientAgent) ClientOption {
	return func(c *Client) {
		c.a = a
	}
}

// WithCollector rests client timeout collector, the implementation
// of ticker which calls function on each tick.
func WithCollector(coll Collector) ClientOption {
	return func(c *Client) {
		c.collector = coll
	}
}

// WithNoConnClose prevents client from closing underlying connection when
// the Close() method is called.
func WithNoConnClose() ClientOption {
	return func(c *Client) {
		c.closeConn = false
	}
}

// WithStrictMode sets strict decode behavior for messages decoded by the client.
func WithStrictMode(strict bool) ClientOption {
	return func(c *Client) {
		c.strict = strict
	}
}

// WithLoggerFactory sets logger used by the client and decoded messages.
func WithLoggerFactory(loggerFactory logging.LoggerFactory) ClientOption {
	return func(c *Client) {
		if loggerFactory == nil {
			c.logger = nil

			return
		}
		c.logger = loggerFactory.NewLogger("stun")
	}
}

// WithNoRetransmit disables retransmissions and sets RTO to
// defaultMaxAttempts * defaultRTO which will be effectively time out
// if not set.
//
// Useful for TCP connections where transport handles RTO.
func WithNoRetransmit(c *Client) {
	c.maxAttempts = 0
	if c.rto == 0 {
		c.rto = defaultMaxAttempts * int64(defaultRTO)
	}
}

const (
	defaultTimeoutRate = time.Millisecond * 5
	defaultRTO         = time.Millisecond * 300
	defaultMaxAttempts = 7
)

// NewClient initializes new Client from provided options,
// starting internal goroutines and using default options fields
// if necessary. Call Close method after using Client to close conn and
// release resources.
//
// The conn will be closed on Close call. Use WithNoConnClose option to
// prevent that.
//
// Note that user should handle the protocol multiplexing, client does not
// provide any API for it, so if you need to read application data, wrap the
// connection with your (de-)multiplexer and pass the wrapper as conn.
func NewClient(conn Connection, options ...ClientOption) (*Client, error) {
	client := &Client{
		close:       make(chan struct{}),
		c:           conn,
		clock:       systemClock(),
		rto:         int64(defaultRTO),
		rtoRate:     defaultTimeoutRate,
		t:           make(map[transactionID]*clientTransaction, 100),
		maxAttempts: defaultMaxAttempts,
		closeConn:   true,
	}
	for _, o := range options {
		o(client)
	}
	if client.c == nil {
		return nil, ErrNoConnection
	}
	if client.a == nil {
		client.a = NewAgent(nil)
	}
	if err := client.a.SetHandler(client.handleAgentCallback); err != nil {
		return nil, err
	}
	if client.collector == nil {
		client.collector = &tickerCollector{
			close: make(chan struct{}),
			clock: client.clock,
		}
	}
	if err := client.collector.Start(client.rtoRate, func(t time.Time) {
		closedOrPanic(client.a.Collect(t))
	}); err != nil {
		return nil, err
	}
	client.wg.Add(1)
	go client.readUntilClosed()
	runtime.SetFinalizer(client, clientFinalizer)

	return client, nil
}

func clientFinalizer(c *Client) {
	if c == nil {
		return
	}
	err := c.Close()
	if errors.Is(err, ErrClientClosed) {
		return
	}
	if err == nil {
		log.Println("client: called finalizer on non-closed client") // nolint

		return
	}
	log.Println("client: called finalizer on non-closed client:", err) // nolint
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
	Start(id [TransactionIDSize]byte, deadline time.Time) error
	Stop(id [TransactionIDSize]byte) error
	Collect(time.Time) error
	SetHandler(h Handler) error
}

// Client simulates "connection" to STUN server.
type Client struct {
	rto         int64 // time.Duration
	a           ClientAgent
	c           Connection
	close       chan struct{}
	rtoRate     time.Duration
	maxAttempts int32
	closed      bool
	closeConn   bool // should call c.Close() while closing
	wg          sync.WaitGroup
	clock       Clock
	handler     Handler
	collector   Collector
	t           map[transactionID]*clientTransaction
	logger      logging.LeveledLogger
	strict      bool

	// mux guards closed and t
	mux sync.RWMutex
}

// clientTransaction represents transaction in progress.
// If transaction is succeed or failed, f will be called
// provided by event.
// Concurrent access is invalid.
type clientTransaction struct {
	id      transactionID
	attempt int32
	calls   int32
	h       Handler
	start   time.Time
	rto     time.Duration
	raw     []byte
}

func (t *clientTransaction) handle(e Event) {
	if atomic.AddInt32(&t.calls, 1) == 1 {
		t.h(e)
	}
}

var clientTransactionPool = &sync.Pool{ //nolint:gochecknoglobals
	New: func() any {
		return &clientTransaction{
			raw: make([]byte, 1500),
		}
	},
}

func acquireClientTransaction() *clientTransaction {
	return clientTransactionPool.Get().(*clientTransaction) //nolint:forcetypeassert
}

func putClientTransaction(t *clientTransaction) {
	t.raw = t.raw[:0]
	t.start = time.Time{}
	t.attempt = 0
	t.id = transactionID{}
	clientTransactionPool.Put(t)
}

func (t *clientTransaction) nextTimeout(now time.Time) time.Time {
	return now.Add(time.Duration(t.attempt+1) * t.rto)
}

// start registers transaction.
//
// Could return ErrClientClosed, ErrTransactionExists.
func (c *Client) start(t *clientTransaction) error {
	c.mux.Lock()
	defer c.mux.Unlock()
	if c.closed {
		return ErrClientClosed
	}
	_, exists := c.t[t.id]
	if exists {
		return ErrTransactionExists
	}
	c.t[t.id] = t

	return nil
}

// Clock abstracts the source of current time.
type Clock interface {
	Now() time.Time
}

type systemClockService struct{}

func (systemClockService) Now() time.Time { return time.Now() }

func systemClock() systemClockService {
	return systemClockService{}
}

// SetRTO sets current RTO value.
func (c *Client) SetRTO(rto time.Duration) {
	atomic.StoreInt64(&c.rto, int64(rto))
}

// StopErr occurs when Client fails to stop transaction while
// processing error.
//
//nolint:errname
type StopErr struct {
	Err   error // value returned by Stop()
	Cause error // error that caused Stop() call
}

func (e StopErr) Error() string {
	return fmt.Sprintf("error while stopping due to %s: %s", sprintErr(e.Cause), sprintErr(e.Err))
}

// CloseErr indicates client close failure.
//
//nolint:errname
type CloseErr struct {
	AgentErr      error
	ConnectionErr error
}

func sprintErr(err error) string {
	if err == nil {
		return "<nil>" //nolint:goconst
	}

	return err.Error()
}

func (c CloseErr) Error() string {
	return fmt.Sprintf("failed to close: %s (connection), %s (agent)", sprintErr(c.ConnectionErr), sprintErr(c.AgentErr))
}

func (c *Client) readUntilClosed() {
	defer c.wg.Done()
	options := []MessageOption{
		WithStrict(c.strict),
	}
	if c.logger != nil {
		options = append(options, withMessageLogger(c.logger))
	}
	msg := NewWithOptions(options...)
	if isStreamConnection(c.c) {
		c.readUntilClosedStream(msg)

		return
	}
	// Datagram connections deliver one message per read, so the buffer
	// must fit the largest possible STUN message to avoid truncation.
	msg.Raw = make([]byte, maxMessageSize)
	for {
		select {
		case <-c.close:
			return
		default:
		}
		_, err := msg.ReadFrom(c.c)
		if err != nil {
			// The connection is unusable after a read error: the peer
			// may have closed it, or the stream may be out of sync.
			// Exiting the read loop avoids busy-spinning on a permanently
			// failing connection; pending transactions will time out via
			// the collector.
			c.warnReadLoop(err)

			return
		}
		if pErr := c.a.Process(msg); errors.Is(pErr, ErrAgentClosed) {
			return
		}
	}
}

// warnReadLoop logs a read loop termination if a logger is configured.
func (c *Client) warnReadLoop(err error) {
	if c.logger != nil {
		c.logger.Warnf("read loop stopped due to error: %v", err)
	}
}

// isStreamConnection reports whether conn delivers a byte stream (TCP,
// TLS) instead of datagrams (UDP, DTLS). Connections that expose no local
// address are treated as datagrams to preserve the previous behavior.
func isStreamConnection(conn Connection) bool {
	la, ok := conn.(interface{ LocalAddr() net.Addr })
	if !ok {
		return false
	}
	addr := la.LocalAddr()
	if addr == nil {
		return false
	}

	switch addr.Network() {
	case "udp", "udp4", "udp6":
		return false
	default:
		return true
	}
}

// readUntilClosedStream reads and processes complete STUN messages from a
// stream connection, reassembling fragmented messages and splitting
// messages coalesced into a single read.
func (c *Client) readUntilClosedStream(msg *Message) {
	buf := make([]byte, 0, 1500)
	scratch := make([]byte, 1500)
	for {
		select {
		case <-c.close:
			return
		default:
		}
		if err := readFullMessage(c.c, msg, &buf, scratch); err != nil {
			c.warnReadLoop(err)

			return
		}
		if err := msg.Decode(); err != nil {
			c.warnReadLoop(err)

			return
		}
		if msg.strict && msg.Type.Value() == 0 {
			c.warnReadLoop(ErrInvalidType)

			return
		}
		if pErr := c.a.Process(msg); errors.Is(pErr, ErrAgentClosed) {
			return
		}
	}
}

// readFullMessage reads from r until buf contains a complete STUN message,
// extracts it into msg.Raw and leaves any trailing bytes in buf for the
// next call.
func readFullMessage(r io.Reader, msg *Message, buf *[]byte, scratch []byte) error {
	for {
		if size, ok := fullSTUNMessageSize(*buf); ok {
			msg.Raw = append(msg.Raw[:0], (*buf)[:size]...)
			copy(*buf, (*buf)[size:])
			*buf = (*buf)[:len(*buf)-size]

			return nil
		}
		n, err := r.Read(scratch)
		if n > 0 {
			*buf = append(*buf, scratch[:n]...)
		}
		if err != nil {
			// The stream ended: deliver a final complete message if one
			// is buffered, otherwise report the error.
			if size, ok := fullSTUNMessageSize(*buf); ok {
				msg.Raw = append(msg.Raw[:0], (*buf)[:size]...)

				return nil
			}

			return err
		}
	}
}

func closedOrPanic(err error) {
	if err == nil || errors.Is(err, ErrAgentClosed) {
		return
	}
	panic(err) //nolint
}

type tickerCollector struct {
	close chan struct{}
	wg    sync.WaitGroup
	clock Clock
}

// Collector calls function f with constant rate.
//
// The simple Collector is ticker which calls function on each tick.
type Collector interface {
	Start(rate time.Duration, f func(now time.Time)) error
	Close() error
}

func (a *tickerCollector) Start(rate time.Duration, f func(now time.Time)) error {
	t := time.NewTicker(rate)
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		for {
			select {
			case <-a.close:
				t.Stop()

				return
			case <-t.C:
				f(a.clock.Now())
			}
		}
	}()

	return nil
}

func (a *tickerCollector) Close() error {
	close(a.close)
	a.wg.Wait()

	return nil
}

// ErrClientClosed indicates that client is closed.
var ErrClientClosed = errors.New("client is closed")

// Close stops internal connection and agent, returning CloseErr on error.
func (c *Client) Close() error {
	if err := c.checkInit(); err != nil {
		return err
	}
	c.mux.Lock()
	if c.closed {
		c.mux.Unlock()

		return ErrClientClosed
	}
	c.closed = true
	c.mux.Unlock()
	if closeErr := c.collector.Close(); closeErr != nil {
		return closeErr
	}
	var connErr error
	agentErr := c.a.Close()
	if c.closeConn {
		connErr = c.c.Close()
	}
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
	return c.Start(m, nil)
}

// callbackWaitHandler blocks on wait() call until callback is called.
type callbackWaitHandler struct {
	handler   Handler
	callback  func(event Event)
	cond      *sync.Cond
	processed bool
}

func (s *callbackWaitHandler) HandleEvent(e Event) {
	s.cond.L.Lock()
	if s.callback == nil {
		panic("s.callback is nil") //nolint
	}
	s.callback(e)
	s.processed = true
	s.cond.Broadcast()
	s.cond.L.Unlock()
}

func (s *callbackWaitHandler) wait() {
	s.cond.L.Lock()
	for !s.processed {
		s.cond.Wait()
	}
	s.processed = false
	s.callback = nil
	s.cond.L.Unlock()
}

func (s *callbackWaitHandler) setCallback(f func(event Event)) {
	if f == nil {
		panic("f is nil") //nolint
	}
	s.cond.L.Lock()
	s.callback = f
	if s.handler == nil {
		s.handler = s.HandleEvent
	}
	s.cond.L.Unlock()
}

var callbackWaitHandlerPool = sync.Pool{ //nolint:gochecknoglobals
	New: func() any {
		return &callbackWaitHandler{
			cond: sync.NewCond(new(sync.Mutex)),
		}
	},
}

// ErrClientNotInitialized means that client connection or agent is nil.
var ErrClientNotInitialized = errors.New("client not initialized")

func (c *Client) checkInit() error {
	if c == nil || c.c == nil || c.a == nil || c.close == nil {
		return ErrClientNotInitialized
	}

	return nil
}

// Do is Start wrapper that waits until callback is called. If no callback
// provided, Indicate is called instead.
//
// Do has cpu overhead due to blocking, see BenchmarkClient_Do.
// Use Start method for less overhead.
func (c *Client) Do(m *Message, f func(Event)) error {
	if err := c.checkInit(); err != nil {
		return err
	}
	if f == nil {
		return c.Indicate(m)
	}
	h := callbackWaitHandlerPool.Get().(*callbackWaitHandler) //nolint:forcetypeassert
	h.setCallback(f)
	defer func() {
		callbackWaitHandlerPool.Put(h)
	}()
	if err := c.Start(m, h.handler); err != nil {
		return err
	}
	h.wait()

	return nil
}

func (c *Client) delete(id transactionID) {
	c.mux.Lock()
	if c.t != nil {
		delete(c.t, id)
	}
	c.mux.Unlock()
}

type buffer struct {
	buf []byte
}

var bufferPool = &sync.Pool{ //nolint:gochecknoglobals
	New: func() any {
		return &buffer{buf: make([]byte, 2048)}
	},
}

func (c *Client) handleAgentCallback(event Event) { //nolint:cyclop
	c.mux.Lock()
	if c.closed {
		c.mux.Unlock()

		return
	}
	transaction, found := c.t[event.TransactionID]
	if found {
		delete(c.t, transaction.id)
	}
	c.mux.Unlock()
	if !found {
		if c.handler != nil && !errors.Is(event.Error, ErrTransactionStopped) {
			c.handler(event)
		}
		// Ignoring.
		return
	}
	if atomic.LoadInt32(&c.maxAttempts) <= transaction.attempt || event.Error == nil {
		// Transaction completed.
		transaction.handle(event)
		putClientTransaction(transaction)

		return
	}
	// Doing re-transmission.
	transaction.attempt++
	buff := bufferPool.Get().(*buffer) //nolint:forcetypeassert
	buff.buf = buff.buf[:copy(buff.buf[:cap(buff.buf)], transaction.raw)]
	defer bufferPool.Put(buff)
	var (
		now     = c.clock.Now()
		timeOut = transaction.nextTimeout(now)
		id      = transaction.id
	)
	// Starting client transaction.
	if startErr := c.start(transaction); startErr != nil {
		c.delete(id)
		event.Error = startErr
		transaction.handle(event)
		putClientTransaction(transaction)

		return
	}
	// Starting agent transaction.
	if startErr := c.a.Start(id, timeOut); startErr != nil {
		c.delete(id)
		event.Error = startErr
		transaction.handle(event)
		putClientTransaction(transaction)

		return
	}
	// Writing message to connection again.
	_, writeErr := c.c.Write(buff.buf)
	if writeErr != nil {
		c.delete(id)
		event.Error = writeErr
		// Stopping agent transaction instead of waiting until it's deadline.
		// This will call handleAgentCallback with "ErrTransactionStopped" error
		// which will be ignored.
		if stopErr := c.a.Stop(id); stopErr != nil {
			// Failed to stop agent transaction. Wrapping the error in StopError.
			event.Error = StopErr{
				Err:   stopErr,
				Cause: writeErr,
			}
		}
		transaction.handle(event)
		putClientTransaction(transaction)

		return
	}
}

// Start starts transaction (if h set) and writes message to server, handler
// is called asynchronously.
func (c *Client) Start(msg *Message, handler Handler) error {
	if err := c.checkInit(); err != nil {
		return err
	}
	c.mux.RLock()
	closed := c.closed
	c.mux.RUnlock()
	if closed {
		return ErrClientClosed
	}
	if handler != nil {
		// Starting transaction only if h is set. Useful for indications.
		t := acquireClientTransaction()
		t.id = msg.TransactionID
		t.start = c.clock.Now()
		t.h = handler
		t.rto = time.Duration(atomic.LoadInt64(&c.rto))
		t.attempt = 0
		t.raw = append(t.raw[:0], msg.Raw...)
		t.calls = 0
		d := t.nextTimeout(t.start)
		if err := c.start(t); err != nil {
			return err
		}
		if err := c.a.Start(msg.TransactionID, d); err != nil {
			return err
		}
	}
	_, err := msg.WriteTo(c.c)
	if err != nil && handler != nil {
		c.delete(msg.TransactionID)
		// Stopping transaction instead of waiting until deadline.
		if stopErr := c.a.Stop(msg.TransactionID); stopErr != nil {
			return StopErr{
				Err:   stopErr,
				Cause: err,
			}
		}
	}

	return err
}
