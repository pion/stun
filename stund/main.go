package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/cydev/stun"
)

var (
	network = flag.String("net", "udp", "network to listen")
	address = flag.String("addr", "0.0.0.0:3478", "address to listen")
)

func main() {
	flag.Parse()
	switch *network {
	case "udp":
		normalized := stun.Normalize(*address)
		fmt.Println("cydev/stun listening on", normalized, "via", *network)
		log.Fatal(stun.ListenUDPAndServe(*network, *address))
	default:
		log.Fatal("unsupported network:", *network)
	}
}
