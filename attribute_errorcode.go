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

// Reason returns recommended reason string.
func (c ErrorCode) Reason() string {
	switch c {
	case CodeTryAlternate:
		return "Try Alternate"
	case CodeBadRequest:
		return "Bad Request"
	case CodeUnauthorised:
		return "Unauthorised"
	case CodeUnknownAttribute:
		return "Unknown attribute"
	case CodeStaleNonce:
		return "Stale Nonce"
	case CodeServerError:
		return "Server Error"
	case CodeRoleConflict:
		return "Role conflict"
	default:
		return "Unknown Error"
	}
}
