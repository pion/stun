// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

// This package implements RFC5780's tests:
// - 4.3.  Determining NAT Mapping Behavior
// - 4.4.  Determining NAT Filtering Behavior
package main

import (
	"errors"
	"flag"
	"net"
	"os"
	"time"

	"github.com/pion/logging"
	"github.com/pion/stun/v3"
)

type stunServerConn struct {
	conn        net.PacketConn
	LocalAddr   net.Addr
	RemoteAddr  *net.UDPAddr
	OtherAddr   *net.UDPAddr
	messageChan chan *stun.Message
}

func (c *stunServerConn) Close() error {
	return c.conn.Close()
}

var (
	//nolint:gochecknoglobals
	addrStrPtr = flag.String("server", "stun.voipgate.com:3478", "STUN server address")
	//nolint:gochecknoglobals
	timeoutPtr = flag.Int("timeout", 3, "the number of seconds to wait for STUN server's response")
	//nolint:gochecknoglobals
	verbose = flag.Int("verbose", 1, "the verbosity level")
	//nolint:gochecknoglobals
	log logging.LeveledLogger
)

const (
	messageHeaderSize = 20
)

var (
	errResponseMessage = errors.New("error reading from response message channel")
	errTimedOut        = errors.New("timed out waiting for response")
	errNoOtherAddress  = errors.New("no OTHER-ADDRESS in message")
)

func main() {
	flag.Parse()

	var logLevel logging.LogLevel
	switch *verbose {
	case 0:
		logLevel = logging.LogLevelWarn
	case 1:
		logLevel = logging.LogLevelInfo // default
	case 2:
		logLevel = logging.LogLevelDebug
	case 3:
		logLevel = logging.LogLevelTrace
	}
	log = logging.NewDefaultLeveledLoggerForScope("", logLevel, os.Stdout)

	if err := mappingTests(*addrStrPtr); err != nil {
		log.Warn("NAT mapping behavior: inconclusive")
	}
	if err := filteringTests(*addrStrPtr); err != nil {
		log.Warn("NAT filtering behavior: inconclusive")
	}
}

// RFC5780: 4.3.  Determining NAT Mapping Behavior.
func mappingTests(addrStr string) error { //nolint:cyclop
	mapTestConn, err := connect(addrStr)
	if err != nil {
		log.Warnf("Error creating STUN connection: %s", err)

		return err
	}

	// Test I: Regular binding request
	log.Info("Mapping Test I: Regular binding request")
	request := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	resp, err := mapTestConn.roundTrip(request, mapTestConn.RemoteAddr)
	if err != nil {
		return err
	}

	// Parse response message for XOR-MAPPED-ADDRESS and make sure OTHER-ADDRESS valid
	resps1 := parse(resp)
	if resps1.xorAddr == nil || resps1.otherAddr == nil {
		log.Info("Error: NAT discovery feature not supported by this server")

		return errNoOtherAddress
	}
	addr, err := net.ResolveUDPAddr("udp4", resps1.otherAddr.String())
	if err != nil {
		log.Infof("Failed resolving OTHER-ADDRESS: %v", resps1.otherAddr)

		return err
	}
	mapTestConn.OtherAddr = addr
	log.Infof("Received XOR-MAPPED-ADDRESS: %v", resps1.xorAddr)

	// Assert mapping behavior
	if resps1.xorAddr.String() == mapTestConn.LocalAddr.String() {
		log.Warn("=> NAT mapping behavior: endpoint independent (no NAT)")

		return nil
	}

	// Test II: Send binding request to the other address but primary port
	log.Info("Mapping Test II: Send binding request to the other address but primary port")
	oaddr := *mapTestConn.OtherAddr
	oaddr.Port = mapTestConn.RemoteAddr.Port
	resp, err = mapTestConn.roundTrip(request, &oaddr)
	if err != nil {
		return err
	}

	// Assert mapping behavior
	resps2 := parse(resp)
	log.Infof("Received XOR-MAPPED-ADDRESS: %v", resps2.xorAddr)
	if resps2.xorAddr.String() == resps1.xorAddr.String() {
		log.Warn("=> NAT mapping behavior: endpoint independent")

		return nil
	}

	// Test III: Send binding request to the other address and port
	log.Info("Mapping Test III: Send binding request to the other address and port")
	resp, err = mapTestConn.roundTrip(request, mapTestConn.OtherAddr)
	if err != nil {
		return err
	}

	// Assert mapping behavior
	resps3 := parse(resp)
	log.Infof("Received XOR-MAPPED-ADDRESS: %v", resps3.xorAddr)
	if resps3.xorAddr.String() == resps2.xorAddr.String() {
		log.Warn("=> NAT mapping behavior: address dependent")
	} else {
		log.Warn("=> NAT mapping behavior: address and port dependent")
	}

	return mapTestConn.Close()
}

