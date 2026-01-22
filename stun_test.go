// SPDX-FileCopyrightText: 2026 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package stun

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type errorReader struct{}

var (
	errErrorReaderFailedToRead  = errors.New("failed to read")
	errErrorReaderFailedToWrite = errors.New("failed to write")
)

func (errorReader) Read([]byte) (int, error) {
	return 0, errErrorReaderFailedToRead
}

func TestReadFullHelper(t *testing.T) {
	defer func() {
		assert.NotNil(t, recover(), "should panic")
	}()
	readFullOrPanic(errorReader{}, make([]byte, 1))
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) {
	return 0, errErrorReaderFailedToWrite
}

func TestWriteHelper(t *testing.T) {
	defer func() {
		assert.NotNil(t, recover(), "should panic")
	}()
	writeOrPanic(errorWriter{}, make([]byte, 1))
}
