package stun

import (
	"encoding/binary"

	"github.com/pkg/errors"
)

type ChannelData struct {
	ChannelNumber uint16
	Data          []byte
}

func NewChannelData(packet []byte) (*ChannelData, error) {
	cn, err := getChannelNumber(packet)
	if err != nil {
		return nil, err
	}

	return &ChannelData{
		ChannelNumber: cn,
		Data:          packet[2:],
	}, nil
}

//  0b01: ChannelData message (since the channel number is the first
//  field in the ChannelData message and channel numbers fall in the
//  range 0x4000 - 0x7FFF).
func getChannelNumber(header []byte) (uint16, error) {
	cn := binary.BigEndian.Uint16(header)
	if cn < 0x4000 || cn > 0x7FFF {
		return 0, errors.Errorf("ChannelNumber is out of range: %d", cn)
	}
	return cn, nil
}
