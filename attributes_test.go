package stun

import (
	"testing"
)

func BenchmarkMessage_GetNotFound(b *testing.B) {
	m := New()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m.Get(AttrRealm)
	}
}

func TestMessage_GetNoAllocs(t *testing.T) {
	m := New()
	NewSoftware("c").AddTo(m)
	m.WriteHeader()

	t.Run("Default", func(t *testing.T) {
		allocs := testing.AllocsPerRun(10, func() {
			m.Get(AttrSoftware)
		})
		if allocs > 0 {
			t.Error("allocated memory, but should not")
		}
	})
	t.Run("Not found", func(t *testing.T) {
		allocs := testing.AllocsPerRun(10, func() {
			m.Get(AttrOrigin)
		})
		if allocs > 0 {
			t.Error("allocated memory, but should not")
		}
	})
}
