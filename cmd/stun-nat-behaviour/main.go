// This cmd implements RFC5780's tests:
// - 4.3.  Determining NAT Mapping Behavior
// - 4.4.  Determining NAT Filtering Behavior
package main

import (
	"flag"
	"fmt"
	"net"
	"time"

	"github.com/pion/stun"
)

type StunServerConn struct {
	conn        net.PacketConn
	LocalAddr   net.Addr
	RemoteAddr  *net.UDPAddr
	OtherAddr   *net.UDPAddr
	messageChan chan *stun.Message
}

func (c *StunServerConn) Close() {
	c.conn.Close()
}

var (
	addrStrPtr = flag.String("server", "stun.voip.blackberry.com:3478", "STUN server address")
	timeoutPtr = flag.Int("timeout", 3, "the number of seconds to wait for STUN server's response")
)

type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrResponseMessage = Error("error reading from response message channel")
	ErrTimedOut        = Error("timed out waiting for response")
	ErrNoOtherAddress  = Error("no OTHER-ADDRESS in message")
	messageHeaderSize  = 20
)

func main() {
	flag.Parse()

	if err := MappingTests(*addrStrPtr); err != nil {
		fmt.Println("NAT mapping behavior: inconclusive")
	}
	if err := FilteringTests(*addrStrPtr); err != nil {
		fmt.Println("NAT filtering behavior: inconclusive")
	}
}

// RFC5780: 4.3.  Determining NAT Mapping Behavior
func MappingTests(addrStr string) error {
	mapTestConn, err := connect(addrStr)
	if err != nil {
		fmt.Printf("Error creating STUN connection: %s\n", err.Error())
		return err
	}
	defer mapTestConn.Close()

	// Test I: Regular binding request
	fmt.Println("\nMapping Test I: Regular binding request")
	request := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	resp, err := mapTestConn.roundTrip(request, mapTestConn.RemoteAddr)
	if err != nil {
		return err
	}

	// Parse response message for XOR-MAPPED-ADDRESS and make sure OTHER-ADDRESS valid
	resps1 := parse(resp)
	if resps1.xorAddr == nil || resps1.otherAddr == nil {
		fmt.Println("Error: NAT discovery feature not supported by this server")
		return ErrNoOtherAddress
	}
	addr, err := net.ResolveUDPAddr("udp4", resps1.otherAddr.String())
	if err != nil {
		fmt.Printf("Failed resolving OTHER-ADDRESS: %v\n", resps1.otherAddr)
		return err
	}
	mapTestConn.OtherAddr = addr
	fmt.Printf("Received XOR-MAPPED-ADDRESS: %v\n", resps1.xorAddr)

	// Assert mapping behavior
	if resps1.xorAddr.String() == mapTestConn.LocalAddr.String() {
		fmt.Println("\n=> NAT mapping behavior: endpoint independent (no NAT)")
		return nil
	}

	// Test II: Send binding request to the other address but primary port
	fmt.Println("\nMapping Test II: Send binding request to the other address but primary port")
	oaddr := *mapTestConn.OtherAddr
	oaddr.Port = mapTestConn.RemoteAddr.Port
	resp, err = mapTestConn.roundTrip(request, &oaddr)
	if err != nil {
		return err
	}

	// Assert mapping behavior
	resps2 := parse(resp)
	fmt.Printf("Received XOR-MAPPED-ADDRESS: %v\n", resps2.xorAddr)
	if resps2.xorAddr.String() == resps1.xorAddr.String() {
		fmt.Println("\n=> NAT mapping behavior: endpoint independent")
		return nil
	}

	// Test III: Send binding request to the other address and port
	fmt.Println("\nMapping Test III: Send binding request to the other address and port")
	resp, err = mapTestConn.roundTrip(request, mapTestConn.OtherAddr)
	if err != nil {
		return err
	}

	// Assert mapping behavior
	resps3 := parse(resp)
	fmt.Printf("Received XOR-MAPPED-ADDRESS: %v\n", resps3.xorAddr)
	if resps3.xorAddr.String() == resps2.xorAddr.String() {
		fmt.Println("\n=> NAT mapping behavior: address dependent")
	} else {
		fmt.Println("\n=> NAT mapping behavior: address and port dependent")
	}

	return nil
}

// RFC5780: 4.4.  Determining NAT Filtering Behavior
func FilteringTests(addrStr string) error {
	mapTestConn, err := connect(addrStr)
	if err != nil {
		fmt.Printf("Error creating STUN connection: %s\n", err.Error())
		return err
	}
	defer mapTestConn.Close()

	// Test I: Regular binding request
	fmt.Println("\nFiltering Test I: Regular binding request")
	request := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	resp, err := mapTestConn.roundTrip(request, mapTestConn.RemoteAddr)
	if err != nil || err == ErrTimedOut {
		return err
	}
	resps := parse(resp)
	if resps.xorAddr == nil || resps.otherAddr == nil {
		fmt.Println("Error: NAT discovery feature not supported by this server")
		return ErrNoOtherAddress
	}
	addr, err := net.ResolveUDPAddr("udp4", resps.otherAddr.String())
	if err != nil {
		fmt.Printf("Failed resolving OTHER-ADDRESS: %v\n", resps.otherAddr)
		return err
	}
	mapTestConn.OtherAddr = addr

	// Test II: Request to change both IP and port
	fmt.Println("\nFiltering Test II: Request to change both IP and port")
	request.Reset()
	request.Add(stun.AttrChangeRequest, []byte{0x00, 0x00, 0x00, 0x06})

	resp, err = mapTestConn.roundTrip(request, mapTestConn.RemoteAddr)
	if err == nil {
		parse(resp) // just to print out the resp
		fmt.Println("\n=> NAT filtering behavior: endpoint independent")
		return nil
	} else if err != ErrTimedOut {
		return err // something else went wrong
	}

	// Test III: Request to change port only
	fmt.Println("\nFiltering Test III: Request to change port only")
	request.Reset()
	request.Add(stun.AttrChangeRequest, []byte{0x00, 0x00, 0x00, 0x02})

	resp, err = mapTestConn.roundTrip(request, mapTestConn.RemoteAddr)
	if err == nil {
		parse(resp) // just to print out the resp
		fmt.Println("\n=> NAT filtering behavior: address dependent")
	} else if err == ErrTimedOut {
		fmt.Println("\n=> NAT filtering behavior: address and port dependent")
	}

	return nil
}