// RFC5780: 4.4.  Determining NAT Filtering Behavior.
func filteringTests(addrStr string) error { //nolint:cyclop
	mapTestConn, err := connect(addrStr)
	if err != nil {
		log.Warnf("Error creating STUN connection: %s", err)

		return err
	}

	// Test I: Regular binding request
	log.Info("Filtering Test I: Regular binding request")
	request := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	resp, err := mapTestConn.roundTrip(request, mapTestConn.RemoteAddr)
	if err != nil || errors.Is(err, errTimedOut) {
		return err
	}
	resps := parse(resp)
	if resps.xorAddr == nil || resps.otherAddr == nil {
		log.Warn("Error: NAT discovery feature not supported by this server")

		return errNoOtherAddress
	}
	addr, err := net.ResolveUDPAddr("udp4", resps.otherAddr.String())
	if err != nil {
		log.Infof("Failed resolving OTHER-ADDRESS: %v", resps.otherAddr)

		return err
	}
	mapTestConn.OtherAddr = addr

	// Test II: Request to change both IP and port
	log.Info("Filtering Test II: Request to change both IP and port")
	request = stun.MustBuild(stun.TransactionID, stun.BindingRequest)
	request.Add(stun.AttrChangeRequest, []byte{0x00, 0x00, 0x00, 0x06})

	resp, err = mapTestConn.roundTrip(request, mapTestConn.RemoteAddr)
	if err == nil {
		parse(resp) // just to print out the resp
		log.Warn("=> NAT filtering behavior: endpoint independent")

		return nil
	} else if !errors.Is(err, errTimedOut) {
		return err // something else went wrong
	}

	// Test III: Request to change port only
	log.Info("Filtering Test III: Request to change port only")
	request = stun.MustBuild(stun.TransactionID, stun.BindingRequest)
	request.Add(stun.AttrChangeRequest, []byte{0x00, 0x00, 0x00, 0x02})

	resp, err = mapTestConn.roundTrip(request, mapTestConn.RemoteAddr)
	if err == nil {
		parse(resp) // just to print out the resp
		log.Warn("=> NAT filtering behavior: address dependent")
	} else if errors.Is(err, errTimedOut) {
		log.Warn("=> NAT filtering behavior: address and port dependent")
	}

	return mapTestConn.Close()
}

// Parse a STUN message.
func parse(msg *stun.Message) (ret struct {
	xorAddr    *stun.XORMappedAddress
	otherAddr  *stun.OtherAddress
	respOrigin *stun.ResponseOrigin
	mappedAddr *stun.MappedAddress
	software   *stun.Software
},
) {
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
	log.Debugf("%v", msg)
	log.Debugf("\tMAPPED-ADDRESS:     %v", ret.mappedAddr)
	log.Debugf("\tXOR-MAPPED-ADDRESS: %v", ret.xorAddr)
	log.Debugf("\tRESPONSE-ORIGIN:    %v", ret.respOrigin)
	log.Debugf("\tOTHER-ADDRESS:      %v", ret.otherAddr)
	log.Debugf("\tSOFTWARE: %v", ret.software)
	for _, attr := range msg.Attributes {
		switch attr.Type {
		case
			stun.AttrXORMappedAddress,
			stun.AttrOtherAddress,
			stun.AttrResponseOrigin,
			stun.AttrMappedAddress,
			stun.AttrSoftware:
			break //nolint:staticcheck
		default:
			log.Debugf("\t%v (l=%v)", attr, attr.Length)
		}
	}

	return ret
}

// Given an address string, returns a StunServerConn.
func connect(addrStr string) (*stunServerConn, error) {
	log.Infof("Connecting to STUN server: %s", addrStr)
	addr, err := net.ResolveUDPAddr("udp4", addrStr)
	if err != nil {
		log.Warnf("Error resolving address: %s", err)

		return nil, err
	}

	c, err := net.ListenUDP("udp4", nil)
	if err != nil {
		return nil, err
	}
	log.Infof("Local address: %s", c.LocalAddr())
	log.Infof("Remote address: %s", addr.String())

	mChan := listen(c)

	return &stunServerConn{
		conn:        c,
		LocalAddr:   c.LocalAddr(),
		RemoteAddr:  addr,
		messageChan: mChan,
	}, nil
}

// Send request and wait for response or timeout.
func (c *stunServerConn) roundTrip(msg *stun.Message, addr net.Addr) (*stun.Message, error) {
	_ = msg.NewTransactionID()
	log.Infof("Sending to %v: (%v bytes)", addr, msg.Length+messageHeaderSize)
	log.Debugf("%v", msg)
	for _, attr := range msg.Attributes {
		log.Debugf("\t%v (l=%v)", attr, attr.Length)
	}
	_, err := c.conn.WriteTo(msg.Raw, addr)
	if err != nil {
		log.Warnf("Error sending request to %v", addr)

		return nil, err
	}

	// Wait for response or timeout
	select {
	case m, ok := <-c.messageChan:
		if !ok {
			return nil, errResponseMessage
		}

		return m, nil
	case <-time.After(time.Duration(*timeoutPtr) * time.Second):
		log.Infof("Timed out waiting for response from server %v", addr)

		return nil, errTimedOut
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
			log.Infof("Response from %v: (%v bytes)", addr, n)
			buf = buf[:n]

			m := new(stun.Message)
			m.Raw = buf
			err = m.Decode()
			if err != nil {
				log.Infof("Error decoding message: %v", err)
				close(messages)

				return
			}

			messages <- m
		}
	}()

	return
}
