// +build debug

package stun

import "testing"

func TestAttrOverflowErr_Error(t *testing.T) {
	err := AttrOverflowErr{
		Got:  100,
		Max:  50,
		Type: AttrLifetime,
	}
	if err.Error() != "incorrect length of LIFETIME attribute: 100 exceeds maximum 50" {
		t.Error("bad error string", err)
	}
}

func TestAttrLengthErr_Error(t *testing.T) {
	err := AttrLengthErr{
		Attr:     AttrErrorCode,
		Expected: 15,
		Got:      99,
	}
	if err.Error() != "incorrect length of ERROR-CODE attribute: got 99, expected 15" {
		t.Errorf("bad error string: %s", err)
	}
}
