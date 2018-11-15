package stun

import (
	"github.com/pkg/errors"
)

// A EvenPort attribute allows the client to request that the port in the
// relayed transport address be even, and (optionally) that the server
// reserve the next-higher port number.  The value portion of this
// attribute is 1 byte long.
type EvenPort struct {
	ReserveAdditional bool
}

// Pack a EvenNumber attribute, adding it to the passed message
func (e *EvenPort) Pack(message *Message) error {
	return errors.Errorf("*EvenPort.Pack has not been implemented")
}

// Unpack a EvenPort, deserializing the rawAttribute and populating the struct
func (e *EvenPort) Unpack(message *Message, rawAttribute *RawAttribute) error {
	e.ReserveAdditional = rawAttribute.Value[0] != 0
	return nil
}
