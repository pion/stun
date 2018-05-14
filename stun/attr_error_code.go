package stun

import (
	"github.com/pkg/errors"
)

// An ErrorCode attribute is used in error response messages.  It
// contains a numeric error code value in the range of 300 to 699 plus a
// textual reason phrase encoded in UTF-8
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |           Reserved, should be 0         |Class|     Number    |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |      Reason Phrase (variable)                                ..
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
type ErrorCode struct {
	ErrorClass  int
	ErrorNumber int
	Reason      []byte
}

var (
	// Err300TryAlternate a ErrorCode value - TryAlernate
	Err300TryAlternate = ErrorCode{3, 0, []byte("Try Alternate: The client should contact an alternate server for this request.")}
	// Err400BadRequest a ErrorCode value - BadRequest
	Err400BadRequest = ErrorCode{4, 0, []byte("Bad Request: The request was malformed.")}
	// Err401Unauthorized a ErrorCode value - Unauthorized
	Err401Unauthorized = ErrorCode{4, 1, []byte("Unauthorized: The request did not contain the correct credentials to proceed.")}
	// Err420UnknownAttributes a ErrorCode value - UnknownAttributes
	Err420UnknownAttributes = ErrorCode{4, 20, []byte("Unknown Attribute: The server received a STUN packet containing a comprehension-required attribute that it did not understand.")}
	// Err437AllocationMismatch a ErrorCode value - AllocationMismatch
	Err437AllocationMismatch = ErrorCode{4, 37, []byte("AllocationMismatch: 5-TUPLE didn't match, or conflicted with existing state.")}
	// Err438StaleNonce a ErrorCode value - StaleNonce
	Err438StaleNonce = ErrorCode{4, 38, []byte("Stale Nonce: The NONCE used by the client was no longer valid.")}
	// Err442UnsupportedTransportProtocol a ErrorCode value - UnsupportedTransportProtocol
	Err442UnsupportedTransportProtocol = ErrorCode{4, 42, []byte("Unsupported Transport Protocol: UDP is the only supported transport protocol.")}
	// Err500ServerError a ErrorCode value - ServerError
	Err500ServerError = ErrorCode{5, 0, []byte("Server Error: The server has suffered a temporary error.")}
	// Err508InsufficentCapacity a ErrorCode value - InsufficentCapacity
	Err508InsufficentCapacity = ErrorCode{5, 8, []byte("Insufficent Capacity: The server doesn't have the capacity to fulfill this request.")}
)

const (
	errorCodeHeaderLength    = 4
	errorCodeMaxReasonLength = 763
	errorCodeClassStart      = 2
	errorCodeNumberStart     = 3
	errorCodeReasonStart     = 4
)

// Pack a ErrorCode attribute, adding it to the passed message
func (e *ErrorCode) Pack(message *Message) error {
	if len(e.Reason) > errorCodeMaxReasonLength {
		return errors.Errorf("invalid reason length %d", len(e.Reason))
	}

	if e.ErrorClass < 3 || e.ErrorClass > 6 {
		return errors.Errorf("invalid error class %d", e.ErrorClass)
	}

	if e.ErrorNumber < 0 || e.ErrorNumber > 99 {
		return errors.Errorf("invalid error subcode %d", e.ErrorNumber)
	}

	len := errorCodeHeaderLength + len(e.Reason)

	v := make([]byte, len)

	v[errorCodeClassStart] = byte(e.ErrorClass)
	v[errorCodeNumberStart] = byte(e.ErrorNumber)

	copy(v[errorCodeReasonStart:], e.Reason)

	message.AddAttribute(AttrErrorCode, v)

	return nil
}

// Unpack a ErrorCode, deserializing the rawAttribute and populating the struct
func (e *ErrorCode) Unpack(message *Message, rawAttribute *RawAttribute) error {
	return errors.New("ErrorCode.Unpack() unimplemented")
}
