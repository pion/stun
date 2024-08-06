// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

// Package main implements a CLI tool which acts as a STUN client
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/pion/stun/v3"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintln(os.Stderr, os.Args[0], "stun:stun.l.google.com:19302")
	}
	flag.Parse()

	uriStr := flag.Arg(0)
	if uriStr == "" {
		uriStr = "stun:stun.l.google.com:19302"
	}

	uri, err := stun.ParseURI(uriStr)
	if err != nil {
		log.Fatalf("Invalid URI '%s': %s", uriStr, err)
	}

	// we only try the first address, so restrict ourselves to IPv4
	c, err := stun.DialURI(uri, &stun.DialConfig{})
	if err != nil {
		log.Fatalf("Failed to dial: %s", err)
	}
	if err = c.Do(stun.MustBuild(stun.TransactionID, stun.BindingRequest), func(res stun.Event) {
		if res.Error != nil {
			log.Fatalf("Failed STUN transaction: %s", res.Error)
		}

		var xorAddr stun.XORMappedAddress
		if getErr := xorAddr.GetFrom(res.Message); getErr != nil {
			log.Fatalf("Failed to get XOR-MAPPED-ADDRESS: %s", getErr)
		}

		log.Print(xorAddr)
	}); err != nil {
		log.Fatal("Do:", err)
	}
	if err := c.Close(); err != nil {
		log.Fatalf("Failed to close connection: %s", err)
	}
}
