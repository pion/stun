// +build gofuzz

package stun

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMessageType_FuzzerCrash1(t *testing.T) {
	input := []byte("\x9c\xbe\x03")
	FuzzType(input)
}

func TestMessageCrash2(t *testing.T) {
	input := []byte("00\x00\x000000000000000000")
	FuzzMessage(input)
}

func corpus(t *testing.T, function, typ string) [][]byte {
	var data [][]byte
	p := filepath.Join("fuzz", function, typ)
	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("does not exist")
		}
		t.Fatal(err)
	}
	list, err := f.Readdir(-1)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range list {
		if strings.Contains(d.Name(), ".") {
			// Skipping non-raw files.
			continue
		}
		df, err := os.Open(filepath.Join(p, d.Name()))
		if err != nil {
			t.Fatal(err)
		}
		buf := make([]byte, 5000)
		n, _ := df.Read(buf)
		data = append(data, buf[:n])
		df.Close()
	}
	return data
}

func TestFuzzMessage_Coverage(t *testing.T) {
	for _, buf := range corpus(t, "stun-msg", "corpus") {
		FuzzMessage(buf)
	}
}

func TestFuzzMessage_Crashers(t *testing.T) {
	for _, buf := range corpus(t, "stun-msg", "crashers") {
		FuzzMessage(buf)
	}
}

func TestFuzzType_Coverage(t *testing.T) {
	for _, buf := range corpus(t, "stun-typ", "corpus") {
		FuzzType(buf)
	}
}

func TestFuzzType_Crashers(t *testing.T) {
	for _, buf := range corpus(t, "stun-typ", "crashers") {
		FuzzType(buf)
	}
}

func TestAttrPick(t *testing.T) {
	attributes := attrs{
		{new(XORMappedAddress), AttrXORMappedAddress},
	}
	for i := byte(0); i < 255; i++ {
		attributes.pick(i)
	}
}

func TestFuzzSetters_Crashers(t *testing.T) {
	for _, buf := range corpus(t, "stun-setters", "crashers") {
		FuzzSetters(buf)
	}
}

func TestFuzzSetters_Coverage(t *testing.T) {
	for _, buf := range corpus(t, "stun-setters", "corpus") {
		FuzzSetters(buf)
	}
}
