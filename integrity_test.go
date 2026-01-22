// SPDX-FileCopyrightText: 2026 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package stun

import (
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
