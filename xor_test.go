package stun

import (
	"testing"
)

func TestXORSafe(t *testing.T) {
	dst := make([]byte, 8)
	a := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	b := []byte{8, 7, 7, 6, 6, 3, 4, 1}
	safeXORBytes(dst, a, b)
	safeXORBytes(dst, dst, a)
	for i, v := range dst {
		if b[i] != v {
			t.Error(b[i], "!=", v)
		}
	}
}

func TestXORFast(t *testing.T) {
	if !supportsUnaligned {
		t.Skip()
	}
	dst := make([]byte, 8)
	a := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	b := []byte{8, 7, 7, 6, 6, 3, 4, 1}
	xorBytes(dst, a, b)
	xorBytes(dst, dst, a)
	for i, v := range dst {
		if b[i] != v {
			t.Error(b[i], "!=", v)
		}
	}
}
