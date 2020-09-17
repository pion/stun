package stun

import (
	"errors"
	"testing"
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
		if r := recover(); r == nil {
			t.Error("should panic")
		}
	}()
	readFullOrPanic(errorReader{}, make([]byte, 1))
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) {
	return 0, errErrorReaderFailedToWrite
}

func TestWriteHelper(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("should panic")
		}
	}()
	writeOrPanic(errorWriter{}, make([]byte, 1))
}
