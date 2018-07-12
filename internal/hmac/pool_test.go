package hmac

import (
	"fmt"
	"testing"
)

func BenchmarkHMACSHA1_512(b *testing.B) {
	key := make([]byte, 32)
	buf := make([]byte, 512)
	b.ReportAllocs()
	h := AcquireSHA1(key)
	b.SetBytes(int64(len(buf)))
	for i := 0; i < b.N; i++ {
		h.Write(buf)
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
		h.Write(buf)
		h.Reset()
		mac := h.Sum(tBuf)
		buf[0] = mac[0]
		PutSHA1(h)
	}
}

func TestHMACPool(t *testing.T) {
	for i, tt := range hmacTests {
		h := New(tt.hash, tt.key)
		h.(*hmac).resetTo(tt.key)
		if s := h.Size(); s != tt.size {
			t.Errorf("Size: got %v, want %v", s, tt.size)
		}
		if b := h.BlockSize(); b != tt.blocksize {
			t.Errorf("BlockSize: got %v, want %v", b, tt.blocksize)
		}
		for j := 0; j < 2; j++ {
			n, err := h.Write(tt.in)
			if n != len(tt.in) || err != nil {
				t.Errorf("test %d.%d: Write(%d) = %d, %v", i, j, len(tt.in), n, err)
				continue
			}

			// Repetitive Sum() calls should return the same value
			for k := 0; k < 2; k++ {
				sum := fmt.Sprintf("%x", h.Sum(nil))
				if sum != tt.out {
					t.Errorf("test %d.%d.%d: have %s want %s\n", i, j, k, sum, tt.out)
				}
			}

			// Second iteration: make sure reset works.
			h.Reset()
		}
	}
}
