package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/ernado/stun"
)

func main() {
	var (
		addr *net.UDPAddr
		err  error
	)
	fmt.Println("START")
	for i := 0; i < 10; i++ {
		addr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("stun-server:%d", stun.DefaultPort))
		if err == nil {
			break
		}
		time.Sleep(time.Millisecond * 300 * time.Duration(i))
	}
	if err != nil {
		log.Println("too many attempts")
		log.Fatalln("unable to resolve addr:", err)
	}
	c, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Fatal(err)
	}
	laddr := c.LocalAddr()
	fmt.Println("LISTEN ON", laddr)
	defer c.Close()
	m, err := stun.Build(stun.BindingRequest, stun.TransactionID)
	if err != nil {
		log.Fatalln("failed to build:", err)
	}
	if _, err := m.WriteTo(c); err != nil {
		log.Fatalln("failed to write:", err)
	}
	response := new(stun.Message)
	response.Raw = make([]byte, 0, 1024)
	c.SetReadDeadline(time.Now().Add(time.Second * 5))
	if _, err := response.ReadFrom(c); err != nil {
		log.Fatalln("failed to read:", err)
	}
	if response.Type != stun.BindingSuccess {
		log.Fatalln("bad message", response)
	}
	var xorMapped stun.XORMappedAddress
	if err = response.Parse(&xorMapped); err != nil {
		log.Fatalln("failed to parse xor mapped address:", err)
	}
	if laddr.String() != xorMapped.String() {
		log.Fatalln(laddr, "!=", xorMapped)
	}
	fmt.Println("OK", response, "GOT", xorMapped)
}
