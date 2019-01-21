package stun

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSTUN(t *testing.T) {
	msg, err := Build(ClassIndication, MethodBinding, GenerateTransactionID())
	if err != nil {
		panic(err)
	}

	// Basic message
	base := msg.Raw

	// Break the first byte
	t2 := make([]byte, len(base))
	copy(t2, base)
	t2[0] = 255

	// Break the magic cookie
	t3 := make([]byte, len(base))
	copy(t3, base)
	t3[messageHeaderStart+magicCookieStart] = 2

	// Break the message length
	t4 := make([]byte, messageHeaderStart+magicCookieStart+magicCookieLength)
	copy(t4, base)

	testCases := []struct {
		raw    []byte
		result bool
	}{
		{base, true},
		{t2, false},
		{t3, false},
		{t4, false},
	}

	for i, testCase := range testCases {
		assert.Equal(t, testCase.result, IsSTUN(testCase.raw), "testCase: %d %v", i, testCase)
	}
}
