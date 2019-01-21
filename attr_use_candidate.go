package stun

// https://tools.ietf.org/html/draft-ietf-ice-rfc5245bis-20#section-16.1

// UseCandidate has no field struct
type UseCandidate struct {
}

// Pack with use-candidate attribute
func (u *UseCandidate) Pack(message *Message) error {
	message.AddAttribute(AttrUseCandidate, []byte{})
	return nil
}

// Unpack use-candidate attribute
func (u *UseCandidate) Unpack(message *Message, rawAttribute *RawAttribute) error {
	return nil
}
