package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sync"

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
		log.Fatal("dial:", err)
	}

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		if err := c.ReadUntilClosed(); err != nil {
			log.Fatalln("read until closed loop:", err)
		}
		wg.Done()
	}()

	if err := c.Do(stun.MustBuild(stun.TransactionID, stun.BindingRequest), func(res *stun.Message) error {
		var xorAddr stun.XORMappedAddress
		if err := xorAddr.GetFrom(res); err != nil {
			return err
		}
		fmt.Println(xorAddr)
		return nil
	}); err != nil {
		log.Fatal("do:", err)
	}
	c.Close()
	wg.Wait()
}
