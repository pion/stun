package stun

type ErrorCodeAttribute struct {
	Code   int
	Reason []byte
}

// ErrorCode is code for ERROR-CODE attribute.
type ErrorCode int

// Possible error codes.
const (
	CodeTryAlternate     = 300
	CodeBadRequest       = 400
	CodeUnauthorised     = 401
	CodeUnknownAttribute = 420
	CodeStaleNonce       = 428
	CodeRoleConflict     = 478
	CodeServerError      = 500
)

var errorReasons = map[int]string{
	CodeTryAlternate:     "Try Alternate",
	CodeBadRequest:       "Bad Request",
	CodeUnauthorised:     "Unauthorised",
	CodeUnknownAttribute: "Unknown Attribute",
	CodeStaleNonce:       "Stale Nonce",
	CodeServerError:      "Server Error",
	CodeRoleConflict:     "Role Conflict",
}

// Reason returns recommended reason string.
func (c ErrorCode) Reason() string {
	reason, ok := errorReasons[int(c)]
	if !ok {
		return "Unknown Error"
	}
	return reason
}
