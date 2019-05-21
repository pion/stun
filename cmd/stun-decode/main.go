package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/pion/stun"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", "stun-decode")
		fmt.Fprintln(os.Stderr, "stun-decode AAEAHCESpEJML0JTQWsyVXkwcmGALwAWaHR0cDovL2xvY2FsaG9zdDozMDAwLwAA")
		fmt.Fprintln(os.Stderr, "First argument must be a base64.StdEncoding-encoded message")
		flag.PrintDefaults()
	}
	flag.Parse()
	data, err := base64.StdEncoding.DecodeString(flag.Arg(0))
	if err != nil {
		log.Fatalln("Unable to decode bas64 value:", err)
	}
	m := new(stun.Message)
	m.Raw = data
	if err = m.Decode(); err != nil {
		log.Fatalln("Unable to decode message:", err)
	}
	fmt.Println(m)
}
