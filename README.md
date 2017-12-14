[![Build Status](https://travis-ci.org/gortc/stun.svg)](https://travis-ci.org/gortc/stun)
[![Build status](https://ci.appveyor.com/api/projects/status/fw3drn3k52mf5ghw/branch/master?svg=true)](https://ci.appveyor.com/project/ernado/stun-j08g0/branch/master)
[![GoDoc](https://godoc.org/github.com/gortc/stun?status.svg)](http://godoc.org/github.com/gortc/stun)
[![Coverage Status](https://coveralls.io/repos/github/gortc/stun/badge.svg?branch=master)](https://coveralls.io/github/gortc/stun?branch=master)
[![Go Report](https://goreportcard.com/badge/github.com/gortc/stun?camo=retarded)](http://goreportcard.com/report/gortc/stun)
[![RFC 5389](https://img.shields.io/badge/RFC-5389-blue.svg)](https://tools.ietf.org/html/rfc5389)

# stun
Package stun implements Session Traversal Utilities for
NAT (STUN) [RFC 5389](https://tools.ietf.org/html/rfc5389) with no external dependencies and focuses on speed.
See [example](https://godoc.org/github.com/gortc/stun#example-Message)
or [stun server](https://github.com/gortc/stund) for usage.

# example
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
	if err := c.Do(message, deadline, func(res stun.AgentEvent) {
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

# stability
Package is currently approaching beta stage, API should be fairly stable
and implementation is almost complete. Bug reports are welcome.

Additional attributes are unlikely to be implemented in scope of stun package,
the only exception is constants for attribute or message types.

# RFC 3489 notes
RFC 5389 obsoletes RFC 3489, so implementation was ignored by purpose, however,
RFC 3489 can be easily implemented as separate package.

# requirements
Go 1.9.2 is currently supported and tested in CI. Should work on 1.8, 1.7, and tip.

# benchmarks

Intel(R) Core(TM) i7-8700K:

```
goos: linux
goarch: amd64
pkg: github.com/gortc/stun
PASS
benchmark                                         iter       time/iter      throughput   bytes alloc        allocs
---------                                         ----       ---------      ----------   -----------        ------
BenchmarkMappedAddress_AddTo-12              100000000     22.50 ns/op                        0 B/op   0 allocs/op
BenchmarkAlternateServer_AddTo-12            100000000     22.20 ns/op                        0 B/op   0 allocs/op
BenchmarkAgent_GC-12                           1000000   2038.00 ns/op                        0 B/op   0 allocs/op
BenchmarkAgent_Process-12                     30000000     47.60 ns/op                        0 B/op   0 allocs/op
BenchmarkMessage_GetNotFound-12              300000000      4.29 ns/op                        0 B/op   0 allocs/op
BenchmarkClient_Do-12                          2000000    534.00 ns/op                       16 B/op   1 allocs/op
BenchmarkErrorCode_AddTo-12                   30000000     42.00 ns/op                        0 B/op   0 allocs/op
BenchmarkErrorCodeAttribute_AddTo-12          50000000     30.90 ns/op                        0 B/op   0 allocs/op
BenchmarkErrorCodeAttribute_GetFrom-12       200000000      7.78 ns/op                        0 B/op   0 allocs/op
BenchmarkFingerprint_AddTo-12                 30000000     47.30 ns/op     931.09 MB/s        0 B/op   0 allocs/op
BenchmarkFingerprint_Check-12                 50000000     38.20 ns/op    1360.04 MB/s        0 B/op   0 allocs/op
BenchmarkBuildOverhead/Build-12               10000000    139.00 ns/op                        0 B/op   0 allocs/op
BenchmarkBuildOverhead/BuildNonPointer-12      5000000    249.00 ns/op                      100 B/op   4 allocs/op
BenchmarkBuildOverhead/Raw-12                 20000000    114.00 ns/op                        0 B/op   0 allocs/op
BenchmarkMessageIntegrity_AddTo-12             2000000   1022.00 ns/op      19.56 MB/s      480 B/op   6 allocs/op
BenchmarkMessageIntegrity_Check-12             1000000   1084.00 ns/op      29.50 MB/s      480 B/op   6 allocs/op
BenchmarkMessage_Write-12                    100000000     16.10 ns/op    1743.03 MB/s        0 B/op   0 allocs/op
BenchmarkMessageType_Value-12               2000000000      0.23 ns/op                        0 B/op   0 allocs/op
BenchmarkMessage_WriteTo-12                  200000000      8.11 ns/op                        0 B/op   0 allocs/op
BenchmarkMessage_ReadFrom-12                 100000000     18.30 ns/op    1095.75 MB/s        0 B/op   0 allocs/op
BenchmarkMessage_ReadBytes-12                100000000     10.70 ns/op    1870.83 MB/s        0 B/op   0 allocs/op
BenchmarkIsMessage-12                       2000000000      0.68 ns/op   29576.35 MB/s        0 B/op   0 allocs/op
BenchmarkMessage_NewTransactionID-12           3000000    393.00 ns/op                        0 B/op   0 allocs/op
BenchmarkMessageFull-12                       10000000    138.00 ns/op                        0 B/op   0 allocs/op
BenchmarkMessageFullHardcore-12               30000000     53.90 ns/op                        0 B/op   0 allocs/op
BenchmarkMessage_WriteHeader-12              300000000      5.45 ns/op                        0 B/op   0 allocs/op
BenchmarkUsername_AddTo-12                   100000000     15.00 ns/op                        0 B/op   0 allocs/op
BenchmarkUsername_GetFrom-12                 100000000     11.90 ns/op                        0 B/op   0 allocs/op
BenchmarkNonce_AddTo-12                      100000000     20.70 ns/op                        0 B/op   0 allocs/op
BenchmarkNonce_AddTo_BadLength-12            100000000     27.40 ns/op                       32 B/op   1 allocs/op
BenchmarkNonce_GetFrom-12                    200000000     11.90 ns/op                        0 B/op   0 allocs/op
BenchmarkUnknownAttributes/AddTo-12          100000000     18.90 ns/op                        0 B/op   0 allocs/op
BenchmarkUnknownAttributes/GetFrom-12        100000000     13.70 ns/op                        0 B/op   0 allocs/op
BenchmarkXOR-12                              100000000     14.60 ns/op   70066.01 MB/s        0 B/op   0 allocs/op
BenchmarkXORSafe-12                           20000000     98.80 ns/op   10363.12 MB/s        0 B/op   0 allocs/op
BenchmarkXORFast-12                          100000000     13.80 ns/op   74327.93 MB/s        0 B/op   0 allocs/op
BenchmarkXORMappedAddress_AddTo-12            50000000     35.20 ns/op                        0 B/op   0 allocs/op
BenchmarkXORMappedAddress_GetFrom-12         100000000     22.30 ns/op                        0 B/op   0 allocs/op
```

# development

stun package is low-level core gortc module, so security, efficiency (both memory and cpu), simplicity,
code quality, and low dependencies are paramount goals.
