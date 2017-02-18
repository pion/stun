package stun

import (
	"testing"
)

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
	Username("test").AddTo(m)
	for i := 0; i < b.N; i++ {
		var u Username
		if err := u.GetFrom(m); err != nil {
			b.Fatal(err)
		}
	}
}

func TestUsername(t *testing.T) {
	username := "username"
	u := NewUsername(username)
	m := new(Message)
	m.WriteHeader()
	t.Run("Bad length", func(t *testing.T) {
		badU := make(Username, 600)
		if err := badU.AddTo(m); err != ErrUsernameTooBig {
			t.Errorf("expected %s, got %v", ErrUsernameTooBig, err)
		}
	})
	t.Run("AddTo", func(t *testing.T) {
		if err := u.AddTo(m); err != nil {
			t.Error("errored:", err)
		}
		t.Run("GetFrom", func(t *testing.T) {
			got := new(Username)
			if err := got.GetFrom(m); err != nil {
				t.Error("errored:", err)
			}
			if got.String() != username {
				t.Errorf("expedted: %s, got: %s", username, got)
			}
			t.Run("Not found", func(t *testing.T) {
				m := new(Message)
				u := new(Username)
				if err := u.GetFrom(m); err != ErrAttributeNotFound {
					t.Error("Should error")
				}
			})
		})
	})
	t.Run("No allocations", func(t *testing.T) {
		m := new(Message)
		m.WriteHeader()
		u := NewUsername("username")
		if allocs := testing.AllocsPerRun(10, func() {
			if err := u.AddTo(m); err != nil {
				t.Error(err)
			}
			m.Reset()
		}); allocs > 0 {
			t.Errorf("got %f allocations, zero expected", allocs)
		}
	})
}
