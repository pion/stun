FROM golang:1.12

COPY . /go/src/github.com/gortc/stun

RUN go test github.com/gortc/stun
