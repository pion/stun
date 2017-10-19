FROM golang:1.9.1

ADD . /go/src/github.com/go-rtc/stun

WORKDIR /go/src/github.com/go-rtc/stun/integration-test

RUN go install .

CMD ["integration-test"]

