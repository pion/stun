[![Build Status](https://travis-ci.com/gortc/stun.svg)](https://travis-ci.com/gortc/stun)
[![Build status](https://ci.appveyor.com/api/projects/status/fw3drn3k52mf5ghw/branch/master?svg=true)](https://ci.appveyor.com/project/ernado/stun-j08g0/branch/master)
[![GoDoc](https://godoc.org/github.com/gortc/stun?status.svg)](http://godoc.org/github.com/gortc/stun)
[![Coverage Status](https://coveralls.io/repos/github/gortc/stun/badge.svg?branch=master&v=1)](https://coveralls.io/github/gortc/stun?branch=master)
[![Go Report](https://goreportcard.com/badge/github.com/gortc/stun?camo=retarded)](http://goreportcard.com/report/gortc/stun)

# STUN
Package stun implements Session Traversal Utilities for NAT (STUN) [[RFC 5389](https://tools.ietf.org/html/rfc5389)] with no external dependencies and focuses on speed.
See [example](https://godoc.org/github.com/gortc/stun#example-Message) or [stun server](https://github.com/gortc/stund) for usage.

## RFCs

The package aims to implement the follwing RFCs. Note that the requirement status is based on the [WebRTC spec](https://tools.ietf.org/html/draft-ietf-rtcweb-overview), focusing on data channels for now.

rfc | status | requirement | description
----|--------|-------------|----
[![RFC5389](https://img.shields.io/badge/RFC-5389-blue.svg)](https://tools.ietf.org/html/rfc5389) | ![status](https://img.shields.io/badge/status-beta-green.svg) | [![status](https://img.shields.io/badge/requirement-MUST-green.svg)](https://tools.ietf.org/html/rfc2119) | Session Traversal Utilities for NAT
IPv6 | ![status](https://img.shields.io/badge/status-research-orange.svg) | [![status](https://img.shields.io/badge/requirement-MUST-green.svg)](https://tools.ietf.org/html/rfc2119) | IPv6 support
[(TLS-over-)TCP](https://tools.ietf.org/html/rfc5389#section-7.2.2) | ![status](https://img.shields.io/badge/status-research-orange.svg) | [![status](https://img.shields.io/badge/requirement-MUST-green.svg)](https://tools.ietf.org/html/rfc2119) | Sending over TCP or TLS-over-TCP
[ALTERNATE-SERVER](https://tools.ietf.org/html/rfc5389#section-11) | ![status](https://img.shields.io/badge/status-dev-blue.svg) | [![status](https://img.shields.io/badge/requirement-MUST-green.svg)](https://tools.ietf.org/html/rfc2119) | ALTERNATE-SERVER Mechanism

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

# Stability
Package is currently approaching beta stage, API should be fairly stable
and implementation is almost complete. Bug reports are welcome.

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
goos: linux
goarch: amd64
pkg: github.com/gortc/stun
PASS
benchmark                                         iter       time/iter      throughput   bytes alloc        allocs
---------                                         ----       ---------      ----------   -----------        ------
BenchmarkMappedAddress_AddTo-12              100000000     22.80 ns/op                        0 B/op   0 allocs/op
BenchmarkAlternateServer_AddTo-12            100000000     22.80 ns/op                        0 B/op   0 allocs/op
BenchmarkAgent_GC-12                           1000000   1943.00 ns/op                        0 B/op   0 allocs/op
BenchmarkAgent_Process-12                     30000000     50.60 ns/op                        0 B/op   0 allocs/op
BenchmarkMessage_GetNotFound-12              300000000      4.21 ns/op                        0 B/op   0 allocs/op
BenchmarkClient_Do-12                          3000000    710.00 ns/op                        0 B/op   0 allocs/op
BenchmarkErrorCode_AddTo-12                   30000000     41.30 ns/op                        0 B/op   0 allocs/op
BenchmarkErrorCodeAttribute_AddTo-12          50000000     31.10 ns/op                        0 B/op   0 allocs/op
BenchmarkErrorCodeAttribute_GetFrom-12       200000000      7.77 ns/op                        0 B/op   0 allocs/op
BenchmarkFingerprint_AddTo-12                 30000000     43.60 ns/op    1008.12 MB/s        0 B/op   0 allocs/op
BenchmarkFingerprint_Check-12                 50000000     36.60 ns/op    1422.45 MB/s        0 B/op   0 allocs/op
BenchmarkBuildOverhead/Build-12               10000000    134.00 ns/op                        0 B/op   0 allocs/op
BenchmarkBuildOverhead/BuildNonPointer-12      5000000    301.00 ns/op                      100 B/op   4 allocs/op
BenchmarkBuildOverhead/Raw-12                 20000000    112.00 ns/op                        0 B/op   0 allocs/op
BenchmarkMessageIntegrity_AddTo-12             2000000    695.00 ns/op      28.77 MB/s        0 B/op   0 allocs/op
BenchmarkMessageIntegrity_Check-12             2000000    692.00 ns/op      46.20 MB/s        0 B/op   0 allocs/op
BenchmarkMessage_Write-12                    100000000     16.60 ns/op    1691.61 MB/s        0 B/op   0 allocs/op
BenchmarkMessageType_Value-12               2000000000      0.23 ns/op                        0 B/op   0 allocs/op
BenchmarkMessage_WriteTo-12                  200000000      8.40 ns/op                        0 B/op   0 allocs/op
BenchmarkMessage_ReadFrom-12                 100000000     16.00 ns/op    1248.26 MB/s        0 B/op   0 allocs/op
BenchmarkMessage_ReadBytes-12                100000000     10.50 ns/op    1906.19 MB/s        0 B/op   0 allocs/op
BenchmarkIsMessage-12                       2000000000      0.66 ns/op   30498.98 MB/s        0 B/op   0 allocs/op
BenchmarkMessage_NewTransactionID-12           3000000    529.00 ns/op                        0 B/op   0 allocs/op
BenchmarkMessageFull-12                       10000000    134.00 ns/op                        0 B/op   0 allocs/op
BenchmarkMessageFullHardcore-12               30000000     54.50 ns/op                        0 B/op   0 allocs/op
BenchmarkMessage_WriteHeader-12              300000000      5.48 ns/op                        0 B/op   0 allocs/op
BenchmarkMessage_CloneTo-12                   50000000     24.00 ns/op    2835.42 MB/s        0 B/op   0 allocs/op
BenchmarkMessage_AddTo-12                    300000000      4.19 ns/op                        0 B/op   0 allocs/op
BenchmarkDecode-12                           100000000     14.80 ns/op                        0 B/op   0 allocs/op
BenchmarkUsername_AddTo-12                   100000000     14.90 ns/op                        0 B/op   0 allocs/op
BenchmarkUsername_GetFrom-12                 100000000     11.60 ns/op                        0 B/op   0 allocs/op
BenchmarkNonce_AddTo-12                      100000000     20.30 ns/op                        0 B/op   0 allocs/op
BenchmarkNonce_AddTo_BadLength-12            100000000     59.30 ns/op                       32 B/op   1 allocs/op
BenchmarkNonce_GetFrom-12                    100000000     11.70 ns/op                        0 B/op   0 allocs/op
BenchmarkUnknownAttributes/AddTo-12          100000000     18.70 ns/op                        0 B/op   0 allocs/op
BenchmarkUnknownAttributes/GetFrom-12        100000000     13.80 ns/op                        0 B/op   0 allocs/op
BenchmarkXOR-12                              100000000     13.20 ns/op   77486.83 MB/s
BenchmarkXORSafe-12                           20000000     90.50 ns/op   11313.32 MB/s
BenchmarkXORFast-12                          100000000     12.80 ns/op   79780.04 MB/s
BenchmarkXORMappedAddress_AddTo-12            50000000     33.60 ns/op                        0 B/op   0 allocs/op
BenchmarkXORMappedAddress_GetFrom-12         100000000     23.10 ns/op                        0 B/op   0 allocs/op
ok  	github.com/gortc/stun	75.241s
```

# Development goals

stun package is low-level core gortc module, so security, efficiency (both memory and cpu), simplicity,
code quality, and low dependencies are paramount goals.
