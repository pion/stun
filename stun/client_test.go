package stun

import (
	"fmt"
	"testing"
	"time"
)

func TestClient_Request(t *testing.T) {
	client, err := NewClient("udp", "stun.l.google.com:19302", time.Second*5)
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
}
