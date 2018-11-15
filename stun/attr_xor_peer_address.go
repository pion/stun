package stun

// The XOR-PEER-ADDRESS specifies the address and port of the peer as
// seen from the TURN server.  (For example, the peer's server-reflexive
// transport address if the peer is behind a NAT.)  It is encoded in the
// same way as XOR-MAPPED-ADDRESS [RFC5389].

//XorPeerAddress include XorAddress which encoded in the same way as XOR-MAPPED-ADDRESS [RFC5389]
type XorPeerAddress struct {
	XorAddress
}

//Pack using XOR-PEER-ADDRESS
func (x *XorPeerAddress) Pack(message *Message) error {
	v, err := x.packInner(message)
	if err != nil {
		return err
	}
	message.AddAttribute(AttrXORPeerAddress, v)
	return nil
}
