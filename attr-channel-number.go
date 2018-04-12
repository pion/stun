package stun

import (
	"encoding/binary"

	"github.com/pkg/errors"
)

type ChannelNumber struct {
	ChannelNumber uint16
}

func (x *ChannelNumber) Pack(message *Message) error {
	v := make([]byte, 2)
	binary.BigEndian.PutUint16(v, x.ChannelNumber)
	message.AddAttribute(AttrChannelNumber, v)
	return nil
}

func (x *ChannelNumber) Unpack(message *Message, rawAttribute *RawAttribute) error {
	v := rawAttribute.Value

	if len(v) != 4 {
		return errors.Errorf("invalid channel number length %d != %d (expected)", len(v), 2)
	}

	x.ChannelNumber = binary.BigEndian.Uint16(v[:2])

	return nil
}
