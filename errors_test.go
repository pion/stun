// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package stun

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeErr_IsInvalidCookie(t *testing.T) {
	m := new(Message)
	m.WriteHeader()
	decoded := new(Message)
	m.Raw[4] = 55
	_, err := decoded.Write(m.Raw)
	assert.Error(t, err, "should error")
	expected := "BadFormat for message/cookie: " +
		"3712a442 is invalid magic cookie (should be 2112a442)"
	assert.Equal(t, expected, err.Error(), "error message mismatch")
	var dErr *DecodeErr
	assert.True(t, errors.As(err, &dErr), "not decode error")
	assert.True(t, dErr.IsInvalidCookie(), "IsInvalidCookie = false, should be true")
	assert.True(t, dErr.IsPlaceChildren("cookie"), "bad children")
	assert.True(t, dErr.IsPlaceParent("message"), "bad parent")
}
