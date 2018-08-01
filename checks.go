// +build !debug

package stun

import "github.com/gortc/stun/internal/hmac"

// CheckSize returns ErrAttrSizeInvalid if got is not equal to expected.
func CheckSize(_ AttrType, got, expected int) error {
	if got == expected {
		return nil
	}
	return ErrAttrSizeInvalid
}

func checkHMAC(got, expected []byte) error {
	if hmac.Equal(got, expected) {
		return nil
	}
	return ErrIntegrityMismatch
}

func checkFingerprint(got, expected uint32) error {
	if got == expected {
		return nil
	}
	return ErrFingerprintMismatch
}

// IsAttrSizeInvalid returns true if error means that attribute size is invalid.
func IsAttrSizeInvalid(err error) bool {
	return err == ErrAttrSizeInvalid
}
