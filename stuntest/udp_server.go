// SPDX-FileCopyrightText: 2026 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

// Package stuntest contains helpers for testing STUN clients
package stuntest

import (
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

var errUDPServerUnsupportedNetwork = errors.New("unsupported network")

// NewUDPServer creates an udp server for testing.
// The supplied handler function will be called with the request
// and should be used to emulate the server behavior.
//
//nolint:cyclop
func NewUDPServer(
	t *testing.T,
	network string,
	maxMessageSize int,
	handler func(req []byte) ([]byte, error),
) (net.Addr, func(t *testing.T), error) {
	t.Helper()

	var ip string
	switch network {
	case "udp4":
		ip = "127.0.0.1"
	case "udp6":
		ip = "[::1]"
	default:
		return nil, nil, fmt.Errorf("%w: %s", errUDPServerUnsupportedNetwork, network)
	}

	udpConn, err := net.ListenUDP(network, &net.UDPAddr{IP: net.ParseIP(ip), Port: 0})
	assert.NoError(t, err)

	// Necessary for IPv6
	address := fmt.Sprintf("%s:%d", ip, udpConn.LocalAddr().(*net.UDPAddr).Port) //nolint:forcetypeassert
	serverAddr, err := net.ResolveUDPAddr(network, address)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve stun host: %s: %w", address, err)
	}

	errCh := make(chan error, 1)
	go func() {
		for {
			bs := make([]byte, maxMessageSize)
			n, addr, err := udpConn.ReadFrom(bs)
			if err != nil {
				errCh <- err

				return
			}

			resp, err := handler(bs[:n])
			if err != nil {
				errCh <- err

				return
			}

			_, err = udpConn.WriteTo(resp, addr)
			if err != nil {
				errCh <- err

				return
			}
		}
	}()

	return serverAddr, func(t *testing.T) {
		t.Helper()

		select {
		case err := <-errCh:
			if err != nil {
				assert.NoError(t, err)

				return
			}
		default:
		}

		assert.NoError(t, udpConn.Close())
		<-errCh
	}, nil
}
