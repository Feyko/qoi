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
	img                          *image.NRGBA
	Header                       Header
	window                       [windowLength]pixel
	previousPixel, currentPixel  pixel
	diffR, diffG, diffB, diffA   int8
	minX, maxX, minY, maxY, x, y int
	done                         bool
}

func NewEncoder(out io.Writer, img image.Image) Encoder {
	if !isImageNRGBA(img) {
		img = convertImageToNRGBA(img)
	}
	return Encoder{out: bufio.NewWriter(out), img: img.(*image.NRGBA)}
}

func isImageNRGBA(img image.Image) bool {
	return img.ColorModel() == color.NRGBAModel
}

func convertImageToNRGBA(img image.Image) image.Image {
	bounds := img.Bounds()
	newImg := image.NewNRGBA(bounds)
	for y := 0; y < bounds.Max.Y; y++ {
		for x := 0; x < bounds.Max.X; x++ {
			newImg.Set(x, y, img.At(x, y))
		}
	}
	return newImg
}

func (enc Encoder) Encode() error {
	width := enc.img.Bounds().Size().X
	height := enc.img.Bounds().Size().Y
	header := Header{
		Magic:      QoiMagicBytes,
		Width:      uint32(width),
		Height:     uint32(height),
		Channels:   4,
		Colorspace: 1,
	}
	enc.Header = header
	err := enc.encodeHeader()
	if err != nil {
		return fmt.Errorf("could not encode the Header: %w", err)
	}
	err = enc.encodeBody()
	if err != nil {
		return fmt.Errorf("could not encode the body: %w", err)
	}
	return nil
}

func (enc *Encoder) encodeHeader() error {
	return enc.Header.write(enc.out)
}

func (enc *Encoder) encodeBody() error {
	enc.currentPixel = newPixel(pixelBytes{0, 0, 0, 255})

	enc.setupBounds()
	enc.setupPosition()

	enc.advancePixel()
	for !enc.done {
		err := enc.dispatchOP()
		if err != nil {
			return err
		}
	}
	_, err := enc.out.Write([]byte{0, 0, 0, 0, 0, 0, 0, 1})
	if err != nil {
		return fmt.Errorf("could not write the end padding: %w", err)
	}
	err = enc.out.Flush()
	if err != nil {
		return fmt.Errorf("could not flush to disk: %w", err)
	}
	return nil
}

func (enc *Encoder) setupBounds() {
	enc.minX = enc.img.Bounds().Min.X
	enc.maxX = enc.img.Bounds().Max.X - 1 // 'Max' is the size of the image, not the maximum index we can use
	enc.minY = enc.img.Bounds().Min.Y
	enc.maxY = enc.img.Bounds().Max.Y - 1
}

func (enc *Encoder) setupPosition() {
	enc.x = enc.minX - 1 // Initialise one step back for the first update to land on the first pixel
	enc.y = enc.minY
}

func (enc *Encoder) advancePixel() {
	enc.updatePosition()
	pix := enc.img.At(enc.x, enc.y).(color.NRGBA)
	enc.previousPixel = enc.currentPixel
	enc.currentPixel = newPixel(pixelBytes{pix.R, pix.G, pix.B, pix.A})
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
	err := enc.out.WriteByte(QOI_OP_RGB)
	if err != nil {
		return fmt.Errorf("could not write the necessary data: %w", err)
	}
	_, err = enc.out.Write(enc.currentPixel.v[:3])
	if err != nil {
		return fmt.Errorf("could not write the necessary data: %w", err)
	}
	enc.advancePixel()
	return nil
}

func (enc *Encoder) op_RGBA() error {
	err := enc.out.WriteByte(QOI_OP_RGBA)
	if err != nil {
		return fmt.Errorf("could not write the necessary data: %w", err)
	}
	_, err = enc.out.Write(enc.currentPixel.v[:])
	if err != nil {
		return fmt.Errorf("could not write the necessary data: %w", err)
	}
	enc.advancePixel()
	return nil
}

func (enc *Encoder) op_INDEX() error {
	err := enc.out.WriteByte(QOI_OP_INDEX | enc.currentPixel.hash)
	if err != nil {
		return fmt.Errorf("could not write the necessary data: %w", err)
	}
	enc.advancePixel()
	return nil
}

func (enc *Encoder) op_DIFF() error {
	r := byte(enc.diffR+diffBias) << 4
	g := byte(enc.diffG+diffBias) << 2
	b := byte(enc.diffB + diffBias)
	err := enc.out.WriteByte(QOI_OP_DIFF | r | g | b)
	if err != nil {
		return fmt.Errorf("could not write the necessary data: %w", err)
	}
	enc.advancePixel()
	return nil
}

func (enc *Encoder) op_LUMA() error {
	directionRG := byte(enc.diffR - enc.diffG + lumaBias)
	directionBG := byte(enc.diffB - enc.diffG + lumaBias)
	err := enc.out.WriteByte(QOI_OP_LUMA | byte(enc.diffG+lumaGreenBias))
	if err != nil {
		return fmt.Errorf("could not write the necessary data: %w", err)
	}
	err = enc.out.WriteByte(directionRG<<4 | directionBG)
	if err != nil {
		return fmt.Errorf("could not write the necessary data: %w", err)
	}
	enc.advancePixel()
	return nil
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
	err := enc.out.WriteByte(QOI_OP_RUN | byte(count) - runBias)
	if err != nil {
		return fmt.Errorf("could not write the necessary data: %w", err)
	}
	return nil
}
