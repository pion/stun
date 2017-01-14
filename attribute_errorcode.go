package stun

// ErrorCode is code for ERROR-CODE attribute.
type ErrorCode int

// Possible error codes.
const (
	CodeTryAlternate     ErrorCode = 300
	CodeBadRequest       ErrorCode = 400
	CodeUnauthorised     ErrorCode = 401
	CodeUnknownAttribute ErrorCode = 420
	CodeStaleNonce       ErrorCode = 428
	CodeRoleConflict     ErrorCode = 478
	CodeServerError      ErrorCode = 500
)

var errorReasons = map[ErrorCode]string{
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
	reason, ok := errorReasons[c]
	if !ok {
		return "Unknown Error"
	}
	return reason
}
