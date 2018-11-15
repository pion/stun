package stun

import (
	"github.com/pkg/errors"
)

// IceControlled struct representated tiebreak
type IceControlled struct {
	TieBreaker uint64
}

// Pack with TieBreak
func (i *IceControlled) Pack(message *Message) error {
	v := make([]byte, 8)
	enc.PutUint64(v, i.TieBreaker)
	message.AddAttribute(AttrIceControlled, v)
	return nil
}

// Unpack with TieBreak
func (i *IceControlled) Unpack(message *Message, rawAttribute *RawAttribute) error {
	v := rawAttribute.Value

	if len(v) != 8 {
		return errors.Errorf("invalid TieBreaker length %d != %d (expected)", len(v), 8)
	}

	i.TieBreaker = enc.Uint64(v)

	return nil
}
