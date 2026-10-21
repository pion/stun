// SPDX-FileCopyrightText: 2026 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package hmac

import (
	"crypto/sha1" //nolint:gosec
	"encoding/hex"
	"hash"
	"testing"

	"github.com/stretchr/testify/assert"
)

type kat struct{ key, msg, wantSHA1, wantSHA256 string }

// katVectors are RFC 2202 (HMAC-SHA1) and RFC 4231 (HMAC-SHA256) vectors 1 and 2.
func katVectors() []kat {
	return []kat{
		{
			key:        "\x0b\x0b\x0b\x0b\x0b\x0b\x0b\x0b\x0b\x0b\x0b\x0b\x0b\x0b\x0b\x0b\x0b\x0b\x0b\x0b",
			msg:        "Hi There",
			wantSHA1:   "b617318655057264e28bc0b6fb378c8ef146be00",
			wantSHA256: "b0344c61d8db38535ca8afceaf0bf12b881dc200c9833da726e9376c2e32cff7",
		},
		{
			key:        "Jefe",
			msg:        "what do ya want for nothing?",
			wantSHA1:   "effcdf6ae5eb2fa2d27416d5f184df9c259a7c79",
			wantSHA256: "5bdcc146bf60754e6a042426089575c75a003f089d2739839dec58b964ec3843",
		},
	}
}

func macHex(h hash.Hash, msg string) string {
	h.Write([]byte(msg)) //nolint:errcheck

	return hex.EncodeToString(h.Sum(nil))
}

// TestPoolKnownAnswers verifies the pooled HMAC against the RFC 2202 / RFC 4231
// vectors for both digests, through the Acquire/Put pool API.
func TestPoolKnownAnswers(t *testing.T) {
	for i, tt := range katVectors() {
		h1 := AcquireSHA1([]byte(tt.key))
		assert.Equal(t, tt.wantSHA1, macHex(h1, tt.msg), "vector %d SHA1", i)
		PutSHA1(h1)

		h256 := AcquireSHA256([]byte(tt.key))
		assert.Equal(t, tt.wantSHA256, macHex(h256, tt.msg), "vector %d SHA256", i)
		PutSHA256(h256)
	}
}

// TestKeyedRekey drives the key-change path directly: the same instance
// re-keyed to a different key must produce the new key's MAC, and returning to
// the first key must reproduce it - guarding against stale-key reuse.
func TestKeyedRekey(t *testing.T) {
	v := katVectors()
	a, b := v[0], v[1]
	k := &keyed{newHash: sha1.New}
	for _, step := range []struct{ key, msg, want string }{
		{a.key, a.msg, a.wantSHA1}, // build (hash was nil)
		{b.key, b.msg, b.wantSHA1}, // rebuild (key changed)
		{a.key, a.msg, a.wantSHA1}, // rebuild back
		{a.key, a.msg, a.wantSHA1}, // reuse (same key -> Reset)
	} {
		k.reset([]byte(step.key))
		assert.Equal(t, step.want, macHex(k, step.msg), "key %q", step.key)
	}
}

func BenchmarkHMACSHA1_512(b *testing.B) {
	key := make([]byte, 32)
	buf := make([]byte, 512)
	b.ReportAllocs()
	h := AcquireSHA1(key)
	b.SetBytes(int64(len(buf)))
	for i := 0; i < b.N; i++ {
		h.Write(buf) //nolint:errcheck,gosec
		h.Reset()
		mac := h.Sum(nil)
		buf[0] = mac[0]
	}
}

func BenchmarkHMACSHA1_512_Pool(b *testing.B) {
	key := make([]byte, 32)
	buf := make([]byte, 512)
	tBuf := make([]byte, 0, 512)
	b.ReportAllocs()
	b.SetBytes(int64(len(buf)))
	for i := 0; i < b.N; i++ {
		h := AcquireSHA1(key)
		h.Write(buf) //nolint:errcheck,gosec
		h.Reset()
		mac := h.Sum(tBuf)
		buf[0] = mac[0]
		PutSHA1(h)
	}
}
