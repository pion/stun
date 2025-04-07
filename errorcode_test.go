// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

//go:build !js
// +build !js

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
