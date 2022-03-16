FROM golang:1.18

COPY . /go/src/github.com/pion/stun

RUN go test github.com/pion/stun
