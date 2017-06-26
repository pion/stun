package stun

import (
	"errors"
	"testing"
)

func BenchmarkBuildOverhead(b *testing.B) {
	var (
		t        = BindingRequest
		username = NewUsername("username")
		nonce    = NewNonce("nonce")
		realm    = NewRealm("example.org")
	)
	b.Run("Build", func(b *testing.B) {
		b.ReportAllocs()
		m := new(Message)
		for i := 0; i < b.N; i++ {
			m.Build(&t, &username, &nonce, &realm, &Fingerprint)
		}
	})
	b.Run("BuildNonPointer", func(b *testing.B) {
		b.ReportAllocs()
		m := new(Message)
		for i := 0; i < b.N; i++ {
			m.Build(t, username, nonce, realm, Fingerprint)
		}
	})
	b.Run("Raw", func(b *testing.B) {
		b.ReportAllocs()
		m := new(Message)
		for i := 0; i < b.N; i++ {
			m.Reset()
			m.WriteHeader()
			m.SetType(t)
			username.AddTo(m)
			nonce.AddTo(m)
			realm.AddTo(m)
			Fingerprint.AddTo(m)
		}
	})
}

func TestMessage_Apply(t *testing.T) {
	var (
		integrity = NewShortTermIntegrity("password")
		decoded   = new(Message)
	)
	m, err := Build(BindingRequest, TransactionID,
		NewUsername("username"),
		NewNonce("nonce"),
		NewRealm("example.org"),
		integrity,
		Fingerprint,
	)
	if err != nil {
		t.Fatal("failed to build:", err)
	}
	if m.Check(Fingerprint, integrity); err != nil {
		t.Fatal(err)
	}
	if _, err := decoded.Write(m.Raw); err != nil {
		t.Fatal(err)
	}
	if !decoded.Equal(m) {
		t.Error("not equal")
	}
	if err := integrity.Check(decoded); err != nil {
		t.Fatal(err)
	}
}

type errReturner struct {
	Err error
}

func (e errReturner) AddTo(m *Message) error {
	return e.Err
}

func (e errReturner) Check(m *Message) error {
	return e.Err
}

func (e errReturner) GetFrom(m *Message) error {
	return e.Err
}

func TestHelpersErrorHandling(t *testing.T) {
	m := New()
	e := errReturner{Err: errors.New("tError")}
	if err := m.Build(e); err != e.Err {
		t.Error(err, "!=", e.Err)
	}
	if err := m.Check(e); err != e.Err {
		t.Error(err, "!=", e.Err)
	}
	if err := m.Parse(e); err != e.Err {
		t.Error(err, "!=", e.Err)
	}
	t.Run("MustBuild", func(t *testing.T) {
		t.Run("Positive", func(t *testing.T) {
			MustBuild(NewTransactionIDSetter(transactionID{}))
		})
		defer func() {
			if p := recover(); p != e.Err {
				t.Errorf("%s != %s",
					p, e.Err,
				)
			}
		}()
		MustBuild(e)
	})
}
