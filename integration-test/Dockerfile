ARG CI_GO_VERSION
FROM golang:${CI_GO_VERSION}

ADD . /go/src/github.com/gortc/stun

WORKDIR /go/src/github.com/gortc/stun/integration-test

RUN go install .

CMD ["integration-test"]

