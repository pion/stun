package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ernado/stun"
)

func main() {
	flag.Parse()
	addr := flag.Arg(0)
	if len(addr) == 0 {
		fmt.Fprintln(os.Stderr, "no uri specified")
		os.Exit(2)
	}
	c, err := stun.Dial(addr)
	if err != nil {
		log.Fatal(err)
	}
	req, err := stun.Build(stun.TransactionID, stun.BindingRequest)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		if err := c.Read(); err != nil {
			log.Fatal(err)
		}
	}()
	if err := c.Do(req, func(res *stun.Message) error {
		var (
			xorAddr stun.XORMappedAddress
		)
		if err := xorAddr.GetFrom(res); err != nil {
			return err
		}
		fmt.Println(xorAddr)
		return nil
	}); err != nil {
		log.Fatal(err)
	}
}
