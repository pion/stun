// Package stun implements Session Traversal Utilities for NAT (STUN) RFC 5389.
package stun

import "encoding/binary"

// bin is shorthand to binary.BigEndian.
var bin = binary.BigEndian

// DefaultPort is IANA assigned Port for "stun" protocol.
const DefaultPort = 3478
