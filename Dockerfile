FROM golang:1.8

COPY . /go/src/github.com/ernado/stun

RUN go test github.com/ernado/stun

