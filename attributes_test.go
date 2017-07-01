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

func TestPadding(t *testing.T) {
	tt := []struct {
		in, out int
	}{
		{4, 4},   // 0
		{2, 4},   // 1
		{5, 8},   // 2
		{8, 8},   // 3
		{11, 12}, // 4
		{1, 4},   // 5
		{3, 4},   // 6
		{6, 8},   // 7
		{7, 8},   // 8
		{0, 0},   // 9
		{40, 40}, // 10
	}
	for i, c := range tt {
		if got := nearestPaddedValueLength(c.in); got != c.out {
			t.Errorf("[%d]: padd(%d) %d (got) != %d (expected)",
				i, c.in, got, c.out,
			)
		}
	}
}

func TestAttrLengthError_Error(t *testing.T) {
	err := AttrOverflowErr{
		Got:  100,
		Max:  50,
		Type: AttrLifetime,
	}
	if err.Error() != "incorrect length of LIFETIME attribute: 100 exceeds maximum 50" {
		t.Error("bad error string", err)
	}
}

func TestAttrLengthErr_Error(t *testing.T) {
	err := AttrLengthErr{
		Attr:     AttrErrorCode,
		Expected: 15,
		Got:      99,
	}
	if err.Error() != "incorrect length of ERROR-CODE attribute: got 99, expected 15" {
		t.Errorf("bad error string: %s", err)
	}
}
