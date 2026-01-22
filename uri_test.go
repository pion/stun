// SPDX-FileCopyrightText: 2026 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package stun

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	errMissingProtocolScheme = errors.New("missing protocol scheme")
	errTooManyColonsAddr     = errors.New("too many colons in address")
)

func TestParseURL(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		testCases := []struct {
			rawURL            string
			expectedURLString string
			expectedScheme    SchemeType
			expectedSecure    bool
			expectedHost      string
			expectedPort      int
			expectedProto     ProtoType
		}{
			{"stun:google.de", "stun:google.de:3478", SchemeTypeSTUN, false, "google.de", 3478, ProtoTypeUDP},
			{"stun:google.de:1234", "stun:google.de:1234", SchemeTypeSTUN, false, "google.de", 1234, ProtoTypeUDP},
			{"stuns:google.de", "stuns:google.de:5349", SchemeTypeSTUNS, true, "google.de", 5349, ProtoTypeTCP},
			{"stun:[::1]:123", "stun:[::1]:123", SchemeTypeSTUN, false, "::1", 123, ProtoTypeUDP},
			{"turn:google.de", "turn:google.de:3478?transport=udp", SchemeTypeTURN, false, "google.de", 3478, ProtoTypeUDP},
			{"turns:google.de", "turns:google.de:5349?transport=tcp", SchemeTypeTURNS, true, "google.de", 5349, ProtoTypeTCP},
			{
				"turn:google.de?transport=udp",
				"turn:google.de:3478?transport=udp",
				SchemeTypeTURN, false, "google.de", 3478, ProtoTypeUDP,
			},
			{
				"turns:google.de?transport=tcp",
				"turns:google.de:5349?transport=tcp",
				SchemeTypeTURNS, true, "google.de", 5349, ProtoTypeTCP,
			},
		}

		for i, testCase := range testCases {
			url, err := ParseURI(testCase.rawURL)
			assert.Nil(t, err, "testCase: %d %v", i, testCase)
			if err != nil {
				return
			}

			assert.Equal(t, testCase.expectedScheme, url.Scheme, "testCase: %d %v", i, testCase)
			assert.Equal(t, testCase.expectedURLString, url.String(), "testCase: %d %v", i, testCase)
			assert.Equal(t, testCase.expectedSecure, url.IsSecure(), "testCase: %d %v", i, testCase)
			assert.Equal(t, testCase.expectedHost, url.Host, "testCase: %d %v", i, testCase)
			assert.Equal(t, testCase.expectedPort, url.Port, "testCase: %d %v", i, testCase)
			assert.Equal(t, testCase.expectedProto, url.Proto, "testCase: %d %v", i, testCase)
		}
	})

	t.Run("Failure", func(t *testing.T) {
		testCases := []struct {
			rawURL      string
			expectedErr error
		}{
			{"", ErrSchemeType},
			{":::", errMissingProtocolScheme},
			{"stun:[::1]:123:", errTooManyColonsAddr},
			{"stun:[::1]:123a", ErrPort},
			{"google.de", ErrSchemeType},
			{"stun:", ErrHost},
			{"stun:google.de:abc", ErrPort},
			{"stun:google.de?transport=udp", ErrSTUNQuery},
			{"stuns:google.de?transport=udp", ErrSTUNQuery},
			{"turn:google.de?trans=udp", ErrInvalidQuery},
			{"turns:google.de?trans=udp", ErrInvalidQuery},
			{"turns:google.de?transport=udp&another=1", ErrInvalidQuery},
			{"turn:google.de?transport=ip", ErrProtoType},
		}

		for i, testCase := range testCases {
			_, err := ParseURI(testCase.rawURL)
			var (
				urlError  *url.Error
				addrError *net.AddrError
			)
			switch {
			case errors.As(err, &urlError):
				err = urlError.Err
			case errors.As(err, &addrError):
				err = fmt.Errorf(addrError.Err) //nolint:err113, govet
			}
			assert.EqualError(t, err, testCase.expectedErr.Error(), "testCase: %d %v", i, testCase)
		}
	})
}
