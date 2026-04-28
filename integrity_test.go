// SPDX-FileCopyrightText: 2026 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package stun

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessageIntegrity_AddTo_Simple(t *testing.T) {
	integrity := NewLongTermIntegrity("user", "realm", "pass")
	expected, err := hex.DecodeString("8493fbc53ba582fb4c044c456bdc40eb")
	assert.NoError(t, err)
	assert.EqualValues(t, expected, integrity)
	t.Run("Check", func(t *testing.T) {
		m := new(Message)
		m.WriteHeader()
		assert.NoError(t, integrity.AddTo(m))
		NewSoftware("software").AddTo(m) //nolint:errcheck,gosec
		m.WriteHeader()
		dM := new(Message)
		dM.Raw = m.Raw
		assert.NoError(t, dM.Decode())
		assert.NoError(t, integrity.Check(dM))
		dM.Raw[24] += 12 // HMAC now invalid
		assert.Error(t, integrity.Check(dM))
	})
}

func TestMessageIntegrityWithFingerprint(t *testing.T) {
	msg := new(Message)
	msg.TransactionID = [TransactionIDSize]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	msg.WriteHeader()
	NewSoftware("software").AddTo(msg) //nolint:errcheck,gosec
	integrity := NewShortTermIntegrity("pwd")
	assert.Equal(t, "KEY: 0x707764", integrity.String())
	assert.NoError(t, integrity.AddTo(msg))
	assert.NoError(t, integrity.AddTo(msg))
	assert.NoError(t, integrity.Check(msg))
	assert.NoError(t, Fingerprint.AddTo(msg))
	assert.NoError(t, integrity.Check(msg))
	msg.Raw[24] = 33
	assert.Error(t, integrity.Check(msg))
}

func TestMessageIntegrity(t *testing.T) {
	m := new(Message)
	i := NewShortTermIntegrity("password")
	m.WriteHeader()
	assert.NoError(t, i.AddTo(m))
	_, err := m.Get(AttrMessageIntegrity)
	assert.NoError(t, err)
}

func TestMessageIntegrityBeforeFingerprint(t *testing.T) {
	m := new(Message)
	m.WriteHeader()
	Fingerprint.AddTo(m) //nolint:errcheck,gosec
	i := NewShortTermIntegrity("password")
	assert.Error(t, i.AddTo(m))
}

func TestAttributeAfterMessageIntegrity(t *testing.T) {
	m := new(Message)
	m.Type = BindingRequest
	m.WriteHeader()
	i := NewShortTermIntegrity("password")
	assert.NoError(t, i.AddTo(m))
	assert.NoError(t, NewSoftware("after").AddTo(m))
	assert.NoError(t, Fingerprint.AddTo(m))

	mDecoded := New()
	_, err := mDecoded.ReadFrom(bytes.NewReader(m.Raw))
	assert.NoError(t, err)

	assert.NoError(t, i.Check(mDecoded))
	assert.NoError(t, Fingerprint.Check(mDecoded))
	_, found := mDecoded.Attributes.Get(AttrSoftware)
	assert.Equal(t, found, !mDecoded.strict)
}

func TestAttributeAfterMessageIntegrityStrict(t *testing.T) {
	m := new(Message)
	m.Type = BindingRequest
	m.WriteHeader()
	i := NewShortTermIntegrity("password")
	assert.NoError(t, i.AddTo(m))
	assert.NoError(t, NewSoftware("after").AddTo(m))
	assert.NoError(t, Fingerprint.AddTo(m))

	mDecoded := NewWithOptions(WithStrict(true))
	_, err := mDecoded.ReadFrom(bytes.NewReader(m.Raw))
	assert.NoError(t, err)

	assert.NoError(t, i.Check(mDecoded))
	assert.NoError(t, Fingerprint.Check(mDecoded))
	_, found := mDecoded.Attributes.Get(AttrSoftware)
	assert.False(t, found)
}

