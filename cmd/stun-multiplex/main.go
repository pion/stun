// Command stun-multiplex is example of doing UDP connection multiplexing
// that splits incoming UDP packets to two streams, "STUN Data" and
// "Application Data".
package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pion/stun"
)

func copyAddr(dst *stun.XORMappedAddress, src stun.XORMappedAddress) {
	dst.IP = append(dst.IP, src.IP...)
	dst.Port = src.Port
}

func keepAlive(c *stun.Client) {
	// Keep-alive for NAT binding.
	t := time.NewTicker(time.Second * 5)
	for range t.C {
		if err := c.Do(stun.MustBuild(stun.TransactionID(), stun.BindingRequest), func(res stun.Event) {
			if res.Error != nil {
				panic(res.Error)
			}
		}); err != nil {
			panic(err)
		}
	}
}

type message struct {
	text string
	addr net.Addr
}

func demultiplex(conn *net.UDPConn, stunConn io.Writer, messages chan message) {
	buf := make([]byte, 1024)
	for {
		n, raddr, err := conn.ReadFrom(buf)
		if err != nil {
			panic(err)
		}
		// De-multiplexing incoming packets.
		if stun.IsMessage(buf[:n]) {
			// If buf looks like STUN message, send it to STUN client connection.
			if _, err = stunConn.Write(buf[:n]); err != nil {
				panic(err)
			}
		} else {
			// If not, it is application data.
			fmt.Printf("demultiplex: [%s]: %s\n", raddr, buf[:n])
			messages <- message{
				text: string(buf[:n]),
				addr: raddr,
			}
		}
	}
}

func multiplex(conn *net.UDPConn, stunAddr net.Addr, stunConn io.Reader) {
	// Sending all data from stun client to stun server.
	buf := make([]byte, 1024)
	for {
		n, err := stunConn.Read(buf)
		if err != nil {
			panic(err)
		}
		if _, err = conn.WriteTo(buf[:n], stunAddr); err != nil {
			panic(err)
		}
	}
}

var stunServer = flag.String("stun", "stun.l.google.com:19302", "STUN Server to use") // nolint:gochecknoglobals

func main() {
	flag.Parse()
	// Allocating local UDP socket that will be used both for STUN and
	// our application data.
	addr, err := net.ResolveUDPAddr("udp4", "0.0.0.0:0")
	if err != nil {
		panic(err)
	}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		panic(err)
	}
	// Resolving STUN Server address.
	stunAddr, err := net.ResolveUDPAddr("udp4", *stunServer)
	if err != nil {
		panic(err)
	}
	fmt.Println("local addr:", conn.LocalAddr(), "stun server addr:", stunAddr)
	stunL, stunR := net.Pipe()
	c, err := stun.NewClient(stunR)
	if err != nil {
		panic(err)
	}
	// Starting multiplexing (writing back STUN messages) with de-multiplexing
	// (passing STUN messages to STUN client and processing application
	// data separately).
	//
	// stunL and stunR are virtual connections, see net.Pipe for reference.
	messages := make(chan message)
	go demultiplex(conn, stunL, messages)
	go multiplex(conn, stunAddr, stunL)

	// Getting our "real" IP address from STUN Server.
	// This will create a NAT binding on your provider/router NAT Server,
	// and the STUN server will return allocated public IP for that binding.
	//
	// This can fail if your NAT Server is strict and will use separate ports
	// for application data and STUN
	var gotAddr stun.XORMappedAddress
	if err = c.Do(stun.MustBuild(stun.TransactionID(), stun.BindingRequest), func(res stun.Event) {
		if res.Error != nil {
			panic(res.Error)
		}
		var xorAddr stun.XORMappedAddress
		if getErr := xorAddr.GetFrom(res.Message); getErr != nil {
			panic(getErr)
		}
		copyAddr(&gotAddr, xorAddr)
	}); err != nil {
		panic(err)
	}
	fmt.Println("public addr:", gotAddr)

	// Keep-alive is needed to keep our NAT port allocated.
	// Any ping-pong will work, but we are just making binding requests.
	// Note that STUN Server is not mandatory for keep alive, application
	// data will keep alive that binding too.
	go keepAlive(c)

	notify := make(chan os.Signal, 1)
	signal.Notify(notify, os.Interrupt, syscall.SIGTERM)
	if flag.Arg(0) == "" {
		fmt.Println("Acting as server. Use following command to connect:")
		fmt.Println(os.Args[0], gotAddr)
		for {
			select {
			case m := <-messages:
				if _, err = conn.WriteTo([]byte(m.text), m.addr); err != nil {
					panic(err)
				}
			case <-notify:
				fmt.Println("\rStopping")
				return
			}
		}
	} else {
		peerAddr, err := net.ResolveUDPAddr("udp4", flag.Arg(0))
		if err != nil {
			panic(err)
		}
		fmt.Println("Acting as client. Connecting to", peerAddr)
		msg := "Hello peer"
		sendMsg := func() {
			fmt.Println("Writing", peerAddr)
			if _, err = conn.WriteTo([]byte(msg), peerAddr); err != nil {
				panic(err)
			}
		}
		sendMsg()
		deadline := time.After(time.Second * 10)
		for {
			select {
			case <-deadline:
				fmt.Println("Failed to connect: deadline reached.")
				os.Exit(2)
			case <-time.After(time.Second):
				// Retry.
				sendMsg()
			case m := <-messages:
				fmt.Printf("Got response from %s: %s\n", m.addr, m.text)
				return
			case <-notify:
				fmt.Println("\rStopping")
				return
			}
		}
	}
}
