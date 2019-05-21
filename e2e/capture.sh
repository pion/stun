#!/usr/bin/env bash
echo "net: $INTERFACE $SUBNET"
tcpdump -U -v -i $INTERFACE \
    src net $SUBNET and dst net $SUBNET \
    -w /root/dump/dump.pcap
