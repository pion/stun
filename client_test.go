package stun

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/pion/stun/test"
	"github.com/stretchr/testify/assert"
)

func TestClient_Request(t *testing.T) {

	testCases := []struct {
		network string
		url     string
	}{
		{"udp", "stun.l.google.com:19302"},
		{"udp", "u3.xirsys.com:3478"},
	}

	for _, testCase := range testCases {
		client, err := NewClient(testCase.network, testCase.url, time.Second*5)
		if err != nil {
			t.Fatalf("Failed to create STUN client: %v", err)
		}
		resp, err := client.Request()
		if err != nil {
			t.Fatalf("Failed to send a STUN Request to: %v", err)
		}
		attr, ok := resp.GetOneAttribute(AttrXORMappedAddress)
		if !ok {
			t.Fatalf("Failed to get XOR mapped address")
		}
		var addr XorAddress
		if err := addr.Unpack(resp, attr); err != nil {
			t.Fatalf("Unpacking created error: %#v", err.Error())
		}
		fmt.Printf("remote address: %s:%d\n", addr.IP.String(), addr.Port)
		fmt.Printf("local address: %s\n", client.conn.LocalAddr().String())
	}
}

func TestGetMappedAddress(t *testing.T) {
	for _, data := range []struct {
		name       string
		network    string
		localIP    string
		remoteIP   string
		remotePort int
	}{
		{
			name:       "udp4",
			network:    "udp4",
			localIP:    "127.0.0.1",
			remoteIP:   "1.2.3.4",
			remotePort: 1234,
		},
		{
			name:       "udp6",
			network:    "udp6",
			localIP:    "[::1]",
			remoteIP:   "c031:ca06:a453:e56d:cda:7122:a1d7:b01b",
			remotePort: 1234,
		},
	} {
		d := data
		t.Run(d.name, func(t *testing.T) {
			serverAddr, closeServer, err := test.NewUDPServer(
				t,
				d.network,
				maxMessageSize,
				func(req []byte) ([]byte, error) {
					msg, err := NewMessage(req)
					if err != nil {
						return nil, err
					}

					resp, err := Build(ClassSuccessResponse, MethodBinding, msg.TransactionID,
						&XorMappedAddress{XorAddress: XorAddress{IP: net.ParseIP(d.remoteIP), Port: d.remotePort}},
					)
					if err != nil {
						return nil, err
					}

					return resp.Pack(), nil
				})
			if err != nil {
				t.Fatal(err)
			}
			defer closeServer(t)

			conn, err := net.ListenUDP(d.network, &net.UDPAddr{IP: net.ParseIP(d.localIP), Port: 0})
			if err != nil {
				t.Fatal(err)
			}

			xoraddr, err := GetMappedAddressUDP(
				conn,
				serverAddr,
				time.Second*1,
			)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, d.remoteIP, xoraddr.IP.String())
			assert.Equal(t, d.remotePort, xoraddr.Port)
		})
	}
}
