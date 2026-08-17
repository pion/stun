// SPDX-FileCopyrightText: 2026 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

//go:build !js

package stun

import (
	"encoding/base64"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkErrorCode_AddTo(b *testing.B) {
	m := New()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		CodeStaleNonce.AddTo(m) //nolint:errcheck,gosec
		m.Reset()
	}
}

func BenchmarkErrorCodeAttribute_AddTo(b *testing.B) {
	m := New()
	b.ReportAllocs()
	a := &ErrorCodeAttribute{
		Code:   404,
		Reason: []byte("not found!"),
	}
	for i := 0; i < b.N; i++ {
		a.AddTo(m) //nolint:errcheck,gosec
		m.Reset()
	}
}

func BenchmarkErrorCodeAttribute_GetFrom(b *testing.B) {
	m := New()
	b.ReportAllocs()
	a := &ErrorCodeAttribute{
		Code:   404,
		Reason: []byte("not found!"),
	}
	a.AddTo(m) //nolint:errcheck,gosec
	for i := 0; i < b.N; i++ {
		a.GetFrom(m) //nolint:errcheck,gosec
	}
}

func TestErrorCodeAttribute_GetFrom(t *testing.T) {
	m := New()
	m.Add(AttrErrorCode, []byte{1})
	c := new(ErrorCodeAttribute)
	assert.ErrorIs(t, c.GetFrom(m), io.ErrUnexpectedEOF)
}

func TestMessage_AddErrorCode(t *testing.T) {
	m := New()
	m.Type = BindingError
	transactionID, err := base64.StdEncoding.DecodeString("jxhBARZwX+rsC6er")
	assert.NoError(t, err)
	copy(m.TransactionID[:], transactionID)
	expectedCode := ErrorCode(438)
	expectedReason := "Stale Nonce"
	CodeStaleNonce.AddTo(m) //nolint:errcheck,gosec
	m.WriteHeader()

	mRes := New()
	_, err = mRes.ReadFrom(m.reader())
	assert.NoError(t, err)
	errCodeAttr := new(ErrorCodeAttribute)
	assert.NoError(t, errCodeAttr.GetFrom(mRes))
	code := errCodeAttr.Code
	assert.Equal(t, expectedCode, code, "bad code")
	assert.Equal(t, expectedReason, string(errCodeAttr.Reason), "bad reason")
}

func TestErrorCode(t *testing.T) {
	attr := &ErrorCodeAttribute{
		Code:   404,
		Reason: []byte("not found!"),
	}
	assert.Equal(t, "404: not found!", attr.String(), "bad string")
	m := New()
	cod := ErrorCode(666)
	assert.ErrorIs(t, cod.AddTo(m), ErrNoDefaultReason, "should be ErrNoDefaultReason")
	assert.Error(t, attr.GetFrom(m), "attr should not be in message")
	attr.Reason = make([]byte, 2048)
	assert.Error(t, attr.AddTo(m), "should error")
}

func TestErrorCodeAttribute_AddToInvalidCode(t *testing.T) {
	m := New()

	assert.ErrorIs(t, (&ErrorCodeAttribute{Code: -1}).AddTo(m), errInvalidErrorCode)
	assert.ErrorIs(t, (&ErrorCodeAttribute{Code: 25600}).AddTo(m), errInvalidErrorCode)
}

func TestTurnError(t *testing.T) {
	te := TurnError{
		StunMessageType: NewType(MethodCreatePermission, ClassErrorResponse),
		ErrorCodeAttr: ErrorCodeAttribute{
			Code:   CodeForbidden,
			Reason: []byte("Forbidden"),
		},
	}
	expected := "CreatePermission error response (error 403: Forbidden)"
	assert.Equal(t, expected, te.Error())
	assert.Equal(t, expected, te.String())
}

func TestErrorCodeAttribute_ReservedBitsIgnored(t *testing.T) {
	// RFC 5389 §15.6 / RFC 8489 §14.8: the 5 bits above Class are Reserved and
	// receivers MUST ignore them. Class is only the low 3 bits of the byte.
	m := New()
	m.WriteHeader()
	// 0xE4 has reserved bits set; low 3 bits = 4 (class 4), number 20 => code 420.
	m.Add(AttrErrorCode, []byte{0x00, 0x00, 0xE4, 20})
	var c ErrorCodeAttribute
	assert.NoError(t, c.GetFrom(m))
	assert.Equal(t, ErrorCode(420), c.Code, "reserved bits not ignored")
}
