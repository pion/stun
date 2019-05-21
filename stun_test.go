package stun

import (
	"errors"
	"testing"
)

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) {
	return 0, errors.New("failed to read")
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
	return 0, errors.New("failed to write")
}

func TestWriteHelper(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("should panic")
		}
	}()
	writeOrPanic(errorWriter{}, make([]byte, 1))
}
