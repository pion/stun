// +build debug

package stun

// CheckSize returns *AttrLengthError if got is not equal to expected.
func CheckSize(a AttrType, got, expected int) error {
	if got == expected {
		return nil
	}
	return &AttrLengthErr{
		Got:      got,
		Expected: expected,
		Attr:     a,
	}
}
