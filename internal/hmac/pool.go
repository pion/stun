package hmac

import (
	"crypto/sha1"
	"hash"
	"sync"
)

func resetBytes(b []byte) {
	for i := 0; i < len(b); i++ {
		b[i] = 0
	}
}

func (h *hmac) resetTo(key []byte) {
	h.outer.Reset()
	h.inner.Reset()
	resetBytes(h.ipad)
	resetBytes(h.opad)
	if len(key) > h.blocksize {
		// If key is too big, hash it.
		h.outer.Write(key)
		key = h.outer.Sum(nil)
	}
	copy(h.ipad, key)
	copy(h.opad, key)
	for i := range h.ipad {
		h.ipad[i] ^= 0x36
	}
	for i := range h.opad {
		h.opad[i] ^= 0x5c
	}
	h.inner.Write(h.ipad)
}

var hmacSHA1Pool = &sync.Pool{
	New: func() interface{} {
		h := New(sha1.New, make([]byte, sha1.BlockSize))
		return h
	},
}

// AcquireSHA1 returns new HMAC from pool.
func AcquireSHA1(key []byte) hash.Hash {
	h := hmacSHA1Pool.Get().(*hmac)
	h.resetTo(key)
	return h
}

// PutSHA1 puts h to pool.
func PutSHA1(h hash.Hash) {
	hm := h.(*hmac)
	hmacSHA1Pool.Put(hm)
}
