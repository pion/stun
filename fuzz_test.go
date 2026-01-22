// SPDX-FileCopyrightText: 2026 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package stun

import (
	"encoding/binary"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func FuzzMessage(f *testing.F) {
	msg1 := New()

	f.Fuzz(func(t *testing.T, data []byte) {
		msg1.Reset()

		// Fuzzer does not know about cookies
		if len(data) >= 8 {
			binary.BigEndian.PutUint32(data[4:8], magicCookie)
		}

		// Trying to read data as message
		if _, err := msg1.Write(data); err != nil {
			return // We expect invalid messages to fail here
		}

		msg2 := New()
		_, err := msg2.Write(msg1.Raw)
		assert.NoError(t, err, "Failed to write")

		assert.Equal(t, msg1.TransactionID, msg2.TransactionID, "Transaction ID mismatch")
		assert.Equal(t, msg1.Type, msg2.Type, "Type mismatch")
		assert.Equal(t, len(msg1.Attributes), len(msg2.Attributes), "Attributes length mismatch")
	})
}

func FuzzType(f *testing.F) {
	f.Fuzz(func(t *testing.T, data uint16) {
		v := data & 0x1fff // First 3 bits are empty

		t1 := MessageType{}
		t1.ReadValue(v)
		v2 := t1.Value()
		assert.Equal(t, v, v2, "v != v2")

		t2 := MessageType{}
		t2.ReadValue(v2)
		assert.Equal(t, t1, t2, "t2 != t1")
	})
}

func FuzzSetters(f *testing.F) {
	f.Fuzz(func(t *testing.T, firstByte byte, value []byte) {
		var (
			m1 = &Message{
				Raw: make([]byte, 0, 2048),
			}
			m2 = &Message{
				Raw: make([]byte, 0, 2048),
			}
			m3 = &Message{
				Raw: make([]byte, 0, 2048),
			}
		)

		attrs := attributes{
			{new(Realm), AttrRealm},
			{new(XORMappedAddress), AttrXORMappedAddress},
			{new(Nonce), AttrNonce},
			{new(Software), AttrSoftware},
			{new(AlternateServer), AttrAlternateServer},
			{new(ErrorCodeAttribute), AttrErrorCode},
			{new(UnknownAttributes), AttrUnknownAttributes},
			{new(Username), AttrUsername},
			{new(MappedAddress), AttrMappedAddress},
			{new(Realm), AttrRealm},
		}
		attr := attrs.pick(firstByte)

		m1.WriteHeader()
		m1.Add(attr.t, value)
		err := attr.g.GetFrom(m1)
		assert.False(t, errors.Is(err, ErrAttributeNotFound))
		if err != nil {
			return
		}

		m2.WriteHeader()
		err = attr.g.AddTo(m2)
		if err != nil {
			// We allow decoding some text attributes
			// when their length is too big, but
			// not encoding.
			if !IsAttrSizeOverflow(err) {
				assert.NoError(t, err)
			}

			return
		}

		m3.WriteHeader()
		v, err := m2.Get(attr.t)
		assert.NoError(t, err)
		m3.Add(attr.t, v)

		assert.True(t, m2.Equal(m3), "Not equal: %s != %s", m2, m3)
	})
}

func TestAttrPick(*testing.T) {
	attrs := attributes{
		{new(XORMappedAddress), AttrXORMappedAddress},
	}

	for i := byte(0); i < 255; i++ {
		attrs.pick(i)
	}
}

type attr interface {
	Getter
	Setter
}

type attributes []struct {
	g attr
	t AttrType
}

func (a attributes) pick(v byte) struct {
	g attr
	t AttrType
} {
	idx := int(v) % len(a)

	return a[idx]
}
