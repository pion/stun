[![Build Status](https://travis-ci.org/ernado/stun.svg)](https://travis-ci.org/ernado/stun)
[![Build status](https://ci.appveyor.com/api/projects/status/92mfv3vxlc8t8jjp/branch/master?svg=true)](https://ci.appveyor.com/project/ernado/stun/branch/master)
[![GoDoc](https://godoc.org/github.com/ernado/stun?status.svg)](http://godoc.org/github.com/ernado/stun)
[![Coverage Status](https://coveralls.io/repos/github/ernado/stun/badge.svg?branch=master)](https://coveralls.io/github/ernado/stun?branch=master)
[![Go Report](https://goreportcard.com/badge/github.com/ernado/stun?camo=retarded)](http://goreportcard.com/report/ernado/stun)
[![RFC 5389](https://img.shields.io/badge/RFC-5389-blue.svg)](https://tools.ietf.org/html/rfc5389)

# stun
Package stun implements Session Traversal Utilities for
NAT (STUN) [RFC 5389](https://tools.ietf.org/html/rfc5389) with focus
on speed and zero allocations in hot paths, no external dependencies.

Currently in active development, API is subject to change.

Needs go 1.7 or better.