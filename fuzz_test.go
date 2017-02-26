// +build gofuzz

package stun

import "testing"

func TestMessageType_FuzzerCrash1(t *testing.T) {
	input := []byte("\x9c\xbe\x03")
	FuzzType(input)
}

func TestMessageCrash2(t *testing.T) {
	input := []byte("00\x00\x000000000000000000")
	FuzzMessage(input)
}
