package qoi

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	QOI_OP_RGB   byte = 0b11111110
	QOI_OP_RGBA  byte = 0b11111111
	QOI_OP_INDEX byte = 0b00000000
	QOI_OP_DIFF  byte = 0b01000000
	QOI_OP_LUMA  byte = 0b10000000
	QOI_OP_RUN   byte = 0b11000000

	QOI_2B_MASK byte = 0b11000000
)

func getOP(b byte) byte {
	switch b {
	case QOI_OP_RGB, QOI_OP_RGBA:
		return b
	}
	masked := b & QOI_2B_MASK
	switch masked {
	case QOI_OP_INDEX, QOI_OP_DIFF, QOI_OP_LUMA, QOI_OP_RUN:
		return masked
	default:
		panic("Unknown OP")
	}
}

const (
	headerLength  = 4 + 4 + 4 + 1 + 1
	windowLength  = 64
	diffBias      = 2
	lumaBias      = 8
	lumaGreenBias = 32
	runBias       = 1
)

type headerBytes [headerLength]byte

var QoiMagicBytes = [4]byte{byte('q'), byte('o'), byte('i'), byte('f')}

type Header struct {
	Magic      [4]byte
	Width      uint32
	Height     uint32
	Channels   byte
	Colorspace byte
}

func interpretHeaderBytes(headerBytes headerBytes) (Header, error) {
	var magic [4]byte
	copy(magic[:], headerBytes[:4])
	if magic != QoiMagicBytes {
		return Header{}, fmt.Errorf("invalid Magic %v, expected %v", magic, QoiMagicBytes)
	}
	width := binary.BigEndian.Uint32(headerBytes[4:])
	length := binary.BigEndian.Uint32(headerBytes[8:])

	channels := headerBytes[9]
	colorspace := headerBytes[10]
	return Header{magic, width, length, channels, colorspace}, nil
}

func (h Header) write(w io.Writer) error {
	return binary.Write(w, binary.BigEndian, h)
}

func isValueWithinDIFFSpec(v int8) bool {
	return v > -3 && v < 2
}

func isValueWithinLUMASpec(v int8) bool {
	return v > -9 && v < 8
}

func isGreenValueWithinLUMASpec(v int8) bool {
	return v > -33 && v < 32
}
