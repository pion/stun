[![Build Status](https://travis-ci.com/gortc/stun.svg)](https://travis-ci.com/gortc/stun)
[![Build status](https://ci.appveyor.com/api/projects/status/fw3drn3k52mf5ghw/branch/master?svg=true)](https://ci.appveyor.com/project/ernado/stun-j08g0/branch/master)
[![GoDoc](https://godoc.org/github.com/gortc/stun?status.svg)](http://godoc.org/github.com/gortc/stun)
[![codecov](https://codecov.io/gh/gortc/stun/branch/master/graph/badge.svg)](https://codecov.io/gh/gortc/stun)
[![Go Report](https://goreportcard.com/badge/github.com/gortc/stun?camo=retarded)](http://goreportcard.com/report/gortc/stun)

# STUN
Package stun implements Session Traversal Utilities for NAT (STUN) [[RFC 5389](https://tools.ietf.org/html/rfc5389)]
with no external dependencies and zero allocations in hot paths.
Complies to [gortc principles](https://github.com/gortc/dev/blob/master/README.md#principles) as core package.

See [example](https://godoc.org/github.com/gortc/stun#example-Message) and [stun server](https://github.com/gortc/stund) for simple usage,
or [gortcd](https://github.com/gortc/gortcd) for advanced one.

# Example
You can get your current IP address from any STUN server by sending
binding request. See more idiomatic example at `cmd/stun-client`.
```go
package main

import (
	"fmt"
	"time"

	"github.com/gortc/stun"
)

func main() {
	// Creating a "connection" to STUN server.
	c, err := stun.Dial("udp", "stun.l.google.com:19302")
	if err != nil {
		panic(err)
	}
	deadline := time.Now().Add(time.Second * 5)
	// Bulding binding request with random transaction id.
	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)
	// Sending request to STUN server, waiting for response message.
	if err := c.Do(message, deadline, func(res stun.Event) {
		if res.Error != nil {
			panic(res.Error)
		}
		// Decoding XOR-MAPPED-ADDRESS attribute from message.
		var xorAddr stun.XORMappedAddress
		if err := xorAddr.GetFrom(res.Message); err != nil {
			panic(err)
		}
		fmt.Println("your IP is", xorAddr.IP)
	}); err != nil {
		panic(err)
	}
}
```

## RFCs

The package aims to implement the follwing RFCs. Note that the requirement status is based on the [WebRTC spec](https://tools.ietf.org/html/draft-ietf-rtcweb-overview), focusing on data channels for now.

rfc | status | requirement | description
----|--------|-------------|----
[![RFC5389](https://img.shields.io/badge/RFC-5389-blue.svg)](https://tools.ietf.org/html/rfc5389) | ![status](https://img.shields.io/badge/status-beta-green.svg) | [![status](https://img.shields.io/badge/requirement-MUST-green.svg)](https://tools.ietf.org/html/rfc2119) | Session Traversal Utilities for NAT
IPv6 | ![status](https://img.shields.io/badge/status-research-orange.svg) | [![status](https://img.shields.io/badge/requirement-MUST-green.svg)](https://tools.ietf.org/html/rfc2119) | IPv6 support
[(TLS-over-)TCP](https://tools.ietf.org/html/rfc5389#section-7.2.2) | ![status](https://img.shields.io/badge/status-research-orange.svg) | [![status](https://img.shields.io/badge/requirement-MUST-green.svg)](https://tools.ietf.org/html/rfc2119) | Sending over TCP or TLS-over-TCP
[ALTERNATE-SERVER](https://tools.ietf.org/html/rfc5389#section-11) | ![status](https://img.shields.io/badge/status-dev-blue.svg) | [![status](https://img.shields.io/badge/requirement-MUST-green.svg)](https://tools.ietf.org/html/rfc2119) | ALTERNATE-SERVER Mechanism


# Stability
Package is currently approaching beta stage, API should be fairly stable
(with exception of Agent and Client) and implementation is almost complete.
Bug reports are welcome.

Additional attributes are unlikely to be implemented in scope of stun package,
the only exception is constants for attribute or message types.

# RFC 3489 notes
RFC 5389 obsoletes RFC 3489, so implementation was ignored by purpose, however,
RFC 3489 can be easily implemented as separate package.

# Requirements
Go 1.9.2 is currently supported and tested in CI. Should work on 1.8, 1.7, and tip.

# Benchmarks

Intel(R) Core(TM) i7-8700K:

```
version: v1.8.3
goos: linux
goarch: amd64
pkg: github.com/gortc/stun
PASS
benchmark                                          iter       time/iter      throughput   bytes alloc        allocs
---------                                          ----       ---------      ----------   -----------        ------
BenchmarkMappedAddress_AddTo-12               300000000     22.90 ns/op                        0 B/op   0 allocs/op
BenchmarkAlternateServer_AddTo-12             300000000     23.00 ns/op                        0 B/op   0 allocs/op
BenchmarkAgent_GC-12                            5000000   1965.00 ns/op                        0 B/op   0 allocs/op
BenchmarkAgent_Process-12                     200000000     48.50 ns/op                        0 B/op   0 allocs/op
BenchmarkMessage_GetNotFound-12              2000000000      4.27 ns/op                        0 B/op   0 allocs/op
BenchmarkMessage_Get-12                      2000000000      5.00 ns/op                        0 B/op   0 allocs/op
BenchmarkClient_Do-12                          10000000    576.00 ns/op                        0 B/op   0 allocs/op
BenchmarkErrorCode_AddTo-12                   200000000     41.30 ns/op                        0 B/op   0 allocs/op
BenchmarkErrorCodeAttribute_AddTo-12          200000000     35.90 ns/op                        0 B/op   0 allocs/op
BenchmarkErrorCodeAttribute_GetFrom-12       1000000000      9.24 ns/op                        0 B/op   0 allocs/op
BenchmarkFingerprint_AddTo-12                 100000000     52.30 ns/op     840.91 MB/s        0 B/op   0 allocs/op
BenchmarkFingerprint_Check-12                 200000000     42.30 ns/op    1228.93 MB/s        0 B/op   0 allocs/op
BenchmarkBuildOverhead/Build-12                50000000    163.00 ns/op                        0 B/op   0 allocs/op
BenchmarkBuildOverhead/BuildNonPointer-12      20000000    321.00 ns/op                      100 B/op   4 allocs/op
BenchmarkBuildOverhead/Raw-12                 100000000    140.00 ns/op                        0 B/op   0 allocs/op
BenchmarkMessageIntegrity_AddTo-12             10000000    832.00 ns/op      24.01 MB/s        0 B/op   0 allocs/op
BenchmarkMessageIntegrity_Check-12             10000000    822.00 ns/op      38.90 MB/s        0 B/op   0 allocs/op
BenchmarkMessage_Write-12                     500000000     19.30 ns/op    1448.31 MB/s        0 B/op   0 allocs/op
BenchmarkMessageType_Value-12               10000000000      0.30 ns/op                        0 B/op   0 allocs/op
BenchmarkMessage_WriteTo-12                  1000000000      9.69 ns/op                        0 B/op   0 allocs/op
BenchmarkMessage_ReadFrom-12                  500000000     18.90 ns/op    1057.69 MB/s        0 B/op   0 allocs/op
BenchmarkMessage_ReadBytes-12                 500000000     12.20 ns/op    1639.03 MB/s        0 B/op   0 allocs/op
BenchmarkIsMessage-12                       10000000000      0.74 ns/op   26884.88 MB/s        0 B/op   0 allocs/op
BenchmarkMessage_NewTransactionID-12           10000000    615.00 ns/op                        0 B/op   0 allocs/op
BenchmarkMessageFull-12                        50000000    157.00 ns/op                        0 B/op   0 allocs/op
BenchmarkMessageFullHardcore-12               100000000     62.90 ns/op                        0 B/op   0 allocs/op
BenchmarkMessage_WriteHeader-12              1000000000      6.22 ns/op                        0 B/op   0 allocs/op
BenchmarkMessage_CloneTo-12                   300000000     27.80 ns/op    2445.51 MB/s        0 B/op   0 allocs/op
BenchmarkMessage_AddTo-12                    2000000000      4.38 ns/op                        0 B/op   0 allocs/op
BenchmarkDecode-12                            500000000     17.40 ns/op                        0 B/op   0 allocs/op
BenchmarkUsername_AddTo-12                    500000000     17.40 ns/op                        0 B/op   0 allocs/op
BenchmarkUsername_GetFrom-12                  500000000     13.50 ns/op                        0 B/op   0 allocs/op
BenchmarkNonce_AddTo-12                       300000000     23.70 ns/op                        0 B/op   0 allocs/op
BenchmarkNonce_AddTo_BadLength-12             200000000     32.70 ns/op                       32 B/op   1 allocs/op
BenchmarkNonce_GetFrom-12                     500000000     13.40 ns/op                        0 B/op   0 allocs/op
BenchmarkUnknownAttributes/AddTo-12           300000000     22.10 ns/op                        0 B/op   0 allocs/op
BenchmarkUnknownAttributes/GetFrom-12         500000000     15.40 ns/op                        0 B/op   0 allocs/op
BenchmarkXOR-12                               500000000     16.10 ns/op   63573.54 MB/s
BenchmarkXORSafe-12                           100000000    108.00 ns/op    9450.13 MB/s
BenchmarkXORFast-12                           500000000     15.60 ns/op   65480.14 MB/s
BenchmarkXORMappedAddress_AddTo-12            100000000     50.10 ns/op                        0 B/op   0 allocs/op
BenchmarkXORMappedAddress_GetFrom-12          300000000     26.80 ns/op                        0 B/op   0 allocs/op
ok  	github.com/gortc/stun	390.104s
```

# Development goals

stun package is low-level core gortc module, so security, efficiency (both memory and cpu), simplicity,
code quality, and low dependencies are paramount goals.
