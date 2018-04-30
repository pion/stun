package stun

// A Data represents the attribute is present in all Send and Data indications.  The
// value portion of this attribute is variable length and consists of
// the application data (that is, the data that would immediately follow
// the UDP header if the data was been sent directly between the client
// and the peer).  If the length of this attribute is not a multiple of
// 4, then padding must be added after this attribute.
type Data struct {
	Data []byte
}

// Pack a Data attribute, adding it to the passed message
func (d *Data) Pack(message *Message) error {
	message.AddAttribute(AttrData, d.Data)
	return nil
}

// Unpack a Data, deserializing the rawAttribute and populating the struct
func (d *Data) Unpack(message *Message, rawAttribute *RawAttribute) error {
	d.Data = rawAttribute.Value
	return nil
}
