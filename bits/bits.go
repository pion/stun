package stun

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
