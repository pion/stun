package stun

import (
	"io"
	"net"

	"github.com/pkg/errors"
)

// Clipped from net.IP
// Is p all zeros?
func isZeros(p net.IP) bool {
	for i := 0; i < len(p); i++ {
		if p[i] != 0 {
			return false
		}
	}
	return true
}

// Clipped from net.IP
// To4 converts the IPv4 address ip to a 4-byte representation.
// If ip is not an IPv4 address, To4 returns nil.
func toIPv4(ip net.IP) net.IP {
	if len(ip) == net.IPv4len {
		return ip
	}
	if len(ip) == net.IPv6len &&
		isZeros(ip[0:10]) &&
		ip[10] == 0xff &&
		ip[11] == 0xff {
		return ip[12:16]
	}
	return nil
}

func xor(dst, l, r []byte) {
	n := len(l)
	if len(r) < n {
		n = len(r)
	}
	for i := 0; i < n; i++ {
		dst[i] = l[i] ^ r[i]
	}
}

// https://tools.ietf.org/html/rfc5389#section-15.1
// The address family can take on the following values:
//   0x01:IPv4
//   0x02:IPv6
const (
	familyIPv4 uint16 = 0x01
	familyIPv6 uint16 = 0x02
)

const (
	familyStart = 0
	// family length is actually 1 byte, but the preceding byte in the header must be all zero according
	// to https://tools.ietf.org/html/rfc5389#section-15.1
	familyLength = 2
	portStart    = 2
	portLength   = 2
	addressStart = 4
)

func getIPAndFamily(unkIP net.IP) (ip net.IP, family uint16, err error) {
	ip = unkIP
	family = familyIPv4

	if len(ip) == net.IPv6len {
		ipv4 := toIPv4(ip)
		if ipv4 != nil {
			ip = ipv4
		} else {
			family = familyIPv6
		}
	} else if len(ip) != net.IPv4len {
		err = errors.Errorf("invalid IP length %d", len(ip))
	}

	return ip, family, err
}

// XorAddress is struct with in ip address and port number
type XorAddress struct {
	IP   net.IP
	Port int
}

func (x *XorAddress) packInner(message *Message) ([]byte, error) {
	ip, family, err := getIPAndFamily(x.IP)
	if err != nil {
		return []byte{}, errors.Wrap(err, "unable to get IP and family")
	}

	len := familyLength + portLength + len(ip)
	v := make([]byte, len)

	// Family
	enc.PutUint16(v[familyStart:familyStart+familyLength], family)
	// Port
	enc.PutUint16(v[portStart:portStart+portLength], uint16(x.Port))
	xor(v[portStart:portStart+portLength], v[portStart:portStart+portLength], message.TransactionID[0:2])
	// Address
	copy(v[addressStart:], ip)
	xor(v[addressStart:], v[addressStart:], message.TransactionID)

	return v, nil
}

// Unpack message checking address and port suites
func (x *XorAddress) Unpack(message *Message, rawAttribute *RawAttribute) error {
	v := rawAttribute.Value

	if len(v) < familyStart+familyLength {
		return io.ErrUnexpectedEOF
	}

	family := enc.Uint16(v[familyStart : familyStart+familyLength])

	if family != familyIPv4 && family != familyIPv6 {
		return errors.Errorf("invalid family %d (expected IPv4(%d) or IPv6(%d)", family, familyIPv4, familyIPv6)
	}

	if len(v[portStart:]) < portLength {
		return io.ErrUnexpectedEOF
	}

	var p [2]byte
	// Transaction ID [0,2] is top half of magic cookie
	xor(p[:], v[portStart:portStart+portLength], message.TransactionID[0:2])

	x.Port = int(enc.Uint16(p[:]))

	al := net.IPv4len
	if family == familyIPv6 {
		al = net.IPv6len
	}

	if len(v[addressStart:]) < al {
		return io.ErrUnexpectedEOF
	}

	if len(v[addressStart:]) > al {
		return errors.Errorf("invalid length for %d family address (%d)", family, len(v[4:]))
	}

	x.IP = make([]byte, al)
	xor(x.IP[:], v[4:], message.TransactionID)

	return nil
}
