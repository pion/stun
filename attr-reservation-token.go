package stun

import "github.com/pkg/errors"

// https://tools.ietf.org/html/rfc5766#sect
// The RESERVATION-TOKEN attribute contains a token that uniquely
// identifies a relayed transport address being held in reserve by the
// server.  The server includes this attribute in a success response to
// tell the client about the token, and the client includes this
// attribute in a subsequent Allocate request to request the server use
// that relayed transport address for the allocation.
//
// The attribute value is 8 bytes and contains the token value.
type ReservationToken struct {
	ReservationToken string
}

const (
	reservationTokenMaxLength = 8
)

func (r *ReservationToken) Pack(message *Message) error {
	if len([]byte(r.ReservationToken)) > reservationTokenMaxLength {
		return errors.Errorf("invalid ReservationToken length %d", len([]byte(r.ReservationToken)))
	}
	message.AddAttribute(AttrReservationToken, []byte(r.ReservationToken))
	return nil
}

func (r *ReservationToken) Unpack(message *Message, rawAttribute *RawAttribute) error {
	r.ReservationToken = string(rawAttribute.Value)
	return nil
}
