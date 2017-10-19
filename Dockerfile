FROM golang:1.9.1

COPY . /go/src/github.com/go-rtc/stun

RUN go test github.com/go-rtc/stun

