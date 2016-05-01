[![Build Status](https://travis-ci.org/cydev/stun.svg)](https://travis-ci.org/cydev/stun)
[![GoDoc](https://godoc.org/github.com/cydev/stun?status.svg)](http://godoc.org/github.com/cydev/stun)
[![Coverage Status](https://coveralls.io/repos/github/cydev/stun/badge.svg?branch=master)](https://coveralls.io/github/cydev/stun?branch=master)
[![Go Report](http://goreportcard.com/badge/cydev/stun)](http://goreportcard.com/report/cydev/stun)
[![Concourse](https://img.shields.io/badge/ci-concourse-blue.svg?logo=data%3Aimage%2Fpng%3Bbase64%2CiVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAMAAAAoLQ9TAAAABGdBTUEAALGPC%2FxhBQAAACBjSFJNAAB6JgAAgIQAAPoAAACA6AAAdTAAAOpgAAA6mAAAF3CculE8AAABtlBMVEUAAAAjHyAiHh8jHyAjHyAjHyAjHyAjHyAiHh8eGhsgHB0lISIjHyAjHyAjHyA0MTEhHR4jHyAjHyAjHyAjHyAjHyAjHyAjHyAjHyAjHyAjHyAjHyAjHyAnJCQjHyA%2BOjslISIjHyAjHyAjHyAhHR4mIiMkICEjHyAjHyAiHh8hHR4iHh8jHyAoJCVhXl4wLC0yLzAnIyQhHR55dnfg399DP0ApJSZYVVZOS0wzLzBGQ0Ti4uLu7u5xbm8mIiNvbW1cWVpKR0g1MTKQjo%2F%2F%2F%2F%2FDwsKbmZphXl8hHB1UUVKDgYJraWlEQUEqJifHxcbb29uNi4uFg4QyLy8fGxyHhIWFg4N3dHUgHB2opqdlYmNraGliX2BMSUrGxcXV1dXJyMihn59kYmKvrq6Vk5NJRUZLSEm%2Bvb7v7%2B%2F19fXv7u%2FW1dV4dnaioaGZl5cyLi87Nzg9Ojs1MjNRTk9HREQxLS6enJ22tbU3NDQvLC0rJygiHR4dGRpwbm77%2B%2FurqaokICE0MDE4NTZPTExTUFHi4eH4%2BPhua2w%2BOjtVUlJVUlNoZWZ8enrBv8A3MzR1c3OBfn9saWqfnZ5d%2FABMAAAAK3RSTlMAAAAHSqnl%2FOWoSQcZk%2B%2FvkhiysQaS7u1JqKfk4vv6%2B%2BOmSJGw7pHj%2B0gHERDNRQAAAAFiS0dERPm0mMEAAAAJcEhZcwAALiMAAC4jAXilP3YAAAAHdElNRQfgAx0DBia5UtiSAAABD0lEQVQY0xXP5yNCYRSA8fNyyShc3UpJyCjOSbzIbBoVrhUysrIie2dl7%2Fkfq4%2FPt%2BcHACxTyMpWKHJy8%2FIZpJIpVQWIhLaGwiKRMWDKYiR7YxNvbmnlapGBpEJHW3sHdXZ1O4k0Eghal9vj9fGe3r5%2BP9eVgD4QHBgcQnl4ZNQ5No4GKA1NTE6Fwr7pGcfsXIQboSw4v7C4tBxdIb66FuEmKF%2FfiG1uxbd37Hx3bx9NUHHgPTw6JtfJ6RmeJ9AIegzwCxtdXl3fyEmiShB0JN%2Fe3T%2FEH5%2Bew6Qzg6Sh5Mvr2%2FtHNPaZwCoJmKgm%2BvJ8%2F%2Fz6%2F3zVqXXGRI0WOWGY19SmcWm%2B2WCxWi2GuvoMgH9zUDliBIqlogAAACV0RVh0ZGF0ZTpjcmVhdGUAMjAxNi0wMy0yOVQwMzowNjozOCswMjowMAUkgHkAAAAldEVYdGRhdGU6bW9kaWZ5ADIwMTYtMDMtMjlUMDM6MDY6MzgrMDI6MDB0eTjFAAAAGXRFWHRTb2Z0d2FyZQBBZG9iZSBJbWFnZVJlYWR5ccllPAAAAFd6VFh0UmF3IHByb2ZpbGUgdHlwZSBpcHRjAAB4nOPyDAhxVigoyk%2FLzEnlUgADIwsuYwsTIxNLkxQDEyBEgDTDZAMjs1Qgy9jUyMTMxBzEB8uASKBKLgDqFxF08kI1lQAAAABJRU5ErkJggg%3D%3D)](https://ci.cydev.ru/pipelines/stun)

# stun
Package stun implements Session Traversal Utilities for
NAT (STUN) [RFC 5389](https://tools.ietf.org/html/rfc5389).

Currently in active development. Do not use this package at all. API will
definetly break. TURN and ICE implementations are planned too.

Needs go 1.6 or better.

# Continious integration

This project uses [concourse](https://concourse.ci/) continious integration.


All development is made in [dev](https://github.com/cydev/stun/tree/dev) branch.
It is automatically merged to master with [concourse job](https://ci.cydev.ru/pipelines/stun/jobs/integration).
Staging branch will be introduced after alpha version, and will be used instead of master.
