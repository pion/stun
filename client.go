package stun

import (
	"errors"
	"net"
	"strconv"
)

type Client struct {
	c         net.Conn
	addresses string
}

// SetServerHost allows user to set the STUN hostname and port.
func (c *Client) SetServerHost(host string, port int) error {
	ips, err := net.LookupHost(host)
	if err != nil {
		return err
	}
	if len(ips) == 0 {
		return errors.New("Failed to get IP address of " + host + ".")
	}
	c.addresses = net.JoinHostPort(ips[0], strconv.Itoa(port))
	return nil
}
