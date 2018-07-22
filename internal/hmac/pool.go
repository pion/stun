package hmac

import (
	"crypto/sha1"
	"crypto/sha256"
	"hash"
	"sync"
)

// setZeroes sets all bytes from b to zeroes.
//
// See https://github.com/golang/go/issues/5373
func setZeroes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func (h *hmac) resetTo(key []byte) {
	h.outer.Reset()
	h.inner.Reset()
	setZeroes(h.ipad)
	setZeroes(h.opad)
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

var hmacSHA256Pool = &sync.Pool{
	New: func() interface{} {
		h := New(sha256.New, make([]byte, sha256.BlockSize))
		return h
	},
}

// AcquireSHA256 returns new HMAC from SHA256 pool.
func AcquireSHA256(key []byte) hash.Hash {
	h := hmacSHA256Pool.Get().(*hmac)
	h.resetTo(key)
	return h
}

// PutSHA256 puts h to SHA256 pool.
func PutSHA256(h hash.Hash) {
	hm := h.(*hmac)
	hmacSHA256Pool.Put(hm)
}
