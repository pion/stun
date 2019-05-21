# Multiplex

An example of doing UDP connection multiplexing
that splits incoming UDP packets to two streams, "STUN Data" and "Application Data".

Usage:
```sh
$ go get github.com/pion/stun/cmd/stun-multiplex
```

On "server":
```sh
$ stun-multiplex
local addr: 0.0.0.0:34690 stun server addr: 64.233.161.127:19302
public addr: 123.131.100.200:34690
Acting as server. Use following command to connect:
stun-multiplex 123.131.100.200:34690
```

On "client":
```sh
$ stun-multiplex 123.131.100.200:34690
local addr: 0.0.0.0:37551 stun server addr: 66.102.1.127:19302
public addr: 159.69.13.15:37551
Acting as client. Connecting to 123.131.100.200:34690
Writing 123.131.100.200:34690
demultiplex: [123.131.100.200:34690]: Hello peer
Got response from 123.131.100.200:34690: Hello peer
```

On "server" you will see `demultiplex: [159.69.13.15:37551]: Hello peer` message.