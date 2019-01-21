package stun

import (
	"fmt"
	"net"
	"strconv"

	"github.com/pkg/errors"
)

// TransportAddr is struct with ip address and port number
type TransportAddr struct {
	IP   net.IP
	Port int
	//Zone    string (udpv6, tcpv6)
	//Network string (udp, tcp)
}

func netAddrIPPort(addr net.Addr) (net.IP, int, error) {
	host, portStr, err := net.SplitHostPort(addr.String())
	if err != nil {
		return nil, 0, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, 0, err
	}

	return net.ParseIP(host), port, nil
}

// NewTransportAddr returns transportadd struct within address and port
func NewTransportAddr(addr net.Addr) (*TransportAddr, error) {
	ip, port, err := netAddrIPPort(addr)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse addr")
	}

	return &TransportAddr{
		IP:   ip,
		Port: port,
	}, nil
}

// Equal returns both of address and port is same
func (a *TransportAddr) Equal(b *TransportAddr) bool {
	return a.IP.Equal(b.IP) && a.Port == b.Port
}

// Addr returns net.UDPAddr from TransportAddr
func (a *TransportAddr) Addr() net.Addr {
	return &net.UDPAddr{
		IP:   a.IP,
		Port: a.Port,
		//Zone: a.Zone, // udpv6
	}

	// Handle other network types here (TCPv4/6)
}

// String returns "address:port"
func (a *TransportAddr) String() string {
	return fmt.Sprintf("%s:%d", a.IP.String(), a.Port)
}
