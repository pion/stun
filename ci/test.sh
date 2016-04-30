#!/bin/bash

set -e -u -x

export GOPATH=$PWD/cydev-stun

cd cydev-stun
go get -t
go test -race
