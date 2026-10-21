// SPDX-FileCopyrightText: 2026 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

// Package hmac provides pooled HMAC-SHA1 and HMAC-SHA256 helpers for STUN
// MESSAGE-INTEGRITY. The HMAC itself is the standard library crypto/hmac; this
// package only adds a sync.Pool so that repeated signing or verification with
// the same key reuses one HMAC instance and avoids a per-message allocation.
package hmac

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // STUN MESSAGE-INTEGRITY is HMAC-SHA1 (RFC 5389 §15.4).
	"crypto/sha256"
	"hash"
	"sync"
)

// keyed wraps a standard library HMAC together with the key it was built with.
// crypto/hmac cannot be re-keyed, so a pooled instance is reused only when the
// next caller presents the same key: Reset then restores the pre-computed,
// key-dependent hash state (allocation free). A different key builds a fresh
// HMAC, which is no worse than not pooling at all.
type keyed struct {
	hash.Hash
	newHash func() hash.Hash
	key     []byte
}

// reset readies the wrapped HMAC for key: Reset on a cache hit, rebuild on a miss.
func (k *keyed) reset(key []byte) {
	if k.Hash != nil && bytes.Equal(k.key, key) {
		k.Hash.Reset()

		return
	}
	k.Hash = hmac.New(k.newHash, key)
	k.key = append(k.key[:0], key...)
}

var hmacSHA1Pool = &sync.Pool{ //nolint:gochecknoglobals
	New: func() any { return &keyed{newHash: sha1.New} },
}

var hmacSHA256Pool = &sync.Pool{ //nolint:gochecknoglobals
	New: func() any { return &keyed{newHash: sha256.New} },
}

// AcquireSHA1 returns an HMAC-SHA1 hash keyed with key, taken from a pool.
// Return it with PutSHA1 once done.
func AcquireSHA1(key []byte) hash.Hash {
	k := hmacSHA1Pool.Get().(*keyed) //nolint:forcetypeassert
	k.reset(key)

	return k
}

// PutSHA1 returns h to the HMAC-SHA1 pool.
func PutSHA1(h hash.Hash) {
	if k, ok := h.(*keyed); ok {
		hmacSHA1Pool.Put(k)
	}
}

// AcquireSHA256 returns an HMAC-SHA256 hash keyed with key, taken from a pool.
// Return it with PutSHA256 once done.
func AcquireSHA256(key []byte) hash.Hash {
	k := hmacSHA256Pool.Get().(*keyed) //nolint:forcetypeassert
	k.reset(key)

	return k
}

// PutSHA256 returns h to the HMAC-SHA256 pool.
func PutSHA256(h hash.Hash) {
	if k, ok := h.(*keyed); ok {
		hmacSHA256Pool.Put(k)
	}
}

// Equal reports whether two MACs are equal without leaking timing information.
func Equal(mac1, mac2 []byte) bool {
	return hmac.Equal(mac1, mac2)
}
