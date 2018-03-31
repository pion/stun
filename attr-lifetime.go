package stun

import (
	"encoding/binary"

	"github.com/pkg/errors"
)

type Lifetime struct {
	Duration uint32
}

func (x *Lifetime) Pack(message *Message) (*RawAttribute, error) {
	ra := RawAttribute{
		Type:   AttrLifetime,
		Length: 4,
		Pad:    0,
	}
	v := make([]byte, 4)

	binary.BigEndian.PutUint32(v, x.Duration)

	return &ra, nil
}

func (x *Lifetime) Unpack(message *Message, rawAttribute *RawAttribute) error {
	v := rawAttribute.Value

	if len(v) != 4 {
		return errors.Errorf("invalid lifetime length %d != %d (expected)", len(v), 4)
	}

	x.Duration = binary.BigEndian.Uint32(v)

	return nil
}
