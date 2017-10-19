package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/go-rtc/stun"
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
		log.Fatalln("too many attempts to resolve:", err)
	}

	fmt.Println("DIALING", addr)
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Fatalln("failed to dial:", err)
	}
	client := stun.NewClient(stun.ClientOptions{
		Connection: conn,
	})
	laddr := conn.LocalAddr()
	fmt.Println("LISTEN ON", laddr)

	request, err := stun.Build(stun.BindingRequest, stun.TransactionID)
	if err != nil {
		log.Fatalln("failed to build:", err)
	}
	timeout := time.Second
	deadline := time.Now().Add(timeout)
	if err := client.Do(request, deadline, func(event stun.AgentEvent) {
		if event.Error != nil {
			log.Fatalln("got event with error:", event.Error)
		}
		response := event.Message
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
	}); err != nil {
		log.Fatalln("failed to Do:", err)
	}
	if err := client.Close(); err != nil {
		log.Fatalln("failed to close client:", err)
	}
}
