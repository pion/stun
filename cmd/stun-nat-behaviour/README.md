# NAT behaviour discovery using STUN ([RFC 5780](https://tools.ietf.org/html/rfc5780))

This is an example of how to use the pion/stun package for client-side NAT
behaviour discovery. It performs two types of tests: one to determine the
client's NAT mapping behaviour, and one to determine the NAT filtering
behaviour.


### Usage
```sh
$ go install github.com/pion/stun/cmd/stun-nat-behaviour@latest
$ $GOPATH/bin/stun-nat-behaviour [options] [--server IP:port]
```

If `$GOPATH` is unset it defaults to `~/go`

The default value `--server` is stun.voip.blackberry.com:3478

Use `-h` to see all options

### Output
For a successful run you will see output like the following.

```
connecting to STUN server: stun.voip.blackberry.com:3478
...
Mapping Test I: Regular binding request
Received XOR-MAPPED-ADDRESS: ...:...
...
Mapping Test II: Send binding request to the other address but primary port
...
=> NAT mapping behavior: endpoint independent

connecting to STUN server: stun.voip.blackberry.com:3478
...
Filtering Test I: Regular binding request
...
Filtering Test II: Request to change both IP and port
...
Filtering Test III: Request to change port only
...
=> NAT filtering behavior: address and port dependent

```

These tests are defined in [RFC 5780 section 4](https://tools.ietf.org/html/rfc5780#section-4) and the asserted behaviours of NAT are defined in [RFC 4787](https://tools.ietf.org/html/rfc4787).

#### `XOR-MAPPED-ADDRESS`
This is how the STUN server sees the request. The IP/Port tuple is the hole punched in your NAT, and how other client sees the request.

####  `NAT mapping behaviour` ([RFC 4787 section 4](https://tools.ietf.org/html/rfc4787#section-4))
For each request your NAT will create a temporary mapping (hole). These are the different rules for creating and maintaining it.

* **`endpoint independent`**
If you send two UDP packets from the same port to different places on the other side of the NAT, the NAT will use the same port to send those packets. This is the only RTC-friendly mapping behavior, and RFC 4787 requires it as a best practice (REQ-1) on all NATs. If both NATs do this, ICE will be able to connect directly, for all of the filtering behaviors described below. This is the most important setting to get right. This mapping type corresponds to the "cone NATs" in the classic STUN defined in [RFC 3489 section 5](https://tools.ietf.org/html/rfc3489#section-5).

* **`address dependent`** and **`address and port dependent`**
If you send two UDP packets from the same port to different places on the other side of the NAT, the NAT will use the same port to send those packets ff the destination address is the same; the destination port does not matter. Sending to two different hosts will use different ports. This will require the use of a TURN server if both sides are using it, unless neither side is doing port-specific filtering, and at least one is doing `endpoint independent filtering`. If you want webrtc or anything else that uses P2P UDP networking, ***do not configure your NAT like this***. This mapping type (loosely) corresponds to the "symmetric NATs" in the classic STUN defined in [RFC 3489 section 5](https://tools.ietf.org/html/rfc3489#section-5).

#### `NAT filtering behavior` ([RFC 4787 section 5](https://tools.ietf.org/html/rfc4787#section-5))
Each hole punch will also have rules around what external traffic is accepted (and routed back to the hole creator).

* **`endpoint independent`**
This is the most permissive of the three. Once you have sent a UDP packet to anywhere on the other side of the NAT, anything else on the other side of the NAT can send UDP packets back to you on that port. This filtering policy gives sysadmins cold sweats, but RFC 4787 recommends its use when real-time-communication (or other things that require "application transparency"; eg gaming) is a priority. Note that this will not do a very good job of compensating if your NAT's mapping behavior is misconfigured. It is more important to get the mapping behavior right.

* **`address dependent`**
This is a middle ground that sysadmins have an easier time justifying, but my impression is that it is harder to configure. Once you have sent a UDP packet to a host on the other side of the NAT, the NAT will allow return UDP traffic from that host, regardless of the port that host sends from, but will not allow inbound UDP traffic from other addresses. If your mapping behavior is configured appropriately, this should function as well as `endpoint independent filtering`.

* **`address and port dependent`**
This is the strictest of the three. Your NAT will only allow return traffic from exactly where you sent your UDP packet. Using this is ***not recommended***, even if you configure mapping behavior correctly, because it will work poorly when the other NAT is misconfigured (fairly common).
