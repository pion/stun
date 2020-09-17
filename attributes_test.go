package stun

import (
	"bytes"
	"testing"
)

func BenchmarkMessage_GetNotFound(b *testing.B) {
	m := New()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m.Get(AttrRealm) // nolint:errcheck
	}
}

func BenchmarkMessage_Get(b *testing.B) {
	m := New()
	m.Add(AttrUsername, []byte{1, 2, 3, 4, 5, 6, 7})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m.Get(AttrUsername) // nolint:errcheck
	}
}

func TestRawAttribute_AddTo(t *testing.T) {
	v := []byte{1, 2, 3, 4}
	m, err := Build(RawAttribute{
		Type:  AttrData,
		Value: v,
	})
	if err != nil {
		t.Fatal(err)
	}
	gotV, gotErr := m.Get(AttrData)
	if gotErr != nil {
		t.Fatal(gotErr)
	}
	if !bytes.Equal(gotV, v) {
		t.Error("value mismatch")
	}
}

func TestMessage_GetNoAllocs(t *testing.T) {
	m := New()
	NewSoftware("c").AddTo(m) // nolint:errcheck
	m.WriteHeader()

	t.Run("Default", func(t *testing.T) {
		allocs := testing.AllocsPerRun(10, func() {
			m.Get(AttrSoftware) // nolint:errcheck
		})
		if allocs > 0 {
			t.Error("allocated memory, but should not")
		}
	})
	t.Run("Not found", func(t *testing.T) {
		allocs := testing.AllocsPerRun(10, func() {
			m.Get(AttrOrigin) // nolint:errcheck
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

func TestAttrTypeRange(t *testing.T) {
	for _, a := range []AttrType{
		AttrPriority,
		AttrErrorCode,
		AttrUseCandidate,
		AttrEvenPort,
		AttrRequestedAddressFamily,
	} {
		a := a
		t.Run(a.String(), func(t *testing.T) {
			a := a
			if a.Optional() || !a.Required() {
				t.Error("should be required")
			}
		})
	}
	for _, a := range []AttrType{
		AttrSoftware,
		AttrICEControlled,
		AttrOrigin,
	} {
		a := a
		t.Run(a.String(), func(t *testing.T) {
			if a.Required() || !a.Optional() {
				t.Error("should be optional")
			}
		})
	}
}
