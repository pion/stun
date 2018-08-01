// +build !debug

package stun

// CheckSize returns ErrAttrSizeInvalid if got is not equal to expected.
func CheckSize(_ AttrType, got, expected int) error {
	if got == expected {
		return nil
	}
	return ErrAttrSizeInvalid
}
