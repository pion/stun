// SPDX-FileCopyrightText: 2026 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package stun

import (
	"errors"
	"fmt"
	"testing"

	"github.com/pion/stun/v3/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func BenchmarkBuildOverhead(b *testing.B) {
	var (
		msgType  = BindingRequest
		username = NewUsername("username")
		nonce    = NewNonce("nonce")
		realm    = NewRealm("example.org")
	)
	b.Run("Build", func(b *testing.B) {
		b.ReportAllocs()
		m := new(Message)
		for i := 0; i < b.N; i++ {
			m.Build(&msgType, &username, &nonce, &realm, &Fingerprint) //nolint:errcheck,gosec
		}
	})
	b.Run("BuildNonPointer", func(b *testing.B) {
		b.ReportAllocs()
		m := new(Message)
		for i := 0; i < b.N; i++ {
			m.Build(msgType, username, nonce, realm, Fingerprint) //nolint:errcheck,gosec //nolint:errcheck,gosec
		}
	})
	b.Run("Raw", func(b *testing.B) {
		b.ReportAllocs()
		m := new(Message)
		for i := 0; i < b.N; i++ {
			m.Reset()
			m.WriteHeader()
			m.SetType(msgType)
			username.AddTo(m)    //nolint:errcheck,gosec
			nonce.AddTo(m)       //nolint:errcheck,gosec
			realm.AddTo(m)       //nolint:errcheck,gosec
			Fingerprint.AddTo(m) //nolint:errcheck,gosec
		}
	})
}

func TestMessage_Apply(t *testing.T) {
	var (
		integrity = NewShortTermIntegrity("password")
		decoded   = new(Message)
	)
	msg, err := Build(BindingRequest, TransactionID,
		NewUsername("username"),
		NewNonce("nonce"),
		NewRealm("example.org"),
		integrity,
		Fingerprint,
	)
	assert.NoError(t, err, "failed to build")
	assert.NoError(t, msg.Check(Fingerprint, integrity))
	_, err = decoded.Write(msg.Raw)
	assert.NoError(t, err)
	assert.True(t, decoded.Equal(msg))
	assert.NoError(t, integrity.Check(decoded))
}

type errReturner struct {
	Err error
}

var errTError = errors.New("tError")

func (e errReturner) AddTo(*Message) error {
	return e.Err
}

func (e errReturner) Check(*Message) error {
	return e.Err
}

func (e errReturner) GetFrom(*Message) error {
	return e.Err
}

func TestHelpersErrorHandling(t *testing.T) {
	m := New()
	errReturn := errReturner{Err: errTError}
	assert.ErrorIs(t, m.Build(errReturn), errReturn.Err)
	assert.ErrorIs(t, m.Check(errReturn), errReturn.Err)
	assert.ErrorIs(t, m.Parse(errReturn), errReturn.Err)
	t.Run("MustBuild", func(t *testing.T) {
		t.Run("Positive", func(*testing.T) {
			MustBuild(NewTransactionIDSetter(transactionID{}))
		})
		defer func() {
			p, ok := recover().(error)
			assert.True(t, ok)
			assert.ErrorIs(t, p, errReturn.Err)
		}()
		MustBuild(errReturn)
	})
}

func TestMessage_ForEach(t *testing.T) { //nolint:cyclop
	initial := New()
	assert.NoError(t, initial.Build(
		NewRealm("realm1"), NewRealm("realm2"),
	))
	newMessage := func() *Message {
		m := New()
		assert.NoError(t, m.Build(
			NewRealm("realm1"), NewRealm("realm2"),
		))

		return m
	}
	t.Run("NoResults", func(t *testing.T) {
		m := newMessage()
		assert.True(t, m.Equal(initial), "m should be equal to initial")
		assert.NoError(t, m.ForEach(AttrUsername, func(*Message) error {
			assert.Fail(t, "should not be called")

			return nil
		}))
		assert.True(t, m.Equal(initial), "m should be equal to initial")
	})
	t.Run("ReturnOnError", func(t *testing.T) {
		m := newMessage()
		var calls int
		err := m.ForEach(AttrRealm, func(*Message) error {
			if calls > 0 {
				assert.Fail(t, "called multiple times")
			}
			calls++

			return ErrAttributeNotFound
		})
		assert.ErrorIs(t, err, ErrAttributeNotFound)
		assert.True(t, m.Equal(initial), "m should be equal to initial")
	})
	t.Run("Positive", func(t *testing.T) {
		msg := newMessage()
		var realms []string
		assert.NoError(t, msg.ForEach(AttrRealm, func(m *Message) error {
			var realm Realm
			assert.NoError(t, realm.GetFrom(m))
			realms = append(realms, realm.String())

			return nil
		}))
		assert.Len(t, realms, 2)
		assert.Equal(t, "realm1", realms[0], "bad value for 1 realm")
		assert.Equal(t, "realm2", realms[1], "bad value for 2 realm")
		assert.True(t, msg.Equal(initial), "m should be equal to initial")
		t.Run("ZeroAlloc", func(t *testing.T) {
			msg = newMessage()
			var realm Realm
			testutil.ShouldNotAllocate(t, func() {
				assert.NoError(t, msg.ForEach(AttrRealm, realm.GetFrom))
			})
			assert.True(t, msg.Equal(initial), "m should be equal to initial")
		})
	})
}

func ExampleMessage_ForEach() {
	m := MustBuild(NewRealm("realm1"), NewRealm("realm2"))
	if err := m.ForEach(AttrRealm, func(m *Message) error {
		var r Realm
		if err := r.GetFrom(m); err != nil {
			return err
		}
		fmt.Println(r)

		return nil
	}); err != nil {
		fmt.Println("error:", err)
	}

	// Output:
	// realm1
	// realm2
}

func BenchmarkMessage_ForEach(b *testing.B) {
	b.ReportAllocs()
	m := MustBuild(
		NewRealm("realm1"),
		NewRealm("realm2"),
		NewRealm("realm3"),
		NewRealm("realm4"),
	)
	for i := 0; i < b.N; i++ {
		if err := m.ForEach(AttrRealm, func(*Message) error {
			return nil
		}); err != nil {
			b.Fatal(err)
		}
	}
}
