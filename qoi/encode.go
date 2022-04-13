package qoi

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"io"
)

//Encode writes the Image m to w in PNG format. Any Image may be encoded, but images that are not image.NRGBA might be encoded lossily.
func Encode(w io.Writer, m image.Image) error {
	return NewEncoder(w, m).Encode()
}

type Encoder struct {
	out                          *bufio.Writer
	img                          image.Image
	header                       Header
	window                       [windowLength]pixel
	previousPixel, currentPixel  pixel
	diffR, diffG, diffB, diffA   int8
	minX, maxX, minY, maxY, x, y int
	done                         bool
}

func NewEncoder(out io.Writer, img image.Image) Encoder {
	return Encoder{out: bufio.NewWriter(out), img: img}
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
	return enc.encodeBody()
}

func (enc *Encoder) encodeHeader() error {
	return enc.header.write(enc.out)
}

func (enc *Encoder) encodeBody() error {
	enc.currentPixel = newPixel([4]byte{0, 0, 0, 255})
	enc.minX = enc.img.Bounds().Min.X
	enc.maxX = enc.img.Bounds().Max.X
	enc.minY = enc.img.Bounds().Min.Y
	enc.maxY = enc.img.Bounds().Max.Y
	enc.x = enc.minX
	enc.y = enc.minY
	enc.advancePixel()
	for !enc.done {
		err := enc.dispatchOP()
		if err != nil {
			return err
		}
		//enc.cacheCurrentPixel()
	}
	_, err := enc.out.Write([]byte{0, 0, 0, 0, 0, 0, 0, 1})
	return err
}

func (enc *Encoder) advancePixel() {
	c := color.NRGBAModel.Convert(enc.img.At(enc.x, enc.y))
	c_r, c_g, c_b, c_a := c.RGBA()
	enc.previousPixel = enc.currentPixel
	enc.currentPixel = newPixel([4]byte{byte(c_r >> 8), byte(c_g >> 8), byte(c_b >> 8), byte(c_a >> 8)})
	enc.updatePosition()
}

func (enc *Encoder) updatePosition() {
	if enc.x == enc.maxX {
		enc.y++
		enc.x = enc.minX
	} else {
		enc.x++
	}
	if enc.x == enc.maxX && enc.y == enc.maxY {
		enc.done = true
		return
	}
	return
}

func (enc *Encoder) cacheCurrentPixel() {
	enc.window[enc.previousPixel.Hash()] = enc.previousPixel // We do not check for equality as copying a 4B array is faster than checking
}

func (enc *Encoder) dispatchOP() error {
	if enc.currentPixel == enc.previousPixel {
		return enc.op_RUN()
	}
	if enc.window[enc.currentPixel.hash] == enc.currentPixel {
		return enc.op_INDEX()
	}
	enc.cacheCurrentPixel()
	enc.calculateDiff()
	if enc.diffA != 0 {
		return enc.op_RGBA()
	}
	if enc.isWithinDIFF() {
		return enc.op_DIFF()
	}
	if enc.isWithinLUMA() {
		return enc.op_LUMA()
	}

	return enc.op_RGBA()
}

func (enc *Encoder) calculateDiff() {
	enc.diffR, enc.diffG, enc.diffB, enc.diffA = enc.currentPixel.Minus(enc.previousPixel)
}

func (enc *Encoder) isWithinDIFF() bool {
	return isValueWithinDIFF(enc.diffR) && isValueWithinDIFF(enc.diffG) && isValueWithinDIFF(enc.diffB)
}

func isValueWithinDIFF(v int8) bool {
	return v > -3 && v < 2
}

func (enc *Encoder) isWithinLUMA() bool {
	return isValueWithinLUMA(enc.diffR) && isValueWithinLUMAGreen(enc.diffG) && isValueWithinLUMA(enc.diffB)
}

func isValueWithinLUMA(v int8) bool {
	return v > -9 && v < 8
}

func isValueWithinLUMAGreen(v int8) bool {
	return v > -33 && v < 32
}

func (enc *Encoder) op_RGB() error {
	err := enc.out.WriteByte(quoi_OP_RGB)
	if err != nil {
		return err
	}
	_, err = enc.out.Write(enc.currentPixel.v[:3])
	enc.advancePixel()
	return err
}

func (enc *Encoder) op_RGBA() error {
	err := enc.out.WriteByte(quoi_OP_RGBA)
	if err != nil {
		return err
	}
	_, err = enc.out.Write(enc.currentPixel.v[:])
	enc.advancePixel()
	return err
}

func (enc *Encoder) op_INDEX() error {
	err := enc.out.WriteByte(quoi_OP_INDEX | enc.currentPixel.hash)
	enc.advancePixel()
	return err
}

func (enc *Encoder) op_DIFF() error {
	r := byte(enc.diffR+2) << 4
	g := byte(enc.diffG+2) << 2
	b := byte(enc.diffB + 2)
	err := enc.out.WriteByte(quoi_OP_DIFF | r | g | b)
	enc.advancePixel()
	return err
}

func (enc *Encoder) op_LUMA() error {
	directionRG := byte(enc.diffR - enc.diffG)
	directionBG := byte(enc.diffB - enc.diffG)
	err := enc.out.WriteByte(quoi_OP_LUMA | byte(enc.diffG))
	if err != nil {
		return err
	}
	err = enc.out.WriteByte(directionRG<<4 | directionBG)
	enc.advancePixel()
	return err
}

func (enc *Encoder) op_RUN() error {
	c := 0
	for enc.currentPixel == enc.previousPixel {
		c++
		if c == 62 {
			break
		}
		enc.advancePixel()
		if enc.done {
			c-- // Went one too far, the last advance did not change the pixel
			break
		}
	}
	return enc.out.WriteByte(quoi_OP_RUN | byte(c) - 1)
}
