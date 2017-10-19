[![Build Status](https://travis-ci.org/ernado/stun.svg)](https://travis-ci.org/ernado/stun)
[![Build status](https://ci.appveyor.com/api/projects/status/92mfv3vxlc8t8jjp/branch/master?svg=true)](https://ci.appveyor.com/project/ernado/stun/branch/master)
[![GoDoc](https://godoc.org/github.com/go-rtc/stun?status.svg)](http://godoc.org/github.com/go-rtc/stun)
[![Coverage Status](https://coveralls.io/repos/github/ernado/stun/badge.svg?branch=master)](https://coveralls.io/github/ernado/stun?branch=master)
[![Go Report](https://goreportcard.com/badge/github.com/go-rtc/stun?camo=retarded)](http://goreportcard.com/report/ernado/stun)
[![RFC 5389](https://img.shields.io/badge/RFC-5389-blue.svg)](https://tools.ietf.org/html/rfc5389)

# stun
Package stun implements Session Traversal Utilities for
NAT (STUN) [RFC 5389](https://tools.ietf.org/html/rfc5389) with no external dependencies and focuses on speed.
See [example](https://godoc.org/github.com/go-rtc/stun#example-Message)
or [stun server](https://github.com/go-rtc/stund) for usage.

# example
You can get your current IP address from any STUN server by sending
binding request. See more idiomatic example at `cmd/stun-client`.
```go
package main

import (
	"fmt"
	"time"

	"github.com/go-rtc/stun"
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
Go 1.9.1 is currently supported and tested in CI. Should work on 1.8, 1.7, and tip.

# benchmarks

Intel(R) Core(TM) i7-6700K CPU @ 4.00GHz, go 1.8:

```
benchmark                                     iter       time/iter      throughput   bytes alloc        allocs
---------                                     ----       ---------      ----------   -----------        ------
BenchmarkMappedAddress_AddTo-8            50000000     25.90 ns/op                        0 B/op   0 allocs/op
BenchmarkAlternateServer_AddTo-8          50000000     25.90 ns/op                        0 B/op   0 allocs/op
BenchmarkMessage_GetNotFound-8           200000000      6.10 ns/op                        0 B/op   0 allocs/op
BenchmarkErrorCode_AddTo-8                30000000     45.10 ns/op                        0 B/op   0 allocs/op
BenchmarkErrorCodeAttribute_AddTo-8       50000000     36.90 ns/op                        0 B/op   0 allocs/op
BenchmarkErrorCodeAttribute_GetFrom-8    200000000      9.20 ns/op                        0 B/op   0 allocs/op
BenchmarkFingerprint_AddTo-8              30000000     52.50 ns/op    1294.63 MB/s        0 B/op   0 allocs/op
BenchmarkFingerprint_Check-8              30000000     45.50 ns/op    1670.92 MB/s        0 B/op   0 allocs/op
BenchmarkMessageIntegrity_AddTo-8          2000000    884.00 ns/op      22.62 MB/s      480 B/op   6 allocs/op
BenchmarkMessageIntegrity_Check-8          2000000    982.00 ns/op      24.43 MB/s      480 B/op   6 allocs/op
BenchmarkMessage_Write-8                 100000000     18.50 ns/op    1513.46 MB/s        0 B/op   0 allocs/op
BenchmarkMessageType_Value-8            2000000000      0.27 ns/op                        0 B/op   0 allocs/op
BenchmarkMessage_WriteTo-8               100000000     15.40 ns/op                        0 B/op   0 allocs/op
BenchmarkMessage_ReadFrom-8              100000000     22.70 ns/op     881.70 MB/s        0 B/op   0 allocs/op
BenchmarkMessage_ReadBytes-8             100000000     14.60 ns/op    1371.19 MB/s        0 B/op   0 allocs/op
BenchmarkIsMessage-8                    2000000000      1.03 ns/op   19389.66 MB/s        0 B/op   0 allocs/op
BenchmarkMessage_NewTransactionID-8        1000000   1448.00 ns/op                        0 B/op   0 allocs/op
BenchmarkMessageFull-8                    10000000    151.00 ns/op                        0 B/op   0 allocs/op
BenchmarkMessageFullHardcore-8            30000000     57.50 ns/op                        0 B/op   0 allocs/op
BenchmarkMessage_WriteHeader-8           200000000      6.42 ns/op                        0 B/op   0 allocs/op
BenchmarkUnknownAttributes/AddTo-8       100000000     23.00 ns/op                        0 B/op   0 allocs/op
BenchmarkUnknownAttributes/GetFrom-8     100000000     16.90 ns/op                        0 B/op   0 allocs/op
BenchmarkXOR-8                            50000000     23.60 ns/op   43421.24 MB/s        0 B/op   0 allocs/op
BenchmarkXORSafe-8                        10000000    169.00 ns/op    6048.93 MB/s        0 B/op   0 allocs/op
BenchmarkXORFast-8                       100000000     23.00 ns/op   44457.23 MB/s        0 B/op   0 allocs/op
BenchmarkXORMappedAddress_AddTo-8         50000000     38.30 ns/op                        0 B/op   0 allocs/op
BenchmarkXORMappedAddress_GetFrom-8       50000000     25.40 ns/op                        0 B/op   0 allocs/op
```

# development

stun package is low-level core gortc module, so security, efficiency (both memory and cpu), simplicity,
code quality, and low dependencies are paramount goals.
