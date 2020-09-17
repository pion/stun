// +build !js

package stun

import (
	"encoding/base64"
	"errors"
	"io"
	"testing"
)

func BenchmarkErrorCode_AddTo(b *testing.B) {
	m := New()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		CodeStaleNonce.AddTo(m) // nolint:errcheck
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
		a.AddTo(m) // nolint:errcheck
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
	a.AddTo(m) // nolint:errcheck
	for i := 0; i < b.N; i++ {
		a.GetFrom(m) // nolint:errcheck
	}
}

func TestErrorCodeAttribute_GetFrom(t *testing.T) {
	m := New()
	m.Add(AttrErrorCode, []byte{1})
	c := new(ErrorCodeAttribute)
	if err := c.GetFrom(m); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("GetFrom should return <%s>, but got <%s>",
			io.ErrUnexpectedEOF, err,
		)
	}
}

func TestMessage_AddErrorCode(t *testing.T) {
	m := New()
	transactionID, err := base64.StdEncoding.DecodeString("jxhBARZwX+rsC6er")
	if err != nil {
		t.Error(err)
	}
	copy(m.TransactionID[:], transactionID)
	expectedCode := ErrorCode(438)
	expectedReason := "Stale Nonce"
	CodeStaleNonce.AddTo(m) // nolint:errcheck
	m.WriteHeader()

	mRes := New()
	if _, err = mRes.ReadFrom(m.reader()); err != nil {
		t.Fatal(err)
	}
	errCodeAttr := new(ErrorCodeAttribute)
	if err = errCodeAttr.GetFrom(mRes); err != nil {
		t.Error(err)
	}
	code := errCodeAttr.Code
	if err != nil {
		t.Error(err)
	}
	if code != expectedCode {
		t.Error("bad code", code)
	}
	if string(errCodeAttr.Reason) != expectedReason {
		t.Error("bad reason", string(errCodeAttr.Reason))
	}
}

func TestErrorCode(t *testing.T) {
	a := &ErrorCodeAttribute{
		Code:   404,
		Reason: []byte("not found!"),
	}
	if a.String() != "404: not found!" {
		t.Error("bad string", a)
	}
	m := New()
	cod := ErrorCode(666)
	if err := cod.AddTo(m); !errors.Is(err, ErrNoDefaultReason) {
		t.Error("should be ErrNoDefaultReason", err)
	}
	if err := a.GetFrom(m); err == nil {
		t.Error("attr should not be in message")
	}
	a.Reason = make([]byte, 2048)
	if err := a.AddTo(m); err == nil {
		t.Error("should error")
	}
}
