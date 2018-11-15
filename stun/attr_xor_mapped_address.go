package stun

//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |x x x x x x x x|    Family     |         X-Port                |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                X-Address (Variable)
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

//XorMappedAddress https://tools.ietf.org/html/rfc5389#section-15.2
type XorMappedAddress struct {
	XorAddress
}

// Pack writes an XorMappedAddress into a message
func (x *XorMappedAddress) Pack(message *Message) error {
	v, err := x.packInner(message)
	if err != nil {
		return err
	}
	message.AddAttribute(AttrXORMappedAddress, v)
	return nil
}
