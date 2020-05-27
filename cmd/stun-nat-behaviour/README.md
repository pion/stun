# NAT behaviour discovery using STUN (RFC 5780

This is an example of how to use the pion/stun package for client-side NAT
behaviour discovery. It performs two types of tests: one to determine the
client's NAT filtering behaviour, and one to determine the NAT mapping
behaviour. The option exists to provide a STUN server as a command-line
argument.

Usage:
```sh
$ go get github.com/pion/stun/cmd/stun-nat-behaviour
$ stun-nat-behaviour --server [IP:port]
```
