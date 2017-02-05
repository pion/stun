package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/codegangsta/cli"
	"github.com/ernado/stun"
	"github.com/pkg/errors"
)

const (
	version = "0.2"
)

var (
	software = &stun.Software{
		Raw: []byte(fmt.Sprintf("cydev/stun %s", version)),
	}
)

func normalize(address string) string {
	if len(address) == 0 {
		address = "0.0.0.0"
	}
	if !strings.Contains(address, ":") {
		address = fmt.Sprintf("%s:%d", address, stun.DefaultPort)
	}
	return address
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
	Message *stun.Message
	Target  string
}

// Response is message returned from STUN server.
type Response struct {
	Message *stun.Message
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
		message    = stun.New()

		err      error
		deadline time.Time
	)
	for i := 0; i < maxRetries; i++ {
		if _, err = conn.Write(r.Message.Raw); err != nil {
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
		if message.TransactionID == r.Message.TransactionID {
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

func wrapWithLogger(f func(c *cli.Context) error) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		err := f(c)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		return err
	}
}

func discover(c *cli.Context) error {
	m := &stun.Message{
		TransactionID: stun.NewTransactionID(),
		Type: stun.MessageType{
			Method: stun.MethodBinding,
			Class:  stun.ClassRequest,
		},
	}
	m.AddRaw(stun.AttrSoftware, software.Raw)
	m.WriteHeader()

	request := Request{
		Message: m,
		Target:  normalize(c.String("server")),
	}

	return DefaultClient.Do(request, func(r Response) error {
		var (
			ip   net.IP
			port int
			err  error
		)
		ip, port, err = r.Message.GetXORMappedAddress()
		if err != nil {
			return errors.Wrap(err, "failed to get ip")
		}
		fmt.Println(ip, port)
		return nil
	})
}

func main() {
	app := cli.NewApp()
	app.Name = "stun"
	app.Usage = "command line client for STUN"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "server",
			Value: "ci.cydev.ru",
			Usage: "STUN server address",
		},
	}
	app.Action = wrapWithLogger(discover)
	app.Version = version
	app.Run(os.Args)
}
