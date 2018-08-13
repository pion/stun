package main

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/gortc/stun"
)

func test(network string) {
	addr := resolve(network)
	fmt.Println("START", strings.ToUpper(addr.Network()))
	var (
		nonce stun.Nonce
		realm stun.Realm
	)
	const (
		username = "user"
		password = "secret"
	)
	conn, err := net.Dial(addr.Network(), addr.String())
	if err != nil {
		log.Fatalln("failed to dial conn:", err)
	}
	var options []stun.ClientOption
	if network == "tcp" {
		// Switching to "NO-RTO" mode.
		fmt.Println("using WithNoRetransmit for TCP")
		options = append(options, stun.WithNoRetransmit)
	}
	client, err := stun.NewClient(conn, options...)
	if err != nil {
		log.Fatal(err)
	}
	// First request should error.
	request, err := stun.Build(stun.BindingRequest, stun.TransactionID, stun.Fingerprint)
	if err != nil {
		log.Fatalln("failed to build:", err)
	}
	if err = client.Do(request, func(event stun.Event) {
		if event.Error != nil {
			log.Fatalln("got event with error:", event.Error)
		}
		response := event.Message
		if response.Type != stun.BindingError {
			log.Fatalln("bad message", response)
		}
		var errCode stun.ErrorCodeAttribute
		if codeErr := errCode.GetFrom(response); codeErr != nil {
			log.Fatalln("failed to get error code:", codeErr)
		}
		if errCode.Code != stun.CodeUnauthorised {
			log.Fatalln("unexpected error code:", errCode)
		}
		if parseErr := response.Parse(&nonce, &realm); parseErr != nil {
			log.Fatalln("failed to parse:", parseErr)
		}
		fmt.Println("Got nonce", nonce, "and realm", realm)
	}); err != nil {
		log.Fatalln("failed to Do:", err)
	}

	// Authenticating and sending second request.
	request, err = stun.Build(stun.TransactionID, stun.BindingRequest,
		stun.NewUsername(username), nonce, realm,
		stun.NewLongTermIntegrity(username, realm.String(), password),
		stun.Fingerprint,
	)
	if err != nil {
		log.Fatalln(err)
	}
	if err = client.Do(request, func(event stun.Event) {
		if event.Error != nil {
			log.Fatalln("got event with error:", event.Error)
		}
		response := event.Message
		if response.Type != stun.BindingSuccess {
			var errCode stun.ErrorCodeAttribute
			if codeErr := errCode.GetFrom(response); codeErr != nil {
				log.Fatalln("failed to get error code:", codeErr)
			}
			log.Fatalln("bad message", response, errCode)
		}
		var xorMapped stun.XORMappedAddress
		if err = response.Parse(&xorMapped); err != nil {
			log.Fatalln("failed to parse xor mapped address:", err)
		}
		if conn.LocalAddr().String() != xorMapped.String() {
			log.Fatalln(conn.LocalAddr(), "!=", xorMapped)
		}
		fmt.Println("OK", response, "GOT", xorMapped)
	}); err != nil {
		log.Fatalln("failed to Do:", err)
	}
	if err := client.Close(); err != nil {
		log.Fatalln("failed to close client:", err)
	}
	fmt.Println("OK", strings.ToUpper(addr.Network()))
}

func resolve(network string) net.Addr {
	addr := fmt.Sprintf("%s:%d", "stun-server", stun.DefaultPort)
	var (
		resolved   net.Addr
		resolveErr error
	)
	for i := 0; i < 10; i++ {
		switch network {
		case "udp":
			resolved, resolveErr = net.ResolveUDPAddr("udp", addr)
		case "tcp":
			resolved, resolveErr = net.ResolveTCPAddr("tcp", addr)
		default:
			panic("unknown network")
		}
		if resolveErr == nil {
			return resolved
		}
		time.Sleep(time.Millisecond * 300 * time.Duration(i))
	}
	panic(resolveErr)
}

func main() {
	test("udp")
	test("tcp")
}
