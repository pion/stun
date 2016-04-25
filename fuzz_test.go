// +build gofuzz

package stun

import "testing"

func TestMessageType_FuzzerCrash1(t *testing.T) {
	input := []byte("\x9c\xbe\x03")
	FuzzType(input)
}
