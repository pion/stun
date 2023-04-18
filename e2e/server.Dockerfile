# SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
# SPDX-License-Identifier: MIT

FROM ubuntu:latest

RUN apt-get update
RUN apt-get install -y coturn

USER turnserver

ENTRYPOINT ["/usr/bin/turnserver"]