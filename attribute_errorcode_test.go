package stun

import "testing"

func TestErrorCode_Reason(t *testing.T) {
	codes := [...]ErrorCode{
		CodeTryAlternate,
		CodeBadRequest,
		CodeUnauthorised,
		CodeUnknownAttribute,
		CodeStaleNonce,
		CodeRoleConflict,
		CodeServerError,
	}
	for _, code := range codes {
		if code.Reason() == "Unknown Error" {
			t.Error(code, "should not be unknown")
		}
		if len(code.Reason()) == 0 {
			t.Error(code, "should not be blank")
		}
	}
	if ErrorCode(999).Reason() != "Unknown Error" {
		t.Error("999 error should be Unknown")
	}
}
