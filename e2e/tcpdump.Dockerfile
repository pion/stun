# SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
# SPDX-License-Identifier: MIT

FROM ubuntu
RUN apt-get update && apt-get install -y tcpdump
RUN apt-get install net-tools -y

ADD capture.sh /root/capture.sh
ENTRYPOINT ["/bin/bash", "/root/capture.sh"]
