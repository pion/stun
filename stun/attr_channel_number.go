package stun

import (
	"github.com/pkg/errors"
)

// A ChannelNumber is a 4-byte header that identifies a Channel. Each channel
// number in use is bound to a specific peer and thus serves as a
// shorthand for the peer's host transport address.
// https://tools.ietf.org/html/rfc5766#section-2.5
type ChannelNumber struct {
	ChannelNumber uint16
}

// Pack a ChannelNumber attribute, adding it to the passed message
func (x *ChannelNumber) Pack(message *Message) error {
	v := make([]byte, 2)
	enc.PutUint16(v, x.ChannelNumber)
	message.AddAttribute(AttrChannelNumber, v)
	return nil
}

// Unpack a ChannelNumber, deserializing the rawAttribute and populating the struct
func (x *ChannelNumber) Unpack(message *Message, rawAttribute *RawAttribute) error {
	v := rawAttribute.Value

	if len(v) != 4 {
		return errors.Errorf("invalid channel number length %d != %d (expected)", len(v), 2)
	}

	x.ChannelNumber = enc.Uint16(v[:2])

	return nil
}
