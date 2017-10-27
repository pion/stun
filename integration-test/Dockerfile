FROM golang:1.9.2

ADD . /go/src/github.com/gortc/stun

WORKDIR /go/src/github.com/gortc/stun/integration-test

RUN go install .

CMD ["integration-test"]

