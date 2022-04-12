package qoi

import (
	"errors"
	"fmt"
	"image"
	"io"
)

//Encode writes the Image m to w in PNG format. Any Image may be encoded, but images that are not image.NRGBA might be encoded lossily.
func Encode(w io.Writer, m image.Image) error {
	return NewEncoder(w, m).Encode()
}

type Encoder struct {
	out          io.Writer
	img          image.Image
	header       Header
	window       [windowLength]pixel
	currentPixel pixel
}

func NewEncoder(out io.Writer, img image.Image) Encoder {
	return Encoder{out: out, img: img}
}

func (enc Encoder) Encode() error {
	width := enc.img.Bounds().Size().X
	height := enc.img.Bounds().Size().Y
	header := Header{
		magic:      qoiMagicBytes,
		width:      uint32(width),
		height:     uint32(height),
		channels:   4,
		colorspace: 1,
	}
	enc.header = header
	err := enc.encodeHeader()
	if err != nil {
		return fmt.Errorf("could not encode the header: %w", err)
	}

	return nil
}

func (enc *Encoder) encodeHeader() error {
	return enc.header.write(enc.out)
}

func (enc *Encoder) encodeBody() (image.Image, error) {
	enc.currentPixel = newPixel([4]byte{0, 0, 0, 255})
	img := image.NewNRGBA(image.Rect(0, 0, int(enc.header.width), int(enc.header.height)))
	enc.img = img
	enc.imgPixelBytes = img.Pix
	for len(enc.imgPixelBytes) > 0 {

		b, err := enc.data.ReadByte()
		if err == io.EOF {
			return enc.img, nil
		}
		if err != nil {
			return nil, err
		}
		enc.currentByte = b
		err = enc.dispatchOP()
		if err != nil {
			return nil, err
		}

		enc.cacheCurrentPixel()
	}
	return enc.img, nil
}

func (enc *Encoder) cacheCurrentPixel() {
	enc.window[enc.currentPixel.Hash()] = enc.currentPixel // We do not check for equality as copying a 4B array is faster than checking
}

func (enc *Encoder) dispatchOP() error {
	op := getOP(enc.currentByte)
	switch op {
	case quoi_OP_RGB:
		return enc.op_RGB()
	case quoi_OP_RGBA:
		return enc.op_RGBA()
	case quoi_OP_INDEX:
		return enc.op_INDEX()
	case quoi_OP_DIFF:
		return enc.op_DIFF()
	case quoi_OP_LUMA:
		return enc.op_LUMA()
	case quoi_OP_RUN:
		return enc.op_RUN()
	default:
		panic("Unknown OP")
	}
}

func (enc *Encoder) op_RGB() error {
	_, err := io.ReadFull(enc.data, enc.currentPixel.v[:3])
	enc.currentPixel.calculateHash()
	enc.writeCurrentPixel()
	return err
}

func (enc *Encoder) op_RGBA() error {
	_, err := io.ReadFull(enc.data, enc.currentPixel.v[:])
	enc.currentPixel.calculateHash()
	enc.writeCurrentPixel()
	return err
}

func (enc *Encoder) op_INDEX() error {
	index := enc.currentByte & 0b00111111
	enc.currentPixel = enc.window[index]
	enc.writeCurrentPixel()
	return nil
}

func (enc *Encoder) op_DIFF() error {
	r, g, b := getDIFFValues(enc.currentByte)
	enc.currentPixel.Add(r, g, b)
	enc.writeCurrentPixel()
	return nil
}

func getDIFFValues(diff byte) (byte, byte, byte) {
	return diff&0b00110000>>4 - 2, diff&0b00001100>>2 - 2, diff&0b00000011 - 2
}

func (enc *Encoder) op_LUMA() error {
	b1 := enc.currentByte
	b2, err := enc.data.ReadByte()
	if err != nil {
		return err
	}
	r, g, b := getLUMAValues(b1, b2)
	enc.currentPixel.Add(r, g, b)
	enc.writeCurrentPixel()
	return nil
}

func getLUMAValues(b1, b2 byte) (byte, byte, byte) {
	diffGreen := b1&0b00111111 - 32
	diffRed := diffGreen + (b2 & 0b11110000 >> 4) - 8
	diffBlue := diffGreen + (b2 & 0b00001111) - 8
	return diffRed, diffGreen, diffBlue
}

func (enc *Encoder) op_RUN() error {
	run := (enc.currentByte & 0b00111111) + 1
	if run > 62 {
		return errors.New("illegal RUN value")
	}
	enc.repeat(run)
	return nil
}

func (enc *Encoder) repeat(n byte) {
	for ; n > 0; n-- {
		enc.writeCurrentPixel()
	}
}

func (enc *Encoder) writeCurrentPixel() {
	copy(enc.imgPixelBytes[:4], enc.currentPixel.v[:])
	enc.imgPixelBytes = enc.imgPixelBytes[4:]
}
