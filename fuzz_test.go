// +build gofuzz

package stun

import (
	"testing"
	"os"
	"path/filepath"
)

func TestMessageType_FuzzerCrash1(t *testing.T) {
	input := []byte("\x9c\xbe\x03")
	FuzzType(input)
}

func TestMessageCrash2(t *testing.T) {
	input := []byte("00\x00\x000000000000000000")
	FuzzMessage(input)
}

func TestFuzzingCoverage(t *testing.T) {
	p := filepath.Join("examples", "stun-msg", "corpus")
	f, err := os.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	list, err := f.Readdir(-1)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range list {
		df, err := os.Open(filepath.Join(p, d.Name()))
		if err != nil {
			t.Fatal(err)
		}
		buf := make([]byte, 5000)
		n, _ := df.Read(buf)
		df.Close()
		FuzzMessage(buf[:n])
	}
}