#!/usr/bin/env bash

# SPDX-FileCopyrightText: 2026 The Pion community <https://pion.ly>
# SPDX-License-Identifier: MIT

echo "net: $INTERFACE $SUBNET"
tcpdump -U -v -i $INTERFACE \
    src net $SUBNET and dst net $SUBNET \
    -w /root/dump/dump.pcap
