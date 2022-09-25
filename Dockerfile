FROM golang:1.19

COPY . /go/src/github.com/pion/stun

RUN go test github.com/pion/stun
