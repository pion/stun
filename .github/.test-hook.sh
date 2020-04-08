#!/usr/bin/env bash

set -e

# test fuzz inputs
go test -tags gofuzz -run TestFuzz -v .

# test with "debug" tag
go test -tags debug ./...

# test concurrency
go test -race -cpu=1,2,4 -run TestClient_DoConcurrent
