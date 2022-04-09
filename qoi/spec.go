package qoi

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	quoi_OP_RGB   byte = 0b11111110
	quoi_OP_RGBA  byte = 0b11111111
	quoi_OP_INDEX byte = 0b00
	quoi_OP_DIFF  byte = 0b01
	quoi_OP_LUMA  byte = 0b10
	quoi_OP_RUN   byte = 0b11

	quoi_2B_MASK byte = 0b11
)

func getOP(b byte) byte {
	masked := b & quoi_2B_MASK
	switch masked {
	case quoi_OP_INDEX, quoi_OP_DIFF, quoi_OP_LUMA, quoi_OP_RUN:
		return masked
	default:
		return b
	}
}

const headerLength = 4 + 4 + 4 + 1 + 1

type headerBytes [headerLength]byte

const qoiMagic = "qoif"

type Header struct {
	magic      string
	width      int
	height     int
	channels   byte
	colorspace byte
}

func interpretHeaderBytes(headerBytes headerBytes) (Header, error) {
	magic := headerBytes[:4]
	if string(magic) != qoiMagic {
		return Header{}, fmt.Errorf("invalid magic '%v'", magic)
	}
	width := int(binary.BigEndian.Uint32(headerBytes[4:]))
	length := int(binary.BigEndian.Uint32(headerBytes[8:]))

	channels := headerBytes[9]
	colorspace := headerBytes[10]
	return Header{string(magic), width, length, channels, colorspace}, nil
}

func (h Header) write(w io.Writer) error {
	return binary.Write(w, binary.BigEndian, h)
}
