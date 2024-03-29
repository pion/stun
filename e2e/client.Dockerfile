# SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
# SPDX-License-Identifier: MIT

ARG CI_GO_VERSION
FROM golang:${CI_GO_VERSION}

ADD . /go/src/github.com/pion/stun

WORKDIR /go/src/github.com/pion/stun/e2e

RUN go install .

CMD ["e2e"]

