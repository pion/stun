package stun

import (
	"fmt"
	"strconv"
)

func reverseByte(b byte) byte {
	var d byte
	for i := 0; i < 8; i++ {
		d <<= 1
		d |= b & 1
		b >>= 1
	}
	return d
}

func reverseUint16(v uint16) uint16 {
	var d uint16
	for i := 0; i < 16; i++ {
		d <<= 1
		d |= v & 1
		v >>= 1
	}
	return d
}

func bByte(v byte) string {
	return fmt.Sprintf("0b%08s", strconv.FormatUint(uint64(v), 2))
}

func bUint16(v uint16) string {
	return fmt.Sprintf("0b%016s", strconv.FormatUint(uint64(v), 2))
}
