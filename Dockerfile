FROM golang:1.9.1

COPY . /go/src/github.com/gortc/stun

RUN go test github.com/gortc/stun

