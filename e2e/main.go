package main

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/pion/stun"
)

func test(network string) {
	addr := resolve(network)
	fmt.Println("START", strings.ToUpper(addr.Network())) // nolint
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
		log.Fatalln("failed to dial conn:", err) // nolint
	}
	var options []stun.ClientOption
	if network == "tcp" {
		// Switching to "NO-RTO" mode.
		fmt.Println("using WithNoRetransmit for TCP") // nolint
		options = append(options, stun.WithNoRetransmit)
	}
	client, err := stun.NewClient(conn, options...)
	if err != nil {
		log.Fatal(err) // nolint
	}
	// First request should error.
	request, err := stun.Build(stun.BindingRequest, stun.TransactionID, stun.Fingerprint)
	if err != nil {
		log.Fatalln("failed to build:", err) // nolint
	}
	if err = client.Do(request, func(event stun.Event) {
		if event.Error != nil {
			log.Fatalln("got event with error:", event.Error) // nolint
		}
		response := event.Message
		if response.Type != stun.BindingError {
			log.Fatalln("bad message", response) // nolint
		}
		var errCode stun.ErrorCodeAttribute
		if codeErr := errCode.GetFrom(response); codeErr != nil {
			log.Fatalln("failed to get error code:", codeErr) // nolint
		}
		if errCode.Code != stun.CodeUnauthorized {
			log.Fatalln("unexpected error code:", errCode) // nolint
		}
		if parseErr := response.Parse(&nonce, &realm); parseErr != nil {
			log.Fatalln("failed to parse:", parseErr) // nolint
		}
		fmt.Println("Got nonce", nonce, "and realm", realm) // nolint
	}); err != nil {
		log.Fatalln("failed to Do:", err) // nolint
	}

	// Authenticating and sending second request.
	request, err = stun.Build(stun.TransactionID, stun.BindingRequest,
		stun.NewUsername(username), nonce, realm,
		stun.NewLongTermIntegrity(username, realm.String(), password),
		stun.Fingerprint,
	)
	if err != nil {
		log.Fatalln(err) // nolint
	}
	if err = client.Do(request, func(event stun.Event) {
		if event.Error != nil {
			log.Fatalln("got event with error:", event.Error) // nolint
		}
		response := event.Message
		if response.Type != stun.BindingSuccess {
			var errCode stun.ErrorCodeAttribute
			if codeErr := errCode.GetFrom(response); codeErr != nil {
				log.Fatalln("failed to get error code:", codeErr) // nolint
			}
			log.Fatalln("bad message", response, errCode) // nolint
		}
		var xorMapped stun.XORMappedAddress
		if err = response.Parse(&xorMapped); err != nil {
			log.Fatalln("failed to parse xor mapped address:", err) // nolint
		}
		if conn.LocalAddr().String() != xorMapped.String() {
			log.Fatalln(conn.LocalAddr(), "!=", xorMapped) // nolint
		}
		fmt.Println("OK", response, "GOT", xorMapped) // nolint
	}); err != nil {
		log.Fatalln("failed to Do:", err) // nolint
	}
	if err := client.Close(); err != nil {
		log.Fatalln("failed to close client:", err) // nolint
	}
	fmt.Println("OK", strings.ToUpper(addr.Network())) // nolint
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
			panic("unknown network") // nolint
		}
		if resolveErr == nil {
			return resolved
		}
		time.Sleep(time.Millisecond * 300 * time.Duration(i))
	}
	panic(resolveErr) // nolint
}

func main() {
	test("udp")
	test("tcp")
}
