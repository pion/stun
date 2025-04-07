// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

//go:build !js
// +build !js

package stun

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSoftware_GetFrom(t *testing.T) {
	msg := New()
	val := "Client v0.0.1"
	msg.Add(AttrSoftware, []byte(val))
	msg.WriteHeader()

	m2 := &Message{
		Raw: make([]byte, 0, 256),
	}
	software := new(Software)
	_, err := m2.ReadFrom(msg.reader())
	assert.NoError(t, err)
	assert.NoError(t, software.GetFrom(msg))
	assert.Equal(t, val, software.String())

	sAttr, ok := msg.Attributes.Get(AttrSoftware)
	assert.True(t, ok, "software attribute should be found")
	s := sAttr.String()
	assert.True(t, strings.HasPrefix(s, "SOFTWARE:"), "bad string representation")
}

func TestSoftware_AddTo_Invalid(t *testing.T) {
	m := New()
	s := make(Software, 1024)
	assert.True(t, IsAttrSizeOverflow(s.AddTo(m)), "AddTo should return *AttrOverflowErr")
	assert.ErrorIs(t, s.GetFrom(m), ErrAttributeNotFound)
}

func TestSoftware_AddTo_Regression(t *testing.T) {
	// s.AddTo checked len(m.Raw) instead of len(s.Raw).
	m := &Message{Raw: make([]byte, 2048)}
	s := make(Software, 100)
	assert.NoError(t, s.AddTo(m))
}

func BenchmarkUsername_AddTo(b *testing.B) {
	b.ReportAllocs()
	m := new(Message)
	u := Username("test")
	for i := 0; i < b.N; i++ {
		if err := u.AddTo(m); err != nil {
			b.Fatal(err)
		}
		m.Reset()
	}
}

func BenchmarkUsername_GetFrom(b *testing.B) {
	b.ReportAllocs()
	m := new(Message)
	Username("test").AddTo(m) //nolint:errcheck,gosec
	var u Username
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := u.GetFrom(m); err != nil {
			b.Fatal(err)
		}
		u = u[:0]
	}
}

func TestUsername(t *testing.T) {
	username := "username"
	uName := NewUsername(username)
	msg := new(Message)
	msg.WriteHeader()
	t.Run("Bad length", func(t *testing.T) {
		badU := make(Username, 600)
		assert.True(t, IsAttrSizeOverflow(badU.AddTo(msg)), "AddTo should return *AttrOverflowErr")
	})
	t.Run("AddTo", func(t *testing.T) {
		assert.NoError(t, uName.AddTo(msg))
		t.Run("GetFrom", func(t *testing.T) {
			got := new(Username)
			assert.NoError(t, got.GetFrom(msg))
			assert.Equal(t, username, got.String())
			t.Run("Not found", func(t *testing.T) {
				m := new(Message)
				u := new(Username)
				assert.ErrorIs(t, u.GetFrom(m), ErrAttributeNotFound)
			})
		})
	})
	t.Run("No allocations", func(t *testing.T) {
		m := new(Message)
		m.WriteHeader()
		u := NewUsername("username")
		assert.Empty(t, testing.AllocsPerRun(10, func() {
			assert.NoError(t, u.AddTo(m))
			m.Reset()
		}))
	})
}

func TestRealm_GetFrom(t *testing.T) {
	msg := New()
	val := "realm"
	msg.Add(AttrRealm, []byte(val))
	msg.WriteHeader()

	m2 := &Message{
		Raw: make([]byte, 0, 256),
	}
	r := new(Realm)
	assert.ErrorIs(t, r.GetFrom(m2), ErrAttributeNotFound)
	_, err := m2.ReadFrom(msg.reader())
	assert.NoError(t, err)
	assert.NoError(t, r.GetFrom(msg))
	assert.Equal(t, val, r.String())

	rAttr, ok := msg.Attributes.Get(AttrRealm)
	assert.True(t, ok, "realm attribute should be found")
	s := rAttr.String()
	assert.True(t, strings.HasPrefix(s, "REALM:"), "bad string representation")
}

func TestRealm_AddTo_Invalid(t *testing.T) {
	m := New()
	r := make(Realm, 1024)
	assert.True(t, IsAttrSizeOverflow(r.AddTo(m)), "AddTo should return *AttrOverflowErr")
	assert.ErrorIs(t, r.GetFrom(m), ErrAttributeNotFound)
}

func TestNonce_GetFrom(t *testing.T) {
	msg := New()
	val := "example.org"
	msg.Add(AttrNonce, []byte(val))
	msg.WriteHeader()

	m2 := &Message{
		Raw: make([]byte, 0, 256),
	}
	var nonce Nonce
	_, err := m2.ReadFrom(msg.reader())
	assert.NoError(t, err)
	assert.NoError(t, nonce.GetFrom(msg))
	assert.Equal(t, val, nonce.String())

	nAttr, ok := msg.Attributes.Get(AttrNonce)
	assert.True(t, ok, "nonce attribute should be found")
	s := nAttr.String()
	assert.True(t, strings.HasPrefix(s, "NONCE:"), "bad string representation")
}

func TestNonce_AddTo_Invalid(t *testing.T) {
	m := New()
	n := make(Nonce, 1024)
	assert.True(t, IsAttrSizeOverflow(n.AddTo(m)), "AddTo should return *AttrOverflowErr")
	assert.ErrorIs(t, n.GetFrom(m), ErrAttributeNotFound)
}

func TestNonce_AddTo(t *testing.T) {
	m := New()
	n := Nonce("example.org")
	assert.NoError(t, n.AddTo(m))
	v, err := m.Get(AttrNonce)
	assert.NoError(t, err)
	assert.Equal(t, "example.org", string(v))
}

func BenchmarkNonce_AddTo(b *testing.B) {
	b.ReportAllocs()
	m := New()
	n := NewNonce("nonce")
	for i := 0; i < b.N; i++ {
		if err := n.AddTo(m); err != nil {
			b.Fatal(err)
		}
		m.Reset()
	}
}

func BenchmarkNonce_AddTo_BadLength(b *testing.B) {
	b.ReportAllocs()
	m := New()
	n := make(Nonce, 2048)
	for i := 0; i < b.N; i++ {
		if err := n.AddTo(m); err == nil {
			b.Fatal("should error")
		}
		m.Reset()
	}
}

func BenchmarkNonce_GetFrom(b *testing.B) {
	b.ReportAllocs()
	m := New()
	n := NewNonce("nonce")
	n.AddTo(m) //nolint:errcheck,gosec
	for i := 0; i < b.N; i++ {
		n.GetFrom(m) //nolint:errcheck,gosec
	}
}
