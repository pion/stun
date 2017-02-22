package stun

import "testing"

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
