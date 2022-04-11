package qoi

import (
	"bufio"
	"errors"
	"fmt"
	"golang.org/x/exp/slices"
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
	return image.Config{
		ColorModel: color.NRGBAModel,
		Width:      int(decoder.header.width),
		Height:     int(decoder.header.height),
	}, nil
}

type Decoder struct {
	data          *bufio.Reader
	headerBytes   headerBytes
	header        Header
	pixelWindow   [windowLength]pixel
	currentPixel  pixel
	currentByte   byte
	img           image.Image
	imgPixelBytes []byte

	opMap map[byte]func() error
}

func NewDecoder(data io.Reader) Decoder {
	return Decoder{data: bufio.NewReader(data)}
}

func (d *Decoder) decodeHeader() error {
	err := d.readHeader()
	if err != nil {
		return fmt.Errorf("could notdecodeImage read header: %w", err)
	}
	header, err := interpretHeaderBytes(d.headerBytes)
	d.header = header
	if err != nil {
		return fmt.Errorf("could not interpret the header: %w", err)
	}
	return nil
}

func (d *Decoder) readHeader() error {
	_, err := io.ReadAtLeast(d.data, d.headerBytes[:], headerLength)
	if err != nil {
		return errors.New("data is too short")
	}
	return nil
}

func (d *Decoder) decodeBody() (image.Image, error) {
	d.fillOPMap()
	d.currentPixel = pixel{0, 0, 0, 255}
	img := image.NewNRGBA(image.Rect(0, 0, int(d.header.width), int(d.header.height)))
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

		d.cacheCurrentPixel()
	}
	return d.img, nil
}

func (d *Decoder) cacheCurrentPixel() {
	d.pixelWindow[d.currentPixel.Hash()] = slices.Clone(d.currentPixel)
}

func (d *Decoder) fillOPMap() {
	d.opMap = map[byte]func() error{
		quoi_OP_RGB:   d.op_RGB,
		quoi_OP_RGBA:  d.op_RGBA,
		quoi_OP_INDEX: d.op_INDEX,
		quoi_OP_DIFF:  d.op_DIFF,
		quoi_OP_LUMA:  d.op_LUMA,
		quoi_OP_RUN:   d.op_RUN,
	}
}

func (d *Decoder) dispatchOP() error {
	op := getOP(d.currentByte)
	return d.opMap[op]()
}

func (d *Decoder) op_RGB() error {
	_, err := io.ReadFull(d.data, d.currentPixel[:3])
	d.writeCurrentPixel()
	return err
}

func (d *Decoder) op_RGBA() error {
	_, err := io.ReadFull(d.data, d.currentPixel)
	d.writeCurrentPixel()
	return err
}

func (d *Decoder) op_INDEX() error {
	index := d.currentByte & 0b00111111
	copy(d.currentPixel, d.pixelWindow[index])
	d.writeCurrentPixel()
	return nil
}

func (d *Decoder) op_DIFF() error {
	r, g, b := getDIFFValues(d.currentByte)
	d.currentPixel.Add(r, g, b)
	d.writeCurrentPixel()
	return nil
}

func getDIFFValues(diff byte) (byte, byte, byte) {
	return diff&0b00110000>>4 - 2, diff&0b00001100>>2 - 2, diff&0b00000011 - 2
}

func (d *Decoder) op_LUMA() error {
	b1 := d.currentByte
	b2, err := d.data.ReadByte()
	if err != nil {
		return err
	}
	r, g, b := getLUMAValues(b1, b2)
	d.currentPixel.Add(r, g, b)
	d.writeCurrentPixel()
	return nil
}

func getLUMAValues(b1, b2 byte) (byte, byte, byte) {
	diffGreen := b1&0b00111111 - 32
	diffRed := diffGreen + (b2 & 0b11110000 >> 4) - 8
	diffBlue := diffGreen + (b2 & 0b00001111) - 8
	return diffRed, diffGreen, diffBlue
}

func (d *Decoder) op_RUN() error {
	run := (d.currentByte & 0b00111111) + 1
	if run > 62 {
		return errors.New("illegal RUN value")
	}
	d.repeat(run)
	return nil
}

func (d *Decoder) repeat(n byte) {
	for ; n > 0; n-- {
		d.writeCurrentPixel()
	}
}

func (d *Decoder) writeCurrentPixel() {
	copy(d.imgPixelBytes[:4], d.currentPixel)
	d.imgPixelBytes = d.imgPixelBytes[4:]
}
