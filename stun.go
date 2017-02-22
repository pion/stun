// Package stun implements Session Traversal Utilities for NAT (STUN) RFC 5389.
//
// The stun package is intended to use by package that implements extension
// to STUN (e.g. TURN) or client/server applications.
package stun

import "encoding/binary"

// bin is shorthand to binary.BigEndian.
var bin = binary.BigEndian

// IANA assigned ports for "stun" protocol/
const (
	DefaultPort    = 3478
	DefaultTLSPort = 5349
)

type transactionIDSetter bool

func (transactionIDSetter) AddTo(m *Message) error {
	return m.NewTransactionID()
}

// TransactionID is Setter for m.TransactionID.
var TransactionID Setter = transactionIDSetter(true)
