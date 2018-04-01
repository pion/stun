package stun

import "github.com/pkg/errors"

// https://tools.ietf.org/html/rfc5766#section-14.7
// This attribute is used by the client to request a specific transport
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

type ProtocolNumber byte

const (
	ProtocolUDP ProtocolNumber = 0x11
)

type RequestedTransport struct {
	Protocol ProtocolNumber
}

func (r *RequestedTransport) Pack(message *Message) error {
	panic("*RequestedTransport Pack not implemented")
	return nil
}

func (r *RequestedTransport) Unpack(message *Message, rawAttribute *RawAttribute) error {
	r.Protocol = ProtocolNumber(rawAttribute.Value[0])
	if r.Protocol != ProtocolUDP {
		return errors.Errorf("UDP is the only supported protocol for RequestedTransport")
	}

	return nil
}
