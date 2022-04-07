package qoi

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
)

const headerLength = 4 + 4 + 4 + 1 + 1

const qoiMagic = "qoif"

type Decoder struct {
	data        io.Reader
	headerBytes []byte
	config      image.Config
}

func (d *Decoder) decodeHeader() error {
	err := d.readHeader()
	if err != nil {
		return fmt.Errorf("could not read header: %w", err)
	}
	err = d.interpretHeaderBytes()
	if err != nil {
		return fmt.Errorf("could not interpret the header: %w", err)
	}
	return nil
}

func (d *Decoder) readHeader() error {
	headerBytes := make([]byte, headerLength)
	_, err := io.ReadAtLeast(d.data, headerBytes, headerLength)
	if err != nil {
		return errors.New("data is too short")
	}
	d.headerBytes = headerBytes
	return nil
}

func (d *Decoder) interpretHeaderBytes() error {
	magic := d.headerBytes[:4]
	if string(magic) != qoiMagic {
		return fmt.Errorf("invalid magic '%v'", magic)
	}
	width := int(binary.BigEndian.Uint32(d.headerBytes[4:]))
	length := int(binary.BigEndian.Uint32(d.headerBytes[8:]))

	d.config = image.Config{
		ColorModel: color.NRGBAModel,
		Width:      width,
		Height:     length,
	}
	return nil
}

// Decode reads a QOI image from r and returns it as an image.Image.
func Decode(r io.Reader) (image.Image, error) {
	decoder := Decoder{data: r}
	err := decoder.decodeHeader()
	if err != nil {
		return nil, fmt.Errorf("could not decode the header: %w", err)
	}
	return nil, err
}

//DecodeConfig returns the color model and dimensions of a QOI image without decoding the entire image.
func DecodeConfig(r io.Reader) (image.Config, error) {
	decoder := Decoder{data: r}
	err := decoder.decodeHeader()
	if err != nil {
		return image.Config{}, fmt.Errorf("could not decode the header: %w", err)
	}
	return decoder.config, nil
}

func init() {
	image.RegisterFormat("qoi", "qoif", Decode, DecodeConfig)
}
