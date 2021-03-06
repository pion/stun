// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stun

import (
	"runtime"
	"unsafe"
)

const wordSize = int(unsafe.Sizeof(uintptr(0))) // nolint: gosec

var supportsUnaligned = runtime.GOARCH == "386" || runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64" // nolint:gochecknoglobals

// fastXORBytes xors in bulk. It only works on architectures that
// support unaligned read/writes.
func fastXORBytes(dst, a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}

	w := n / wordSize
	if w > 0 {
		dw := *(*[]uintptr)(unsafe.Pointer(&dst)) // nolint: gosec
		aw := *(*[]uintptr)(unsafe.Pointer(&a))   // nolint: gosec
		bw := *(*[]uintptr)(unsafe.Pointer(&b))   // nolint: gosec
		for i := 0; i < w; i++ {
			dw[i] = aw[i] ^ bw[i]
		}
	}

	for i := n - n%wordSize; i < n; i++ {
		dst[i] = a[i] ^ b[i]
	}

	return n
}

func safeXORBytes(dst, a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		dst[i] = a[i] ^ b[i]
	}
	return n
}

// xorBytes xors the bytes in a and b. The destination is assumed to have enough
// space. Returns the number of bytes xor'd.
func xorBytes(dst, a, b []byte) int { // nolint:unparam
	if supportsUnaligned {
		return fastXORBytes(dst, a, b)
	}
	return safeXORBytes(dst, a, b)
}