// Parse a STUN message
func parse(msg *stun.Message) (ret struct {
	xorAddr    *stun.XORMappedAddress
	otherAddr  *stun.OtherAddress
	respOrigin *stun.ResponseOrigin
	mappedAddr *stun.MappedAddress
	software   *stun.Software
}) {
	ret.mappedAddr = &stun.MappedAddress{}
	ret.xorAddr = &stun.XORMappedAddress{}
	ret.respOrigin = &stun.ResponseOrigin{}
	ret.otherAddr = &stun.OtherAddress{}
	ret.software = &stun.Software{}
	if ret.xorAddr.GetFrom(msg) != nil {
		ret.xorAddr = nil
	}
	if ret.otherAddr.GetFrom(msg) != nil {
		ret.otherAddr = nil
	}
	if ret.respOrigin.GetFrom(msg) != nil {
		ret.respOrigin = nil
	}
	if ret.mappedAddr.GetFrom(msg) != nil {
		ret.mappedAddr = nil
	}
	if ret.software.GetFrom(msg) != nil {
		ret.software = nil
	}
	fmt.Printf("%v\n", msg)
	fmt.Printf("\tMAPPED-ADDRESS:     %v\n", ret.mappedAddr)
	fmt.Printf("\tXOR-MAPPED-ADDRESS: %v\n", ret.xorAddr)
	fmt.Printf("\tRESPONSE-ORIGIN:    %v\n", ret.respOrigin)
	fmt.Printf("\tOTHER-ADDRESS:      %v\n", ret.otherAddr)
	fmt.Printf("\tSOFTWARE: %v\n", ret.software)
	for _, attr := range msg.Attributes {
		switch attr.Type {
		case
			stun.AttrXORMappedAddress,
			stun.AttrOtherAddress,
			stun.AttrResponseOrigin,
			stun.AttrMappedAddress,
			stun.AttrSoftware:
			break
		default:
			fmt.Printf("\t%v (l=%v)\n", attr, attr.Length)
		}
	}
	return ret
}

// Given an address string, returns a StunServerConn
func connect(addrStr string) (*StunServerConn, error) {
	fmt.Printf("\nconnecting to STUN server: %s\n", addrStr)
	addr, err := net.ResolveUDPAddr("udp4", addrStr)
	if err != nil {
		fmt.Printf("Error resolving address: %s\n", err.Error())
		return nil, err
	}

	c, err := net.ListenUDP("udp4", nil)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Local address: %s\n", c.LocalAddr())
	fmt.Printf("Remote address: %s\n", addr.String())

	mChan := listen(c)

	return &StunServerConn{
		conn:        c,
		LocalAddr:   c.LocalAddr(),
		RemoteAddr:  addr,
		messageChan: mChan,
	}, nil
}

// Send request and wait for response or timeout
func (c *StunServerConn) roundTrip(msg *stun.Message, addr net.Addr) (*stun.Message, error) {
	_ = msg.NewTransactionID()
	fmt.Printf("Sending to %v: (%v bytes)\n", addr, msg.Length+messageHeaderSize)
	fmt.Printf("%v\n", msg)
	for _, attr := range msg.Attributes {
		fmt.Printf("\t%v (l=%v)\n", attr, attr.Length)
	}
	_, err := c.conn.WriteTo(msg.Raw, addr)
	if err != nil {
		fmt.Printf("Error sending request to %v\n", addr)
		return nil, err
	}

	// Wait for response or timeout
	select {
	case m, ok := <-c.messageChan:
		if !ok {
			return nil, ErrResponseMessage
		}
		return m, nil
	case <-time.After(time.Duration(*timeoutPtr) * time.Second):
		fmt.Printf("Timed out waiting for response from server %v\n", addr)
		return nil, ErrTimedOut
	}
}

// taken from https://github.com/pion/stun/blob/master/cmd/stun-traversal/main.go
func listen(conn *net.UDPConn) (messages chan *stun.Message) {
	messages = make(chan *stun.Message)
	go func() {
		for {
			buf := make([]byte, 1024)

			n, addr, err := conn.ReadFromUDP(buf)
			if err != nil {
				close(messages)
				return
			}
			fmt.Printf("Response from %v: (%v bytes)\n", addr, n)
			buf = buf[:n]

			m := new(stun.Message)
			m.Raw = buf
			err = m.Decode()
			if err != nil {
				fmt.Printf("Error decoding message: %v\n", err)
				close(messages)
				return
			}

			messages <- m
		}
	}()
	return
}
