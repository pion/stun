// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

// Package main implements a CLI tool which acts as a STUN client
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/pion/stun"
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
		log.Fatalf("invalid URI '%s': %s", uriStr, err)
	}

	// we only try the first address, so restrict ourselves to IPv4
	c, err := stun.DialURI(uri, &stun.DialConfig{})
	if err != nil {
		log.Fatal("dial:", err)
	}
	if err = c.Do(stun.MustBuild(stun.TransactionID, stun.BindingRequest), func(res stun.Event) {
		if res.Error != nil {
			log.Fatalln(res.Error)
		}
		var xorAddr stun.XORMappedAddress
		if getErr := xorAddr.GetFrom(res.Message); getErr != nil {
			log.Fatalln(getErr)
		}
		fmt.Println(xorAddr)
	}); err != nil {
		log.Fatal("do:", err)
	}
	if err := c.Close(); err != nil {
		log.Fatalln(err)
	}
}
