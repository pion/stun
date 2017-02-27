FROM golang:1.8

ADD . /go/src/github.com/ernado/stun

WORKDIR /go/src/github.com/ernado/stun/integration-test

RUN go install .

CMD ["integration-test"]

