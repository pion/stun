// SPDX-FileCopyrightText: 2026 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package stun

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRFC5769(t *testing.T) { //nolint:cyclop
	// Test Vectors for Session Traversal Utilities for NAT (STUN)
	// see https://tools.ietf.org/html/rfc5769
	t.Run("Request", func(t *testing.T) {
		// nolint
		m := &Message{
			Raw: []byte("\x00\x01\x00\x58" +
				"\x21\x12\xa4\x42" +
				"\xb7\xe7\xa7\x01\xbc\x34\xd6\x86\xfa\x87\xdf\xae" +
				"\x80\x22\x00\x10" +
				"STUN test client" +
				"\x00\x24\x00\x04" +
				"\x6e\x00\x01\xff" +
				"\x80\x29\x00\x08" +
				"\x93\x2f\xf9\xb1\x51\x26\x3b\x36" +
				"\x00\x06\x00\x09" +
				"\x65\x76\x74\x6a\x3a\x68\x36\x76\x59\x20\x20\x20" +
				"\x00\x08\x00\x14" +
				"\x9a\xea\xa7\x0c\xbf\xd8\xcb\x56\x78\x1e\xf2\xb5" +
				"\xb2\xd3\xf2\x49\xc1\xb5\x71\xa2" +
				"\x80\x28\x00\x04" +
				"\xe5\x7a\x3b\xcf",
			),
		}
		assert.NoError(t, m.Decode())
		software := new(Software)
		assert.NoError(t, software.GetFrom(m))
		assert.Equal(t, "STUN test client", software.String())
		assert.NoError(t, Fingerprint.Check(m))
		t.Run("Long-Term credentials", func(t *testing.T) {
			msg := &Message{
				Raw: []byte("\x00\x01\x00\x60" +
					"\x21\x12\xa4\x42" +
					"\x78\xad\x34\x33\xc6\xad\x72\xc0\x29\xda\x41\x2e" +
					"\x00\x06\x00\x12" +
					"\xe3\x83\x9e\xe3\x83\x88\xe3\x83\xaa\xe3\x83\x83" +
					"\xe3\x82\xaf\xe3\x82\xb9\x00\x00" +
					"\x00\x15\x00\x1c" +
					"\x66\x2f\x2f\x34\x39\x39\x6b\x39\x35\x34\x64\x36" +
					"\x4f\x4c\x33\x34\x6f\x4c\x39\x46\x53\x54\x76\x79" +
					"\x36\x34\x73\x41" +
					"\x00\x14\x00\x0b" +
					"\x65\x78\x61\x6d\x70\x6c\x65\x2e\x6f\x72\x67\x00" +
					"\x00\x08\x00\x14" +
					"\xf6\x70\x24\x65\x6d\xd6\x4a\x3e\x02\xb8\xe0\x71" +
					"\x2e\x85\xc9\xa2\x8c\xa8\x96\x66",
				),
			}
			assert.NoError(t, msg.Decode())
			u := new(Username)
			assert.NoError(t, u.GetFrom(msg))
			expectedUsername := "\u30DE\u30C8\u30EA\u30C3\u30AF\u30B9"
			assert.Equal(t, expectedUsername, u.String())
			n := new(Nonce)
			assert.NoError(t, n.GetFrom(msg))
			assert.Equal(t, "f//499k954d6OL34oL9FSTvy64sA", n.String())
			r := new(Realm)
			assert.NoError(t, r.GetFrom(msg))
			assert.Equal(t, "example.org", r.String())
			// checking HMAC
			i := NewLongTermIntegrity(
				"\u30DE\u30C8\u30EA\u30C3\u30AF\u30B9",
				"example.org",
				"TheMatrIX",
			)
			assert.NoError(t, i.Check(msg))
		})
	})
	t.Run("Response", func(t *testing.T) {
		t.Run("IPv4", func(t *testing.T) {
			msg := &Message{
				Raw: []byte("\x01\x01\x00\x3c" +
					"\x21\x12\xa4\x42" +
					"\xb7\xe7\xa7\x01\xbc\x34\xd6\x86\xfa\x87\xdf\xae" +
					"\x80\x22\x00\x0b" +
					"\x74\x65\x73\x74\x20\x76\x65\x63\x74\x6f\x72\x20" +
					"\x00\x20\x00\x08" +
					"\x00\x01\xa1\x47\xe1\x12\xa6\x43" +
					"\x00\x08\x00\x14" +
					"\x2b\x91\xf5\x99\xfd\x9e\x90\xc3\x8c\x74\x89\xf9" +
					"\x2a\xf9\xba\x53\xf0\x6b\xe7\xd7" +
					"\x80\x28\x00\x04" +
					"\xc0\x7d\x4c\x96",
				),
			}
			assert.NoError(t, msg.Decode())

			software := new(Software)
			assert.NoError(t, software.GetFrom(msg))
			assert.Equal(t, "test vector", software.String())
			assert.NoError(t, Fingerprint.Check(msg))
			addr := new(XORMappedAddress)
			assert.NoError(t, addr.GetFrom(msg))
			expected := "192.0.2.1"
			assert.Equalf(t, expected, addr.IP.String(), "Expected %s, got %s", expected, addr.IP)
			assert.Equal(t, 32853, addr.Port)
			assert.NoError(t, Fingerprint.Check(msg))
		})
		t.Run("IPv6", func(t *testing.T) {
			msg := &Message{
				Raw: []byte("\x01\x01\x00\x48" +
					"\x21\x12\xa4\x42" +
					"\xb7\xe7\xa7\x01\xbc\x34\xd6\x86\xfa\x87\xdf\xae" +
					"\x80\x22\x00\x0b" +
					"\x74\x65\x73\x74\x20\x76\x65\x63\x74\x6f\x72\x20" +
					"\x00\x20\x00\x14" +
					"\x00\x02\xa1\x47" +
					"\x01\x13\xa9\xfa\xa5\xd3\xf1\x79" +
					"\xbc\x25\xf4\xb5\xbe\xd2\xb9\xd9" +
					"\x00\x08\x00\x14" +
					"\xa3\x82\x95\x4e\x4b\xe6\x7b\xf1\x17\x84\xc9\x7c" +
					"\x82\x92\xc2\x75\xbf\xe3\xed\x41" +
					"\x80\x28\x00\x04" +
					"\xc8\xfb\x0b\x4c",
				),
			}
			assert.NoError(t, msg.Decode())
			software := new(Software)
			assert.NoError(t, software.GetFrom(msg))
			assert.Equal(t, "test vector", software.String())
			assert.NoError(t, Fingerprint.Check(msg))
			addr := new(XORMappedAddress)
			assert.NoError(t, addr.GetFrom(msg))
			expectedIP := "2001:db8:1234:5678:11:2233:4455:6677"
			assert.Truef(
				t, addr.IP.Equal(net.ParseIP(expectedIP)),
				"Expected %s, got %s", expectedIP, addr.IP,
			)
			assert.Equal(t, 32853, addr.Port)
			assert.NoError(t, Fingerprint.Check(msg))
		})
	})
}
