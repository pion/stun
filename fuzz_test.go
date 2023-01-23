package stun

import (
	"encoding/binary"
	"errors"
	"testing"
)

func FuzzMessage(f *testing.F) {
	msg1 := New()
	msg2 := New()

	f.Add([]byte("00\x00\x000000000000000000"))
	f.Fuzz(func(t *testing.T, data []byte) {
		msg1.Reset()
		msg2.Reset()

		// Fuzzer does not know about cookies
		if len(data) >= 8 {
			binary.BigEndian.PutUint32(data[4:8], magicCookie)
		}

		// Trying to read data as message
		if _, err := msg1.Write(data); err != nil {
			return // We expect invalid messages to fail here
		}

		if _, err := msg2.Write(msg1.Raw); err != nil {
			t.Fatalf("Failed to write: %s", err)
		}

		if msg2.TransactionID != msg1.TransactionID {
			t.Fatal("Transaction ID mismatch")
		}

		if msg2.Type != msg1.Type {
			t.Fatal("Type mismatch")
		}

		if len(msg2.Attributes) != len(msg1.Attributes) {
			t.Fatal("Attributes length mismatch")
		}
	})
}

func FuzzType(f *testing.F) {
	f.Add([]byte("\x9c\xbe\x03"))
	f.Fuzz(func(t *testing.T, data []byte) {
		s := MessageType{}
		vt, _ := binary.Uvarint(data)
		v := uint16(vt) & 0x1fff // First 3 bits are empty
		s.ReadValue(v)
		v2 := s.Value()
		if v != v2 {
			t.Fatal("v != v2")
		}

		t2 := MessageType{}
		t2.ReadValue(v2)
		if t2 != s {
			t.Fatal("t2 != t")
		}
	})
}

func FuzzSetters(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
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

		attributes := attrs{
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

		firstByte := byte(0)
		if len(data) > 0 {
			firstByte = data[0]
		}

		a := attributes.pick(firstByte)
		value := data
		if len(data) > 1 {
			value = value[1:]
		}

		m1.WriteHeader()
		m1.Add(a.t, value)
		err := a.g.GetFrom(m1)
		if errors.Is(err, ErrAttributeNotFound) {
			t.Fatalf("Unexpected 404: %s", err)
		}
		if err != nil {
			return
		}

		m2.WriteHeader()
		if err = a.g.AddTo(m2); err != nil {
			// We allow decoding some text attributes
			// when their length is too big, but
			// not encoding.
			if !IsAttrSizeOverflow(err) {
				t.Fatal(err)
			}
			return
		}
		m3.WriteHeader()
		v, err := m2.Get(a.t)
		if err != nil {
			t.Fatal(err)
		}
		m3.Add(a.t, v)

		if !m2.Equal(m3) {
			t.Fatalf("Not equal: %s != %s", m2, m3)
		}
	})
}

func TestAttrPick(t *testing.T) {
	attributes := attrs{
		{new(XORMappedAddress), AttrXORMappedAddress},
	}
	for i := byte(0); i < 255; i++ {
		attributes.pick(i)
	}
}

type attr interface {
	Getter
	Setter
}

type attrs []struct {
	g attr
	t AttrType
}

func (a attrs) pick(v byte) struct {
	g attr
	t AttrType
} {
	idx := int(v) % len(a)
	return a[idx]
}
