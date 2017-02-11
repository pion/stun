package stun

import "testing"

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
