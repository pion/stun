package stun

import "github.com/pkg/errors"

// https://tools.ietf.org/html/rfc5389#section-15.10
// The SOFTWARE attribute contains a textual description of the software
//  being used by the agent sending the message

// Software struct has SOFTWARE field that rfc5389#section-15.10
type Software struct {
	Software string
}

const (
	softwareMaxLength = 763
)

// Pack with checking softwareMaxLength
func (s *Software) Pack(message *Message) error {
	if len([]byte(s.Software)) > softwareMaxLength {
		return errors.Errorf("invalid software length %d", len([]byte(s.Software)))
	}
	message.AddAttribute(AttrSoftware, []byte(s.Software))
	return nil
}

// Unpack Software field
func (s *Software) Unpack(message *Message, rawAttribute *RawAttribute) error {
	s.Software = string(rawAttribute.Value)
	return nil
}
