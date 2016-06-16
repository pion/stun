#!/bin/bash

set -e -u -x

export GOPATH=$PWD
export PATH=$GOPATH/bin:$PATH
mkdir -p src/github.com/ernado
cp -R dev src/github.com/ernado/stun

pushd ${GOPATH}/src/github.com/ernado/stun
    go get -v -t .
    make lint
popd
