package stun

import "github.com/pkg/errors"

type protocolNumber byte

//ProtocolUDP User Datagram	https://www.iana.org/assignments/protocol-numbers/protocol-numbers.xhtml
const (
	ProtocolUDP protocolNumber = 0x11
)

// A RequestedTransport is used by the client to request a specific transport
// protocol for the allocated transport address.  The value of this
// attribute is 4 bytes with the following format:
//    0                   1                   2                   3
//    0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
//   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//   |    Protocol   |                    RFFU                       |
//   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//
// The Protocol field specifies the desired protocol.  The codepoints
// used in this field are taken from those allowed in the Protocol field
// in the IPv4 header and the NextHeader field in the IPv6 header
// [Protocol-Numbers].  This specification only allows the use of
// codepoint 17 (User Datagram Protocol).
//
// The RFFU field MUST be set to zero on transmission and MUST be
// ignored on reception.  It is reserved for future uses.
// https://tools.ietf.org/html/rfc5766#section-14.7

//RequestedTransport represented transport protocol
type RequestedTransport struct {
	Protocol protocolNumber
}

//Pack always error
func (r *RequestedTransport) Pack(message *Message) error {
	return errors.Errorf("stun.RequestedTransport Pack not implemented")
}

//Unpack RequestedTransport protocol
func (r *RequestedTransport) Unpack(message *Message, rawAttribute *RawAttribute) error {
	r.Protocol = protocolNumber(rawAttribute.Value[0])
	if r.Protocol != ProtocolUDP {
		return errors.Errorf("UDP is the only supported protocol for RequestedTransport")
	}

	return nil
}
