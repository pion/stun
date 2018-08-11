package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/gortc/stun"
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
	client, err := stun.NewClient(stun.ClientOptions{
		Connection: conn,
	})
	if err != nil {
		log.Fatal(err)
	}
	laddr := conn.LocalAddr()
	fmt.Println("LISTEN ON", laddr)

	request, err := stun.Build(stun.BindingRequest, stun.TransactionID, stun.Fingerprint)
	if err != nil {
		log.Fatalln("failed to build:", err)
	}
	var (
		nonce stun.Nonce
		realm stun.Realm
	)
	const (
		username = "user"
		password = "secret"
	)

	// First request should error.
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

	request, err = stun.Build(stun.TransactionID, stun.BindingRequest,
		stun.NewUsername(username), nonce, realm,
		stun.NewLongTermIntegrity(username, realm.String(), password),
		stun.Fingerprint,
	)
	if err != nil {
		log.Fatalln("failed to build:", err)
	}
	if err = client.Do(request, func(event stun.Event) {
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

	// Trying to use TCP.
	var (
		tcpAddr *net.TCPAddr
	)
	fmt.Println("TCP START")
	for i := 0; i < 10; i++ {
		tcpAddr, err = net.ResolveTCPAddr("tcp", fmt.Sprintf("stun-server:%d", stun.DefaultPort))
		if err == nil {
			break
		}
		time.Sleep(time.Millisecond * 300 * time.Duration(i))
	}
	if err != nil {
		log.Fatalln("too many attempts to resolve:", err)
	}
	tcpConn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		log.Fatalln("failed to dial:", err)
	}
	tcpLocalAddr := tcpConn.LocalAddr()
	fmt.Println("TCP LISTEN ON", tcpConn.LocalAddr(), "TO", tcpConn.RemoteAddr())
	client, err = stun.NewClient(stun.ClientOptions{
		Connection: tcpConn,
	})
	if err != nil {
		log.Fatalln("failed to create tcp client:", err)
	}
	// First request should error.
	request, err = stun.Build(stun.BindingRequest, stun.TransactionID, stun.Fingerprint)
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
		if tcpLocalAddr.String() != xorMapped.String() {
			log.Fatalln(tcpLocalAddr, "!=", xorMapped)
		}
		fmt.Println("OK", response, "GOT", xorMapped)
	}); err != nil {
		log.Fatalln("failed to Do:", err)
	}
	if err := client.Close(); err != nil {
		log.Fatalln("failed to close client:", err)
	}
}
