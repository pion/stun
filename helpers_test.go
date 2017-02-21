package stun

import "testing"

func TestMessage_Apply(t *testing.T) {
	var (
		integrity = NewShortTermIntegrity("password")
		decoded = new(Message)
	)
	m, err := Build(
		NewType(ClassRequest, MethodBinding),
		TransactionID,
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
