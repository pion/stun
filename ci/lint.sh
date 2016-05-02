#!/bin/bash

set -e -u -x

export GOPATH=$PWD
export PATH=$GOPATH/bin:$PATH
mkdir -p src/github.com/cydev
cp -R dev src/github.com/cydev/stun

pushd ${GOPATH}/src/github.com/cydev/stun
    go get -v -t .
    make lint-fast
popd
