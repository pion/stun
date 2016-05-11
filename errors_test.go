package stun

import "testing"

func TestDecodeErr(t *testing.T) {
	err := newDecodeErr("parent", "children", "message")
	if !err.IsPlace(DecodeErrPlace{Parent: "parent", Children: "children"}) {
		t.Error("isPlace test failed")
	}
	if !err.IsPlaceParent("parent") {
		t.Error("parent test failed")
	}
	if !err.IsPlaceChildren("children") {
		t.Error("children test failed")
	}
	if err.Error() != "BadFormat for parent/children: message" {
		t.Error("bad Error string")
	}
}

func TestError_Error(t *testing.T) {
	if Error("error").Error() != "error" {
		t.Error("bad Error string")
	}
}
