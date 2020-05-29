package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/pion/stun"
)

type StunServerConn struct {
	conn        net.PacketConn
	PrimaryAddr *net.UDPAddr
	OtherAddr   *net.UDPAddr
	messageChan chan *stun.Message
}

func (c *StunServerConn) Close() {
	c.conn.Close()
}

var (
	addrStrPtr  = flag.String("server", "stun.voip.blackberry.com:3478", "STUN server address")
	ErrTimedOut = errors.New("timed out waiting for response")
)

func main() {
	flag.Parse()
	log.Printf("Connecting to STUN server: %s", *addrStrPtr)

	if err := mappingTests(*addrStrPtr); err != nil {
		log.Println("Results inconclusive.")
		return
	}
	if err := filteringTests(*addrStrPtr); err != nil {
		log.Println("Results inconclusive.")
		return
	}
}

func mappingTests(addrStr string) error {
	var xorAddr1 stun.XORMappedAddress
	var xorAddr2 stun.XORMappedAddress

	mapTestConn, err := Connect(addrStr)
	if err != nil {
		log.Printf("Error creating STUN connection: %s", err.Error())
		return err
	}

	defer mapTestConn.Close()

	// Test I: Regular binding request
	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	resp, err := mapTestConn.RoundTrip(message, mapTestConn.PrimaryAddr)
	if err == ErrTimedOut {
		log.Printf("Error: no response from server")
		return err
	}
	if err != nil {
		log.Printf("Error receiving response from server: %s", err.Error())
		return err
	}

	// Decoding XOR-MAPPED-ADDRESS attribute from message.
	if err = xorAddr1.GetFrom(resp); err != nil {
		log.Printf("Error retrieving XOR-MAPPED-ADDRESS resonse: %s", err.Error())
		return err
	}

	log.Printf("Received xormapped address: %s\t", xorAddr1.String())

	// Decoding OTHER-ADDRESS attribute from message.
	var otherAddr stun.OtherAddress
	if err = otherAddr.GetFrom(resp); err != nil {
		log.Println("NAT discovery feature not supported by this server")
		return err
	}

	if err = mapTestConn.AddOtherAddr(otherAddr.String()); err != nil {
		log.Printf("Failed to resolve address %s\t", otherAddr.String())
		return err
	}

	// Test II: Send binding request to other address
	resp, err = mapTestConn.RoundTrip(message, mapTestConn.OtherAddr)
	if err == ErrTimedOut {
		log.Printf("Error: no response from server")
		return err
	}
	if err != nil {
		log.Printf("Error retrieving server response: %s", err.Error())
		return nil
	}

	// Decoding XOR-MAPPED-ADDRESS attribute from message.
	if err = xorAddr2.GetFrom(resp); err != nil {
		log.Printf("Error retrieving XOR-MAPPED-ADDRESS resonse: %s", err.Error())
		return err
	}
	log.Printf("Received xormapped address: %s\t", xorAddr2.String())

	if xorAddr1.String() == xorAddr2.String() {
		log.Printf("NAT mapping behavior: endpoint-independent")
	} else {
		log.Printf("NAT mapping behavior: address-dependent")
	}
	return nil
}

func filteringTests(addrStr string) error {
	var xorAddr stun.XORMappedAddress

	mapTestConn, err := Connect(addrStr)
	if err != nil {
		log.Printf("Error creating STUN connection: %s", err.Error())
		return err
	}

	defer mapTestConn.Close()

	// Test I: Regular binding request
	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	resp, err := mapTestConn.RoundTrip(message, mapTestConn.PrimaryAddr)
	if err == ErrTimedOut {
		log.Printf("Error: no response from server")
		return err
	}
	if err != nil {
		log.Printf("Error: %s", err.Error())
		return err
	}

	// Decoding XOR-MAPPED-ADDRESS attribute from message.
	if err = xorAddr.GetFrom(resp); err != nil {
		log.Printf("Error retrieving XOR-MAPPED-ADDRESS from resonse: %s", err.Error())
		return err
	}

	log.Printf("Received xormapped address: %s\t", xorAddr.String())

	// Test II: Request to change both IP and port
	message.Add(stun.AttrChangeRequest, []byte{0x00, 0x00, 0x00, 0x06})

	_, err = mapTestConn.RoundTrip(message, mapTestConn.PrimaryAddr)
	if err == nil {
		log.Printf("NAT filtering behavior: endpoint-independent")
		return nil
	}
	if err != ErrTimedOut {
		// something else went wrong
		log.Printf("Error reading response from server: %s", err.Error())
		return err
	}

	// Test III
	message.Add(stun.AttrChangeRequest, []byte{0x00, 0x00, 0x00, 0x02})

	_, err = mapTestConn.RoundTrip(message, mapTestConn.PrimaryAddr)
	if err == ErrTimedOut {
		log.Printf("NAT filtering behavior: address and port-dependent")
	}
	if err == nil {
		log.Printf("NAT filtering behavior: address-dependent")
	}
	if err != ErrTimedOut && err != nil {
		// something else went wrong
		log.Printf("Error reading response from server: %s", err.Error())
		return err
	}
	return nil
}

// Given an address string, returns a StunServerConn
func Connect(addrStr string) (*StunServerConn, error) {
	// Creating a "connection" to STUN server.
	addr, err := net.ResolveUDPAddr("udp4", addrStr)
	if err != nil {
		fmt.Printf("Error resolving address: %s\n", err.Error())
		return nil, err
	}

	c, err := net.ListenUDP("udp4", nil)
	if err != nil {
		return nil, err
	}
	log.Printf("Local address: %s\n", c.LocalAddr())

	mChan := listen(c)

	return &StunServerConn{
		conn:        c,
		PrimaryAddr: addr,
		messageChan: mChan,
	}, nil
}

func (c *StunServerConn) RoundTrip(msg *stun.Message, addr net.Addr) (*stun.Message, error) {
	_, err := c.conn.WriteTo(msg.Raw, addr)
	if err != nil {
		return nil, err
	}

	// Wait for response or timeout
	select {
	case m, ok := <-c.messageChan:
		if !ok {
			return nil, fmt.Errorf("error reading from messageChan")
		}
		return m, nil
	case <-time.After(30 * time.Second):
		return nil, ErrTimedOut
	}
}

func (c *StunServerConn) AddOtherAddr(addrStr string) error {
	addr2, err := net.ResolveUDPAddr("udp4", addrStr)
	if err != nil {
		return err
	}
	c.OtherAddr = addr2
	return nil
}

// taken from https://github.com/pion/stun/blob/master/cmd/stun-traversal/main.go
func listen(conn *net.UDPConn) chan *stun.Message {
	messages := make(chan *stun.Message)
	go func() {
		for {
			buf := make([]byte, 1024)

			n, _, err := conn.ReadFromUDP(buf)
			if err != nil {
				close(messages)
				return
			}
			buf = buf[:n]

			m := new(stun.Message)
			m.Raw = buf
			err = m.Decode()
			if err != nil {
				close(messages)
				return
			}

			messages <- m
		}
	}()
	return messages
}
