package stun

import (
	"github.com/pkg/errors"
)

// https://tools.ietf.org/html/rfc5389#section-15.9
// The UNKNOWN-ATTRIBUTES attribute is present only in an error response
// when the response code in the ERROR-CODE attribute is 420.
//
// The attribute contains a list of 16-bit values, each of which
// represents an attribute type that was not understood by the server.
//
//     0                   1                   2                   3
//     0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
//    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//    |      Attribute 1 Type           |     Attribute 2 Type        |
//    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//    |      Attribute 3 Type           |     Attribute 4 Type    ...
//    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//
//
//           Figure 8: Format of UNKNOWN-ATTRIBUTES Attribute
//
//    Note: In [RFC3489], this field was padded to 32 by duplicating the
//    last attribute.  In this version of the specification, the normal
//    padding rules for attributes are used instead.

// UnknownAttributes has several attrTypes
type UnknownAttributes struct {
	Attributes []AttrType
}

const (
	unknownAttributesMax = 4
)

// Pack AttrUnknownAttributes
func (u *UnknownAttributes) Pack(message *Message) error {
	if len(u.Attributes) > unknownAttributesMax {
		return errors.Errorf("UnknownAttributes only supports up to 4 attributes")
	}

	var v [8]byte
	for i, attr := range u.Attributes {
		enc.PutUint16(v[i*2:], uint16(attr))
	}

	message.AddAttribute(AttrUnknownAttributes, v[0:])
	return nil
}

// Unpack always returns error
func (u *UnknownAttributes) Unpack(message *Message, rawAttribute *RawAttribute) error {
	return errors.Errorf("stun.UnknownAttributes Unpack not implemented")
}
