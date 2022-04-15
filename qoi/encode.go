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
	enc.maxX = enc.img.Bounds().Max.X - 1
	enc.minY = enc.img.Bounds().Min.Y
	enc.maxY = enc.img.Bounds().Max.Y - 1
	enc.x = enc.minX - 1
	enc.y = enc.minY
	enc.advancePixel()
	for !enc.done {
		err := enc.dispatchOP()
		if err != nil {
			return err
		}
	}
	_, err := enc.out.Write([]byte{0, 0, 0, 0, 0, 0, 0, 1})
	if err != nil {
		return err
	}
	return enc.out.Flush()
}

func (enc *Encoder) advancePixel() {
	enc.updatePosition()
	pix := color.NRGBAModel.Convert(enc.img.At(enc.x, enc.y))
	r, g, b, a := pix.RGBA()
	enc.previousPixel = enc.currentPixel
	enc.currentPixel = newUnpremultipliedPixel(r, g, b, a)
}

func (enc *Encoder) updatePosition() {
	if enc.x == enc.maxX && enc.y == enc.maxY {
		enc.done = true
		return
	}
	if enc.x == enc.maxX {
		enc.y++
		enc.x = enc.minX
	} else {
		enc.x++
	}
	return
}

func (enc *Encoder) cacheCurrentPixel() {
	enc.window[enc.currentPixel.Hash()] = enc.currentPixel // We do not check for equality as copying a 4B array is faster than checking
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
	if enc.isCurrentPixelWithinDIFFSpec() {
		return enc.op_DIFF()
	}
	if enc.isCurrentPixelWithinLUMASpec() {
		return enc.op_LUMA()
	}

	return enc.op_RGB()
}

func (enc *Encoder) calculateDiff() {
	enc.diffR, enc.diffG, enc.diffB, enc.diffA = enc.currentPixel.Minus(enc.previousPixel)
}

func (enc *Encoder) isCurrentPixelWithinDIFFSpec() bool {
	return isValueWithinDIFFSpec(enc.diffR) && isValueWithinDIFFSpec(enc.diffG) && isValueWithinDIFFSpec(enc.diffB)
}

func (enc *Encoder) isCurrentPixelWithinLUMASpec() bool {
	return isValueWithinLUMASpec(enc.diffR-enc.diffG) && isGreenValueWithinLUMASpec(enc.diffG) && isValueWithinLUMASpec(enc.diffB-enc.diffG)
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
	r := byte(enc.diffR+diffBias) << 4
	g := byte(enc.diffG+diffBias) << 2
	b := byte(enc.diffB + diffBias)
	err := enc.out.WriteByte(quoi_OP_DIFF | r | g | b)
	enc.advancePixel()
	return err
}

func (enc *Encoder) op_LUMA() error {
	directionRG := byte(enc.diffR - enc.diffG + lumaBias)
	directionBG := byte(enc.diffB - enc.diffG + lumaBias)
	err := enc.out.WriteByte(quoi_OP_LUMA | byte(enc.diffG+lumaGreenBias))
	if err != nil {
		return err
	}
	err = enc.out.WriteByte(directionRG<<4 | directionBG)
	enc.advancePixel()
	return err
}

func (enc *Encoder) op_RUN() error {
	count := 1
	enc.advancePixel()
	for enc.currentPixel == enc.previousPixel && !enc.done {
		count++
		enc.advancePixel()
		if count == 62 {
			break
		}
	}
	return enc.out.WriteByte(quoi_OP_RUN | byte(count) - runBias)
}
