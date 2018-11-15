package stun

import (
	"github.com/pkg/errors"
)

//ChannelData is struct including ChannelNumber and Data
type ChannelData struct {
	ChannelNumber uint16
	Length        uint16
	Data          []byte
}

//NewChannelData return ChannelData from packet
func NewChannelData(packet []byte) (*ChannelData, error) {
	cn, err := getChannelNumber(packet)
	if err != nil {
		return nil, err
	}

	return &ChannelData{
		ChannelNumber: cn,
		Length:        getChannelLength(packet),
		Data:          packet[4:],
	}, nil
}

//  0b01: ChannelData message (since the channel number is the first
//  field in the ChannelData message and channel numbers fall in the
//  range 0x4000 - 0x7FFF).
func getChannelNumber(header []byte) (uint16, error) {
	cn := enc.Uint16(header)
	if cn < 0x4000 || cn > 0x7FFF {
		return 0, errors.Errorf("ChannelNumber is out of range: %d", cn)
	}
	return cn, nil
}

func getChannelLength(header []byte) uint16 {
	return enc.Uint16(header[2:])
}
