package stun

import "testing"

func TestDecodeErr_IsInvalidCookie(t *testing.T) {
	m := new(Message)
	m.WriteHeader()
	decoded := new(Message)
	m.Raw[4] = 55
	_, err := decoded.Write(m.Raw)
	if err == nil {
		t.Fatal("should error")
	}
	expected := "BadFormat for message/cookie: " +
		"3712a442 is invalid magic cookie (should be 2112a442)"
	if err.Error() != expected {
		t.Error(err, "!=", expected)
	}
	dErr, ok := err.(*DecodeErr)
	if !ok {
		t.Error("not decode error")
	}
	if !dErr.IsInvalidCookie() {
		t.Error("IsInvalidCookie = false, should be true")
	}
	if !dErr.IsPlaceChildren("cookie") {
		t.Error("bad children")
	}
	if !dErr.IsPlaceParent("message") {
		t.Error("bad parent")
	}
}
