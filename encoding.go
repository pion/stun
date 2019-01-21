package stun

import "encoding/binary"

// STUN expects all messages to be BigEndian encoded

var (
	enc = binary.BigEndian
)
