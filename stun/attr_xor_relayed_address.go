package stun

// https://tools.ietf.org/html/rfc5766#section-14.5
// The XOR-RELAYED-ADDRESS is present in Allocate responses.  It
// specifies the address and port that the server allocated to the
// client.  It is encoded in the same way as XOR-MAPPED-ADDRESS
// [RFC5389]

//XorRelayedAddress include XorAddress which encoded in the same way as XOR-MAPPED-ADDRESS [RFC5389]
type XorRelayedAddress struct {
	XorAddress
}

//Pack using XOR-RELAYED-ADDRESS
func (x *XorRelayedAddress) Pack(message *Message) error {
	v, err := x.packInner(message)
	if err != nil {
		return err
	}
	message.AddAttribute(AttrXORRelayedAddress, v)
	return nil
}
