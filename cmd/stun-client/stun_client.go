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
		fmt.Fprintln(os.Stderr, os.Args[0], "stun.l.google.com:19302")
	}
	flag.Parse()
	addr := flag.Arg(0)
	if addr == "" {
		addr = "stun.l.google.com:19302"
	}
	c, err := stun.Dial("udp", addr)
	if err != nil {
		log.Fatal("dial:", err)
	}
	if err = c.Do(stun.MustBuild(stun.TransactionID(), stun.BindingRequest), func(res stun.Event) {
		if res.Error != nil {
			log.Fatalln(err)
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
