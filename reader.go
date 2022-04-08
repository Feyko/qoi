package qoi

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
)

func init() {
	image.RegisterFormat("qoi", "qoif", Decode, DecodeConfig)
}

// Decode reads a QOI image from r and returns it as an image.Image.
func Decode(r io.Reader) (image.Image, error) {
	decoder := NewDecoder(r)
	err := decoder.decodeHeader()
	if err != nil {
		return nil, fmt.Errorf("could not decode the header: %w", err)
	}
	img, err := decoder.decodeBody()
	if err != nil {
		return nil, fmt.Errorf("could not decode the image body: %w", err)
	}
	return img, nil
}

//DecodeConfig returns the color model and dimensions of a QOI image without decoding the entire image.
func DecodeConfig(r io.Reader) (image.Config, error) {
	decoder := NewDecoder(r)
	err := decoder.decodeHeader()
	if err != nil {
		return image.Config{}, fmt.Errorf("could not decode the header: %w", err)
	}
	return decoder.config, nil
}

const headerLength = 4 + 4 + 4 + 1 + 1

const qoiMagic = "qoif"

func (p pixel) Hash() byte {
	return (p.R()*3 + p.G()*5 + p.B()*7 + p.A()*11) % 64
}

type pixel []byte

func (p pixel) R() byte {
	return p[0]
}

func (p pixel) G() byte {
	return p[1]
}

func (p pixel) B() byte {
	return p[2]
}

func (p pixel) A() byte {
	return p[3]
}

func (p pixel) Components() (byte, byte, byte, byte) {
	return p.R(), p.B(), p.R(), p.A()
}

func (p pixel) Add(r, g, b byte) {
	p[0] += r
	p[1] += g
	p[2] += b
}

type opCode byte

const (
	quoi_OP_RGB   byte = 0b11111110
	quoi_OP_RGBA  byte = 0b11111111
	quoi_OP_INDEX byte = 0b00
	quoi_OP_DIFF  byte = 0b01
	quoi_OP_LUMA  byte = 0b10
	quoi_OP_RUN   byte = 0b11

	quoi_2B_MASK = 0b11
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

type Decoder struct {
	data          *bufio.Reader
	headerBytes   []byte
	config        image.Config
	pixelWindow   [64]pixel
	currentPixel  pixel
	currentByte   byte
	img           image.Image
	imgPixelBytes []byte
}

func NewDecoder(data io.Reader) Decoder {
	return Decoder{data: bufio.NewReader(data)}
}

func (d *Decoder) decodeHeader() error {
	err := d.readHeader()
	if err != nil {
		return fmt.Errorf("could notdecodeImage read header: %w", err)
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

func (d *Decoder) decodeBody() (image.Image, error) {
	d.currentPixel = pixel{0, 0, 0, 255}
	img := image.NewNRGBA(image.Rect(0, 0, d.config.Width, d.config.Height))
	d.img = img
	d.imgPixelBytes = img.Pix
	for len(d.imgPixelBytes) > 0 {
		b, err := d.data.ReadByte()
		if err == io.EOF {
			return d.img, nil
		}
		if err != nil {
			return nil, err
		}
		d.currentByte = b
		err = d.dispatchOP()
		if err != nil {
			return nil, err
		}
		copy(d.imgPixelBytes[:4], d.currentPixel)
		d.imgPixelBytes = d.imgPixelBytes[4:]
		d.pixelWindow[d.currentPixel.Hash()] = d.currentPixel
	}
	return d.img, nil
}

func (d *Decoder) dispatchOP() error {
	dispatcherMap := map[byte]func() error{
		quoi_OP_RGB:   d.op_RGB,
		quoi_OP_RGBA:  d.op_RGBA,
		quoi_OP_INDEX: d.op_INDEX,
		quoi_OP_DIFF:  d.op_DIFF,
		quoi_OP_LUMA:  d.op_LUMA,
		quoi_OP_RUN:   d.op_RUN,
	}

	op := getOP(d.currentByte)
	return dispatcherMap[op]()
}

func (d *Decoder) op_RGB() error {
	_, err := io.ReadFull(d.data, d.currentPixel[3:])
	return err
}

func (d *Decoder) op_RGBA() error {
	_, err := io.ReadFull(d.data, d.currentPixel)
	return err
}

func (d *Decoder) op_INDEX() error {
	index := d.currentPixel.Hash()
	d.currentPixel = d.pixelWindow[index]
	return nil
}

func (d *Decoder) op_DIFF() error {
	r, g, b := getDIFFValues(d.currentByte)
	d.currentPixel.Add(r, g, b)
	return nil
}

func getDIFFValues(diff byte) (byte, byte, byte) {
	return diff&0b00110000 - 2, diff&0b00001100 - 2, diff&0b00000011 - 2
}

func (d *Decoder) op_LUMA() error {
	b1 := d.currentByte
	b2, err := d.data.ReadByte()
	if err != nil {
		return err
	}
	r, g, b := getLUMAValues(b1, b2)
	d.currentPixel.Add(r, g, b)
	return nil
}

func getLUMAValues(b1, b2 byte) (byte, byte, byte) {
	diffGreen := b1&0b00111111 - 32
	diffRed := diffGreen + (b2 & 0b11110000) - 8
	diffBlue := diffGreen + (b2 & 0b00001111) - 8
	return diffRed, diffGreen, diffBlue
}

func (d *Decoder) op_RUN() error {
	run := d.currentByte & 0b00111111
	d.repeat(run)
	return nil
}

func (d *Decoder) repeat(n byte) {

}

func (d *Decoder) op_NONE() {

}
