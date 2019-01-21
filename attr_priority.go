package stun

import (
	"github.com/pkg/errors"
)

// Priority is a STUN Priority message
type Priority struct {
	Priority uint32
}

// Pack a STUN Priority message
func (p *Priority) Pack(message *Message) error {
	v := make([]byte, 4)
	enc.PutUint32(v, p.Priority)
	message.AddAttribute(AttrPriority, v)
	return nil
}

// Unpack a STUN Priority message
func (p *Priority) Unpack(message *Message, rawAttribute *RawAttribute) error {
	v := rawAttribute.Value

	if len(v) != 4 {
		return errors.Errorf("invalid priority length %d != %d (expected)", len(v), 4)
	}

	p.Priority = enc.Uint32(v)

	return nil
}
