// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

//go:build !debug
// +build !debug

package stun

import (
	"errors"

	"github.com/pion/stun/v3/internal/hmac"
)

// CheckSize returns ErrAttrSizeInvalid if got is not equal to expected.
func CheckSize(_ AttrType, got, expected int) error {
	if got == expected {
		return nil
	}

	return ErrAttributeSizeInvalid
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
	return errors.Is(err, ErrAttributeSizeInvalid)
}

// CheckOverflow returns ErrAttributeSizeOverflow if got is bigger that max.
func CheckOverflow(_ AttrType, got, maxVal int) error {
	if got <= maxVal {
		return nil
	}

	return ErrAttributeSizeOverflow
}

// IsAttrSizeOverflow returns true if error means that attribute size is too big.
func IsAttrSizeOverflow(err error) bool {
	return errors.Is(err, ErrAttributeSizeOverflow)
}
