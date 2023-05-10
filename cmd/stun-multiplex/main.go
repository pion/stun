// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

// Command stun-multiplex is example of doing UDP connection multiplexing
// that splits incoming UDP packets to two streams, "STUN Data" and
// "Application Data".
package main

import (
	"flag"
	"io"
	"log"
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
		if err := c.Do(stun.MustBuild(stun.TransactionID, stun.BindingRequest), func(res stun.Event) {
			if res.Error != nil {
				log.Panicf("Failed STUN transaction: %s", res.Error)
			}
		}); err != nil {
			log.Panicf("Failed STUN transaction: %s", err)
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
			log.Panicf("Failed to read: %s", err)
		}

		// De-multiplexing incoming packets.
		if stun.IsMessage(buf[:n]) {
			// If buf looks like STUN message, send it to STUN client connection.
			if _, err = stunConn.Write(buf[:n]); err != nil {
				log.Panicf("Failed to write: %s", err)
			}
		} else {
			// If not, it is application data.
			log.Printf("Demultiplex: [%s]: %s", raddr, buf[:n])
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
			log.Panicf("Failed to read: %s", err)
		}
		if _, err = conn.WriteTo(buf[:n], stunAddr); err != nil {
			log.Panicf("Failed to write: %s", err)
		}
	}
}

var stunServer = flag.String("stun", "stun.l.google.com:19302", "STUN Server to use") //nolint:gochecknoglobals

func main() {
	isServer := flag.Arg(0) == ""

	flag.Parse()

	// Allocating local UDP socket that will be used both for STUN and
	// our application data.
	addr, err := net.ResolveUDPAddr("udp4", "0.0.0.0:0")
	if err != nil {
		log.Panicf("Failed to resolve: %s", err)
	}

	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		log.Panicf("Failed to listen: %s", err)
	}

	// Resolving STUN server address.
	stunAddr, err := net.ResolveUDPAddr("udp4", *stunServer)
	if err != nil {
		log.Panicf("Failed to resolve '%s': %s", *stunServer, err)
	}

	log.Printf("Local address: %s", conn.LocalAddr())
	log.Printf("STUN server address: %s", stunAddr)

	stunL, stunR := net.Pipe()

	c, err := stun.NewClient(stunR)
	if err != nil {
		log.Panicf("Failed to create client: %s", err)
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
	if err = c.Do(stun.MustBuild(stun.TransactionID, stun.BindingRequest), func(res stun.Event) {
		if res.Error != nil {
			log.Panicf("Failed STUN transaction: %s", res.Error)
		}
		var xorAddr stun.XORMappedAddress
		if getErr := xorAddr.GetFrom(res.Message); getErr != nil {
			log.Panicf("Failed to get XOR-MAPPED-ADDRESS: %s", getErr)
		}
		copyAddr(&gotAddr, xorAddr)
	}); err != nil {
		log.Panicf("Failed STUN transaction: %s", err)
	}

	log.Printf("Public address: %s", gotAddr)

	// Keep-alive is needed to keep our NAT port allocated.
	// Any ping-pong will work, but we are just making binding requests.
	// Note that STUN Server is not mandatory for keep alive, application
	// data will keep alive that binding too.
	go keepAlive(c)

	notify := make(chan os.Signal, 1)
	signal.Notify(notify, os.Interrupt, syscall.SIGTERM)
	if isServer {
		log.Printf("Acting as server. Use following command to connect: %s %s", os.Args[0], gotAddr)

		for {
			select {
			case m := <-messages:
				if _, err = conn.WriteTo([]byte(m.text), m.addr); err != nil {
					log.Panicf("Failed to write: %s", err)
				}
			case <-notify:
				log.Println("Stopping")
				return
			}
		}
	} else {
		peerAddr, err := net.ResolveUDPAddr("udp4", flag.Arg(0))
		if err != nil {
			log.Panicf("Failed to resolve '%s': %s", flag.Arg(0), err)
		}

		log.Printf("Acting as client. Connecting to %s", peerAddr)

		msg := "Hello peer"

		sendMsg := func() {
			log.Printf("Writing to: %s", peerAddr)
			if _, err = conn.WriteTo([]byte(msg), peerAddr); err != nil {
				log.Panicf("Failed to write: %s", err)
			}
		}

		sendMsg()

		deadline := time.After(time.Second * 10)

		for {
			select {
			case <-deadline:
				log.Fatal("Failed to connect: deadline reached.")

			case <-time.After(time.Second):
				// Retry.
				sendMsg()

			case m := <-messages:
				log.Printf("Got response from %s: %s", m.addr, m.text)
				return

			case <-notify:
				log.Print("Stopping")
				return
			}
		}
	}
}
