package qoi

import (
	"bufio"
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
	return decoder.Decode()
}

//DecodeConfig returns the color model and dimensions of a QOI image without decoding the entire image.
func DecodeConfig(r io.Reader) (image.Config, error) {
	decoder := NewDecoder(r)
	err := decoder.DecodeHeader()
	if err != nil {
		return image.Config{}, fmt.Errorf("could not decode the Header: %w", err)
	}
	return image.Config{
		ColorModel: color.NRGBAModel,
		Width:      int(decoder.Header.Width),
		Height:     int(decoder.Header.Height),
	}, nil
}

type Decoder struct {
	data          *bufio.Reader
	headerBytes   headerBytes
	Header        Header
	decodedHeader bool
	pixelWindow   [windowLength]pixel
	currentPixel  pixel
	currentByte   byte
	img           image.Image
	imgPixelBytes []byte
}

func NewDecoder(data io.Reader) Decoder {
	return Decoder{data: bufio.NewReader(data)}
}

func (d *Decoder) Decode() (image.Image, error) {
	if !d.decodedHeader {
		err := d.DecodeHeader()
		if err != nil {
			return nil, fmt.Errorf("could not decode the Header: %w", err)
		}
	}
	img, err := d.decodeBody()
	if err != nil {
		return nil, fmt.Errorf("could not decode the image body: %w", err)
	}
	return img, nil
}

func (d *Decoder) DecodeHeader() error {
	err := d.readHeader()
	if err != nil {
		return fmt.Errorf("could not read Header: %w", err)
	}
	header, err := interpretHeaderBytes(d.headerBytes)
	d.Header = header
	if err != nil {
		return fmt.Errorf("invalid Header: %w", err)
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
	d.currentPixel = newPixel(pixelBytes{0, 0, 0, 255})
	img := image.NewNRGBA(image.Rect(0, 0, int(d.Header.Width), int(d.Header.Height)))
	d.img = img
	d.imgPixelBytes = img.Pix
	for len(d.imgPixelBytes) > 0 {

		b, err := d.data.ReadByte()
		if err == io.EOF {
			return d.img, nil
		}
		if err != nil {
			return nil, fmt.Errorf("could not read the necessary data: %w", err)
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
	d.pixelWindow[d.currentPixel.Hash()] = d.currentPixel // We do not check for equality as copying a 4B array is faster than checking
}

func (d *Decoder) dispatchOP() error {
	op := getOP(d.currentByte)
	switch op {
	case QOI_OP_RGB:
		return d.op_RGB()
	case QOI_OP_RGBA:
		return d.op_RGBA()
	case QOI_OP_INDEX:
		return d.op_INDEX()
	case QOI_OP_DIFF:
		return d.op_DIFF()
	case QOI_OP_LUMA:
		return d.op_LUMA()
	case QOI_OP_RUN:
		return d.op_RUN()
	default:
		panic("Unknown OP")
	}
}

func (d *Decoder) op_RGB() error {
	_, err := io.ReadFull(d.data, d.currentPixel.v[:3])
	if err != nil {
		return fmt.Errorf("could not read the necessary data: %w", err)
	}
	d.currentPixel.calculateHash()
	d.writeCurrentPixel()
	return nil
}

func (d *Decoder) op_RGBA() error {
	_, err := io.ReadFull(d.data, d.currentPixel.v[:])
	if err != nil {
		return fmt.Errorf("could not read the necessary data: %w", err)
	}
	d.currentPixel.calculateHash()
	d.writeCurrentPixel()
	return nil
}

func (d *Decoder) op_INDEX() error {
	index := d.currentByte & 0b00111111
	d.currentPixel = d.pixelWindow[index]
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
	return diff&0b00110000>>4 - diffBias, diff&0b00001100>>2 - diffBias, diff&0b00000011 - diffBias
}

func (d *Decoder) op_LUMA() error {
	b1 := d.currentByte
	b2, err := d.data.ReadByte()
	if err != nil {
		return fmt.Errorf("could not read the necessary data: %w", err)
	}
	r, g, b := getLUMAValues(b1, b2)
	d.currentPixel.Add(r, g, b)
	d.writeCurrentPixel()
	return nil
}

func getLUMAValues(b1, b2 byte) (byte, byte, byte) {
	diffGreen := b1&0b00111111 - lumaGreenBias
	diffRed := diffGreen + (b2 & 0b11110000 >> 4) - lumaBias
	diffBlue := diffGreen + (b2 & 0b00001111) - lumaBias
	return diffRed, diffGreen, diffBlue
}

func (d *Decoder) op_RUN() error {
	run := (d.currentByte & 0b00111111) + runBias
	if run > 62 {
		return errors.New("illegal RUN value (>62)")
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
	copy(d.imgPixelBytes[:4], d.currentPixel.v[:])
	d.imgPixelBytes = d.imgPixelBytes[4:]
}
