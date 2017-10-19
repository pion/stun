package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-rtc/stun"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintln(os.Stderr, os.Args[0], "stun.l.google.com:19302")
	}
	flag.Parse()
	addr := flag.Arg(0)
	if len(addr) == 0 {
		fmt.Fprintln(os.Stderr, "no address specified")
		flag.Usage()
		os.Exit(2)
	}
	c, err := stun.Dial("udp", addr)
	if err != nil {
		log.Fatal("dial:", err)
	}
	deadline := time.Now().Add(time.Second * 5)
	if err := c.Do(stun.MustBuild(stun.TransactionID, stun.BindingRequest), deadline, func(res stun.AgentEvent) {
		if res.Error != nil {
			log.Fatalln(err)
		}
		var xorAddr stun.XORMappedAddress
		if err := xorAddr.GetFrom(res.Message); err != nil {
			log.Fatalln(err)
		}
		fmt.Println(xorAddr)
	}); err != nil {
		log.Fatal("do:", err)
	}
	if err := c.Close(); err != nil {
		log.Fatalln(err)
	}
}
