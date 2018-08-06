package stun

type UseCandidate struct {
}

func (u *UseCandidate) Pack(message *Message) error {
	message.AddAttribute(AttrUseCandidate, []byte{})
	return nil
}

func (u *UseCandidate) Unpack(message *Message, rawAttribute *RawAttribute) error {
	return nil
}
