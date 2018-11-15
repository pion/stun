package stun

import (
	"github.com/pkg/errors"
)

// PacketType is whether STUN or ChannelData
type PacketType int

// PacketTypes
const (
	PacketTypeSTUN        PacketType = iota
	PacketTypeChannelData PacketType = iota
)

// GetPacketType returns PacketType(whether STUN or ChannelData)
func GetPacketType(packet []byte) (PacketType, error) {
	if len(packet) < 2 {
		return 0, errors.Errorf("Packet is too short to determine type: %d", len(packet))
	}

	if verifyStunHeaderMostSignificant2Bits(packet) {
		return PacketTypeSTUN, nil
	} else if _, err := getChannelNumber(packet); err == nil {
		return PacketTypeChannelData, nil
	}

	return 0, errors.Errorf("%08b %08b", packet[0], packet[1])
}