func TestAttributeOrderingAfterMessageIntegritySHA256Strict(t *testing.T) {
	t.Run("MI256, SOFTWARE, FINGERPRINT", func(t *testing.T) {
		m := new(Message)
		m.Type = BindingRequest
		m.WriteHeader()
		m.Add(AttrMessageIntegritySHA256, make([]byte, 32))
		assert.NoError(t, NewSoftware("after").AddTo(m))
		assert.NoError(t, Fingerprint.AddTo(m))

		mDecoded := NewWithOptions(WithStrict(true))
		_, err := mDecoded.ReadFrom(bytes.NewReader(m.Raw))
		assert.NoError(t, err)

		_, foundSoftware := mDecoded.Attributes.Get(AttrSoftware)
		assert.False(t, foundSoftware)
		assert.NoError(t, Fingerprint.Check(mDecoded))
	})

	t.Run("MI256, MI, FINGERPRINT", func(t *testing.T) {
		m := new(Message)
		m.Type = BindingRequest
		m.WriteHeader()
		m.Add(AttrMessageIntegritySHA256, make([]byte, 32))
		m.Add(AttrMessageIntegrity, make([]byte, 20))
		assert.NoError(t, Fingerprint.AddTo(m))

		mDecoded := NewWithOptions(WithStrict(true))
		_, err := mDecoded.ReadFrom(bytes.NewReader(m.Raw))
		assert.NoError(t, err)

		_, foundMI := mDecoded.Attributes.Get(AttrMessageIntegrity)
		assert.False(t, foundMI)
		_, foundMI256 := mDecoded.Attributes.Get(AttrMessageIntegritySHA256)
		assert.True(t, foundMI256)
		assert.NoError(t, Fingerprint.Check(mDecoded))
	})

	t.Run("MI, MI256, FINGERPRINT", func(t *testing.T) {
		m := new(Message)
		m.Type = BindingRequest
		m.WriteHeader()
		m.Add(AttrMessageIntegrity, make([]byte, 20))
		m.Add(AttrMessageIntegritySHA256, make([]byte, 32))
		assert.NoError(t, Fingerprint.AddTo(m))

		mDecoded := NewWithOptions(WithStrict(true))
		_, err := mDecoded.ReadFrom(bytes.NewReader(m.Raw))
		assert.NoError(t, err)

		_, foundMI := mDecoded.Attributes.Get(AttrMessageIntegrity)
		assert.True(t, foundMI)
		_, foundMI256 := mDecoded.Attributes.Get(AttrMessageIntegritySHA256)
		assert.True(t, foundMI256)
		assert.NoError(t, Fingerprint.Check(mDecoded))
	})

	t.Run("MI, MI, FINGERPRINT", func(t *testing.T) {
		m := new(Message)
		m.Type = BindingRequest
		m.WriteHeader()
		m.Add(AttrMessageIntegrity, make([]byte, 20))
		m.Add(AttrMessageIntegrity, make([]byte, 20))
		assert.NoError(t, Fingerprint.AddTo(m))

		mDecoded := NewWithOptions(WithStrict(true))
		_, err := mDecoded.ReadFrom(bytes.NewReader(m.Raw))
		assert.NoError(t, err)

		count := 0
		for _, a := range mDecoded.Attributes {
			if a.Type == AttrMessageIntegrity {
				count++
			}
		}
		assert.Equal(t, 1, count)
		assert.NoError(t, Fingerprint.Check(mDecoded))
	})
}

func BenchmarkMessageIntegrity_AddTo(b *testing.B) {
	m := new(Message)
	integrity := NewShortTermIntegrity("password")
	m.WriteHeader()
	b.ReportAllocs()
	b.SetBytes(int64(len(m.Raw)))
	for i := 0; i < b.N; i++ {
		m.WriteHeader()
		assert.NoError(b, integrity.AddTo(m))
		m.Reset()
	}
}

func BenchmarkMessageIntegrity_Check(b *testing.B) {
	m := new(Message)
	m.Raw = make([]byte, 0, 1024)
	NewSoftware("software").AddTo(m) //nolint:errcheck,gosec
	integrity := NewShortTermIntegrity("password")
	b.ReportAllocs()
	m.WriteHeader()
	b.SetBytes(int64(len(m.Raw)))
	assert.NoError(b, integrity.AddTo(m))
	m.WriteLength()
	for i := 0; i < b.N; i++ {
		assert.NoError(b, integrity.Check(m))
	}
}
