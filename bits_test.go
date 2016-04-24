package stun

import (
	"testing"
)

func TestReverseByte(t *testing.T) {
	var tests = []struct {
		in  byte
		out byte
	}{
		{0x17, 0xe8},
		{0x22, 0x44},
		{0x44, 0x22},
		{0x4a, 0x52},
		{0x02, 0x40},
		{0x2c, 0x34},
		{0xc0, 0x03},
	}
	for _, tt := range tests {
		b := reverseByte(tt.in)
		if b != tt.out {
			t.Errorf("reverseByte(%s) -> %s, want %s", bByte(tt.in), bByte(b), bByte(tt.out))
		}
		r := reverseByte(b)
		if r != tt.in {
			t.Errorf("reverseByte(%s) -> %s, want %s", bByte(b), bByte(r), bByte(tt.in))
		}
	}
}

func TestReverseUint16(t *testing.T) {
	var tests = []struct {
		in  uint16
		out uint16
	}{
		{0x0017, 0xe800},
		{0x0022, 0x4400},
		{0x0216, 0x6840},
		{0x004a, 0x5200},
		{0x0002, 0x4000},
		{0x002c, 0x3400},
		{0x0264, 0x2640},
	}
	for _, tt := range tests {
		b := reverseUint16(tt.in)
		if b != tt.out {
			t.Errorf("reverseUint16(%s) -> %s, want %s", bUint16(tt.in), bUint16(b), bUint16(tt.out))
		}
		r := reverseUint16(b)
		if r != tt.in {
			t.Errorf("reverseUint16(%s) -> %s, want %s", bUint16(b), bUint16(r), bUint16(tt.in))
		}
	}
}
